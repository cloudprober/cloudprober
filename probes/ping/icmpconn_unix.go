// Copyright 2020 The Cloudprober Authors.
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

//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package ping

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

// NativeEndian is the machine native endian implementation of ByteOrder.
var NativeEndian binary.ByteOrder

func sockaddr(sourceIP net.IP, ipVer int) (syscall.Sockaddr, error) {
	switch ipVer {
	case 4:
		v4 := sourceIP.To4()
		if v4 == nil {
			return nil, fmt.Errorf("address %s is not a valid IPv4 address", sourceIP)
		}

		sa := &syscall.SockaddrInet4{}
		copy(sa.Addr[:], v4)
		return sa, nil
	case 6:
		v6 := sourceIP.To16()
		if v6 == nil {
			return nil, fmt.Errorf("address %s is not a valid IPv6 address", sourceIP)
		}

		sa := &syscall.SockaddrInet6{}
		copy(sa.Addr[:], v6)
		return sa, nil
	default:
		return nil, net.InvalidAddrError("unexpected family")
	}
}

// listenPacket listens for incoming ICMP packets addressed to sourceIP.
// We need to write our own listenPacket instead of using "net.ListenPacket"
// for the following reasons:
//  1. ListenPacket doesn't support ICMP for SOCK_DGRAM sockets. You create
//     datagram sockets by specifying network as "udp", but UDP new connection
//     implementation ignores the protocol field entirely.
//  2. ListenPacket doesn't support setting socket options (we need
//     SO_TIMESTAMP) in a straightforward way.
func (p *Probe) listenPacket(sourceIP net.IP) (*icmpPacketConn, error) {
	// Note that the disableFragmentation bit only applies on Linux systems.
	var family, proto int

	switch p.ipVer {
	case 4:
		family, proto = syscall.AF_INET, protocolICMP
	case 6:
		family, proto = syscall.AF_INET6, protocolIPv6ICMP
	}

	sockType := syscall.SOCK_RAW
	if p.useDatagramSocket {
		sockType = syscall.SOCK_DGRAM
	}

	s, err := syscall.Socket(family, sockType, proto)
	if err != nil {
		return nil, os.NewSyscallError("socket", err)
	}

	// Set socket option to receive kernel's timestamp from each packet.
	// Ref: https://man7.org/linux/man-pages/man7/socket.7.html (SO_TIMESTAMP)
	if err := syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_TIMESTAMP, 1); err != nil {
		syscall.Close(s)
		return nil, os.NewSyscallError("setsockopt", err)
	}
	if p.disableFragmentation && p.ipVer == 4 && runtime.GOOS == "linux" {
		// Copied from
		// https://github.com/golang/go/blob/master/src/syscall/zerrors_linux_.*.go
		// to make build work for non-linux systems.
		// compiling on non-linux unix systems.
		const linux_IP_MTU_DISCOVER = 0xa
		const linux_IP_PMTUDISC_DO = 0x2
		// Set don't fragment bit.
		if err := syscall.SetsockoptInt(s, syscall.IPPROTO_IP, linux_IP_MTU_DISCOVER, linux_IP_PMTUDISC_DO); err != nil {
			syscall.Close(s)
			return nil, os.NewSyscallError("setsockopt", err)
		}
	}

	sa, err := sockaddr(sourceIP, p.ipVer)
	if err != nil {
		syscall.Close(s)
		return nil, err
	}
	if err := syscall.Bind(s, sa); err != nil {
		syscall.Close(s)
		return nil, os.NewSyscallError("bind", err)
	}

	// FilePacketConn is you get a PacketConn from a socket descriptor.
	// Behind the scene, FilePacketConn creates either an IPConn or UDPConn,
	// based on socket's local address (it gets that from the fd).
	f := os.NewFile(uintptr(s), "icmp")
	c, cerr := net.FilePacketConn(f)
	f.Close()

	if cerr != nil {
		syscall.Close(s)
		return nil, cerr
	}

	ipc := &icmpPacketConn{c: c}
	ipc.ipConn, _ = c.(*net.IPConn)
	ipc.udpConn, _ = c.(*net.UDPConn)

	return ipc, nil
}

type icmpPacketConn struct {
	c net.PacketConn

	// We use ipConn and udpConn for reading OOB data from the connection.
	ipConn  *net.IPConn
	udpConn *net.UDPConn
}

func timestampFromControlMessage(oob []byte) (time.Time, error) {
	cmsgs, err := syscall.ParseSocketControlMessage(oob)
	if err != nil {
		return time.Time{}, err
	}

	for _, m := range cmsgs {
		// We are interested only in socket-level control messages
		// (syscall.SOL_SOCKET)
		if m.Header.Level != syscall.SOL_SOCKET {
			continue
		}

		// SCM_TIMESTAMP is the type of the timestamp control message.
		// Note that syscall.SO_TIMESTAMP == syscall.SCM_TIMESTAMP for linux, but
		// that doesn't have to be true for other operating systems, e.g. Mac OS X.
		if m.Header.Type == syscall.SCM_TIMESTAMP {
			// Some old 32-bit systems may use smaller struct for timeval. See
			// https://github.com/cloudprober/cloudprober/issues/175 for reference.
			if len(m.Data) == 8 {
				sec := NativeEndian.Uint32(m.Data)
				usec := NativeEndian.Uint32(m.Data[4:])
				return time.Unix(int64(sec), int64(usec)*1e3), nil
			}

			if len(m.Data) < 16 {
				return time.Time{}, fmt.Errorf("timestamp control message data size (%d) is less than timestamp size (16 bytes)", len(m.Data))
			}

			sec := NativeEndian.Uint64(m.Data)
			usec := NativeEndian.Uint64(m.Data[8:])
			return time.Unix(int64(sec), int64(usec)*1e3), nil
		}
	}

	return time.Time{}, nil
}

func (ipc *icmpPacketConn) read(buf []byte) (n int, addr net.Addr, recvTime time.Time, err error) {
	// We need to convert to IPConn/UDPConn so that we can read out-of-band data
	// using ReadMsg<IP,UDP> functions. PacketConn interface doesn't have method
	// that exposes OOB data.
	oob := make([]byte, 64)
	var oobn int
	if ipc.ipConn != nil {
		n, oobn, _, addr, err = ipc.ipConn.ReadMsgIP(buf, oob)
	}
	if ipc.udpConn != nil {
		n, oobn, _, addr, err = ipc.udpConn.ReadMsgUDP(buf, oob)
	}

	if err != nil {
		return
	}
	recvTime, err = timestampFromControlMessage(oob[:oobn])
	return
}

// write writes the ICMP message b to dst.
func (ipc *icmpPacketConn) write(buf []byte, dst net.Addr) (int, error) {
	return ipc.c.WriteTo(buf, dst)
}

// Close closes the endpoint.
func (ipc *icmpPacketConn) close() {
	ipc.c.Close()
}

// setReadDeadline sets the read deadline associated with the
// endpoint.
func (ipc *icmpPacketConn) setReadDeadline(t time.Time) {
	ipc.c.SetReadDeadline(t)
}

func (p *Probe) newICMPConn(sourceIP net.IP) (*icmpPacketConn, error) {
	return p.listenPacket(sourceIP)
}

// Find out native endianness when this packages is loaded.
// This code is based on:
// https://github.com/golang/net/blob/master/internal/socket/sys.go
func init() {
	i := uint32(1)
	b := (*[4]byte)(unsafe.Pointer(&i))
	if b[0] == 1 {
		NativeEndian = binary.LittleEndian
	} else {
		NativeEndian = binary.BigEndian
	}
}
