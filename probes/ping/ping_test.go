// Copyright 2017-2022 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ping

import (
	"fmt"
	"net"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/ping/proto"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"google.golang.org/protobuf/proto"
)

func peerToIP(peer net.Addr) string {
	switch peer := peer.(type) {
	case *net.UDPAddr:
		return peer.IP.String()
	case *net.IPAddr:
		return peer.IP.String()
	}
	return ""
}

// replyPkt creates an ECHO reply packet from the ECHO request packet.
func replyPkt(pkt []byte, ipVersion int) []byte {
	protocol := protocolICMP
	var typ icmp.Type
	typ = ipv4.ICMPTypeEchoReply
	if ipVersion == 6 {
		protocol = protocolIPv6ICMP
		typ = ipv6.ICMPTypeEchoReply
	}
	m, _ := icmp.ParseMessage(protocol, pkt)
	m.Type = typ
	b, _ := m.Marshal(nil)
	return b
}

// testICMPConn implements the icmpConn interface.
// It implements the following packets pipeline:
//
//	write(packet) --> sentPackets channel -> read() -> packet
//
// It has a per-target channel that receives packets through the "write" call.
// "read" call fetches packets from that channel and returns them to the
// caller.
type testICMPConn struct {
	sentPackets map[string](chan []byte)
	c           *configpb.ProbeConf
	ipVersion   int

	flipLastByte   bool
	flipLastByteMu sync.Mutex
}

func newTestICMPConn(opts *options.Options, targets []endpoint.Endpoint) *testICMPConn {
	tic := &testICMPConn{
		c:           opts.ProbeConf.(*configpb.ProbeConf),
		ipVersion:   opts.IPVersion,
		sentPackets: make(map[string](chan []byte)),
	}
	for _, target := range targets {
		tic.sentPackets[target.Name] = make(chan []byte, tic.c.GetPacketsPerProbe())
	}
	return tic
}

func (tic *testICMPConn) setFlipLastByte() {
	tic.flipLastByteMu.Lock()
	defer tic.flipLastByteMu.Unlock()
	tic.flipLastByte = true
}

func (tic *testICMPConn) read(buf []byte) (int, net.Addr, time.Time, error) {
	// We create per-target select cases, with each target's select-case
	// pointing to that target's sentPackets channel.
	var cases []reflect.SelectCase
	var targets []string
	for t, ch := range tic.sentPackets {
		cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)})
		targets = append(targets, t)
	}

	// Select over the select cases.
	chosen, value, ok := reflect.Select(cases)
	if !ok {
		return 0, nil, time.Now(), fmt.Errorf("nothing to read")
	}

	pkt := value.Bytes()

	// Since we are echoing the packets, copy the received packet into the
	// provided buffer.
	respPkt := replyPkt(pkt, tic.ipVersion)
	tic.flipLastByteMu.Lock()
	if tic.flipLastByte {
		lastByte := ^respPkt[len(respPkt)-1]
		respPkt = append(respPkt[:len(respPkt)-1], lastByte)
	}
	tic.flipLastByteMu.Unlock()

	copy(buf[0:len(pkt)], respPkt)
	peerIP := net.ParseIP(targets[chosen])

	var peer net.Addr
	peer = &net.IPAddr{IP: peerIP}
	if tic.c.GetUseDatagramSocket() {
		peer = &net.UDPAddr{IP: peerIP}
	}
	return len(pkt), peer, time.Now(), nil
}

// write simply queues packets into the sentPackets channel. These packets are
// retrieved by the "read" call.
func (tic *testICMPConn) write(in []byte, peer net.Addr) (int, error) {
	target := peerToIP(peer)

	// Copy incoming bytes slice and store in the internal channel for use
	// during the read call.
	b := make([]byte, len(in))
	copy(b, in)
	tic.sentPackets[target] <- b

	return len(b), nil
}

func (tic *testICMPConn) setReadDeadline(deadline time.Time) {
}

func (tic *testICMPConn) close() {
}

// Sends packets and verifies
func sendAndCheckPackets(p *Probe, t *testing.T) {
	tic := newTestICMPConn(p.opts, p.targets)
	p.conn = tic
	trackerChan := make(chan bool, int(p.c.GetPacketsPerProbe())*len(p.targets))
	runID := p.newRunID()
	p.sendPackets(runID, trackerChan)

	protocol := protocolICMP
	var expectedMsgType icmp.Type
	expectedMsgType = ipv4.ICMPTypeEcho
	if p.opts.IPVersion == 6 {
		protocol = protocolIPv6ICMP
		expectedMsgType = ipv6.ICMPTypeEchoRequest
	}

	for _, ep := range p.targets {
		target := ep.Name
		if int(p.results[target].sent) != int(p.c.GetPacketsPerProbe()) {
			t.Errorf("Mismatch in number of packets recorded to be sent. Sent: %d, Recorded: %d", p.c.GetPacketsPerProbe(), p.results[target].sent)
		}
		if len(tic.sentPackets[target]) != int(p.c.GetPacketsPerProbe()) {
			t.Errorf("Mismatch in number of packets received. Sent: %d, Got: %d", p.c.GetPacketsPerProbe(), len(tic.sentPackets[target]))
		}
		close(tic.sentPackets[target])
		for b := range tic.sentPackets[target] {
			// Make sure packets parse ok
			m, err := icmp.ParseMessage(protocol, b)
			if err != nil {
				t.Errorf("%v", err)
			}
			// Check packet type
			if m.Type != expectedMsgType {
				t.Errorf("Wrong packet type. Got: %v, expected: %v", m.Type, expectedMsgType)
			}
			// Check packet size
			if len(b) != int(p.c.GetPayloadSize())+8 {
				t.Errorf("Wrong packet size. Got: %d, expected: %d", len(b), int(p.c.GetPayloadSize())+8)
			}
			// Verify ICMP id and sequence number
			pkt, ok := m.Body.(*icmp.Echo)
			if !ok {
				t.Errorf("Wrong ICMP packet body")
			}
			if pkt.ID != int(runID) {
				t.Errorf("Got wrong ICMP ID. Got: %d, Expected: %d", pkt.ID, runID)
			}
			if pkt.Seq&0xff00 != int(runID)&0xff00 {
				t.Errorf("Got wrong ICMP base seq number. Got: %d, Expected: %d", pkt.Seq&0xff00, runID&0xff00)
			}
		}
	}
}

func newProbe(c *configpb.ProbeConf, ipVersion int, t []string) (*Probe, error) {
	p := &Probe{
		name: "ping_test",
		opts: &options.Options{
			ProbeConf:   c,
			Targets:     targets.StaticTargets(strings.Join(t, ",")),
			Interval:    2 * time.Second,
			Timeout:     time.Second,
			IPVersion:   ipVersion,
			LatencyUnit: time.Millisecond,
		},
	}
	return p, p.initInternal()
}

// Test sendPackets IPv4, raw sockets
func TestSendPackets(t *testing.T) {
	p, err := newProbe(&configpb.ProbeConf{}, 0, []string{"2.2.2.2", "3.3.3.3"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	sendAndCheckPackets(p, t)
}

// Test sendPackets IPv6, raw sockets
func TestSendPacketsIPv6(t *testing.T) {
	p, err := newProbe(&configpb.ProbeConf{}, 6, []string{"::2", "::3"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	sendAndCheckPackets(p, t)
}

// Test sendPackets IPv6, raw sockets, no packets should come on IPv4 target
func TestSendPacketsIPv6ToIPv4Hosts(t *testing.T) {
	c := &configpb.ProbeConf{}
	p, err := newProbe(&configpb.ProbeConf{}, 6, []string{"2.2.2.2"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	tic := newTestICMPConn(p.opts, p.targets)
	p.conn = tic
	trackerChan := make(chan bool, int(c.GetPacketsPerProbe())*len(p.targets))
	p.sendPackets(p.newRunID(), trackerChan)
	for _, target := range p.targets {
		if len(tic.sentPackets[target.Name]) != 0 {
			t.Errorf("IPv6 probe: should not have received any packets for IPv4 only targets, but got %d packets", len(tic.sentPackets[target.Name]))
		}
	}
}

// Test sendPackets IPv4, datagram sockets
func TestSendPacketsDatagramSocket(t *testing.T) {
	c := &configpb.ProbeConf{}
	c.UseDatagramSocket = proto.Bool(true)
	p, err := newProbe(c, 0, []string{"2.2.2.2", "3.3.3.3"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	sendAndCheckPackets(p, t)
}

// Test sendPackets IPv6, datagram sockets
func TestSendPacketsIPv6DatagramSocket(t *testing.T) {
	p, err := newProbe(&configpb.ProbeConf{UseDatagramSocket: proto.Bool(true)}, 6, []string{"::2", "::3"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	sendAndCheckPackets(p, t)
}

// Test runProbe IPv4, raw sockets
func testRunProbe(t *testing.T, ipVersion int, useDatagramSocket bool, payloadSize int) {
	t.Helper()

	c := &configpb.ProbeConf{
		UseDatagramSocket: proto.Bool(useDatagramSocket),
	}

	// if payloadSize is non-zero, set it in config.
	if payloadSize != 0 {
		c.PayloadSize = proto.Int32(int32(payloadSize))
	}

	var targets []string

	if ipVersion == 4 {
		targets = []string{"2.2.2.2", "3.3.3.3", "4.4.4.4"}
	} else {
		targets = []string{"::2", "::3", "::4"}
	}

	p, err := newProbe(c, ipVersion, targets)
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}

	p.conn = newTestICMPConn(p.opts, p.targets)
	p.runProbe()
	for _, ep := range p.targets {
		target := ep.Name

		p.l.Infof("target: %s, sent: %d, received: %d, total_rtt: %s", target, p.results[target].sent, p.results[target].rcvd, p.results[target].latency)
		if p.results[target].sent == 0 || (p.results[target].sent != p.results[target].rcvd) {
			t.Errorf("We are leaking packets. Sent: %d, Received: %d", p.results[target].sent, p.results[target].rcvd)
		}
	}
}

// Test runProbe
func TestRunProbe(t *testing.T) {
	for _, dgram := range []bool{false, true} {
		for _, ipV := range []int{4, 6} {
			for _, size := range []int{8, 56, 256, 1024, maxPacketSize - icmpHeaderSize} {
				t.Run(fmt.Sprintf("version:%d,dgram_socket:%v,payloadSize: %d", ipV, dgram, size), func(t *testing.T) {
					testRunProbe(t, ipV, dgram, size)
				})
			}
		}
	}
}

func TestDataIntegrityValidation(t *testing.T) {
	p, err := newProbe(&configpb.ProbeConf{}, 0, []string{"2.2.2.2", "3.3.3.3"})
	if err != nil {
		t.Fatalf("Got error from newProbe: %v", err)
	}
	tic := newTestICMPConn(p.opts, p.targets)
	p.conn = tic

	p.runProbe()

	// We'll use sent and rcvd to take a snapshot of the probe counters.
	sent := make(map[string]int64)
	rcvd := make(map[string]int64)
	for _, ep := range p.targets {
		target := ep.Name

		sent[target] = p.results[target].sent
		rcvd[target] = p.results[target].rcvd

		p.l.Infof("target: %s, sent: %d, received: %d, total_rtt: %s", target, sent[target], rcvd[target], p.results[target].latency)
		if sent[target] == 0 || (sent[target] != rcvd[target]) {
			t.Errorf("We are leaking packets. Sent: %d, Received: %d", sent[target], rcvd[target])
		}
	}

	// Set the test icmp connection to flip the last byte.
	tic.setFlipLastByte()

	// Run probe again, this time we should see data integrity validation failures.
	p.runProbe()
	for _, ep := range p.targets {
		target := ep.Name

		p.l.Infof("target: %s, sent: %d, received: %d, total_rtt: %s", target, p.results[target].sent, p.results[target].rcvd, p.results[target].latency)

		// Verify that we didn't increased the received counter.
		if p.results[target].rcvd != rcvd[target] {
			t.Errorf("Unexpected change in received packets. Got: %d, Expected: %d", p.results[target].rcvd, rcvd[target])
		}

		// Verify that we increased the validation failure counter.
		expectedFailures := p.results[target].sent - p.results[target].rcvd
		gotFailures := p.results[target].validationFailure.GetKey(dataIntegrityKey)
		if gotFailures != expectedFailures {
			t.Errorf("p.results[%s].validationFailure.GetKey(%s)=%d, expected=%d", target, dataIntegrityKey, gotFailures, expectedFailures)
		}
	}
}

func TestRunProbeRealICMP(t *testing.T) {
	baseTargets := map[int][]string{
		4: {"127.0.1.1", "1.1.1.1", "8.8.8.8", "localhost", "www.google.com", "www.yahoo.com", "www.facebook.com"},
		6: {"2606:4700:4700::1111", "2001:4860:4860::8888", "localhost", "www.google.com", "www.yahoo.com", "www.facebook.com"},
	}

	for _, sockType := range []string{"DGRAM", "RAW"} {
		for _, version := range []int{4, 6} {
			t.Run(fmt.Sprintf("%v_%d", sockType, version), func(t *testing.T) {
				if sockType == "DGRAM" && runtime.GOOS == "windows" {
					t.Skip("Skipping as Windows doesn't support SOCK_DGRAM for ICMP.")
				}

				if sockType == "RAW" && runtime.GOOS != "windows" && os.Geteuid() != 0 {
					t.Skip("Skipping real ping test with RAW sockets as not running as root.")
				}

				if _, disableV6 := os.LookupEnv("DISABLE_IPV6_TESTS"); disableV6 && version == 6 {
					t.Skip("Skipping IPv6 tests as DISABLE_IPV6_TESTS is set.")
				}

				c := &configpb.ProbeConf{
					UseDatagramSocket: proto.Bool(sockType == "DGRAM"),
				}

				targets := baseTargets[version]
				if hosts, ok := os.LookupEnv("PING_HOSTS_V" + strconv.Itoa(version)); ok {
					if hosts == "" {
						t.Skip("No targets provided through env variable, skipping")
					}
					targets = strings.Split(hosts, ",")
				}

				p, err := newProbe(c, version, targets)
				if err != nil {
					t.Fatalf("Got error from newProbe: %v", err)
				}
				if err := p.listen(); err != nil {
					t.Errorf("listen err: %v", err)
					return
				}

				p.runProbe()

				var rcvd int
				for _, ep := range p.targets {
					target := ep.Name

					t.Logf("target: %s, sent: %d, received: %d, total_rtt: %s", target, p.results[target].sent, p.results[target].rcvd, p.results[target].latency)
					rcvd += int(p.results[target].rcvd)
				}

				expectedRcvd := len(targets) * int(p.c.GetPacketsPerProbe())
				// This test passes even if we receive just 20% of the expected packets.
				if rcvd <= expectedRcvd/5 {
					t.Errorf("total success: %d, expected: %d", rcvd, expectedRcvd)
				}
			})
		}
	}
}
