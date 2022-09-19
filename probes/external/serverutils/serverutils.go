// Copyright 2017 The Cloudprober Authors.
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

// Package serverutils provides utilities to work with the cloudprober's external probe.
package serverutils

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	serverpb "github.com/cloudprober/cloudprober/probes/external/proto"
)

func readPayload(r *bufio.Reader) ([]byte, error) {
	// header format is: "\nContent-Length: %d\n\n"
	const prefix = "Content-Length: "
	var line string
	var length int
	var err error

	// Read lines until header line is found
	for {
		line, err = r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(line, prefix) {
			break
		}
	}

	// Parse content length from the header
	length, err = strconv.Atoi(line[len(prefix) : len(line)-1])
	if err != nil {
		return nil, err
	}
	// Consume the blank line following the header line
	if _, err = r.ReadSlice('\n'); err != nil {
		return nil, err
	}

	// Slurp in the payload
	buf := make([]byte, length)
	if _, err = io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// ReadProbeReply reads ProbeReply from the supplied bufio.Reader and returns it to
// the caller.
func ReadProbeReply(r *bufio.Reader) (*serverpb.ProbeReply, error) {
	buf, err := readPayload(r)
	if err != nil {
		return nil, err
	}
	rep := new(serverpb.ProbeReply)
	return rep, proto.Unmarshal(buf, rep)
}

// ReadProbeRequest reads and parses ProbeRequest protocol buffers from the given
// bufio.Reader.
func ReadProbeRequest(r *bufio.Reader) (*serverpb.ProbeRequest, error) {
	buf, err := readPayload(r)
	if err != nil {
		return nil, err
	}
	req := new(serverpb.ProbeRequest)
	return req, proto.Unmarshal(buf, req)
}

// WriteMessage marshals the a proto message and writes it to the writer "w"
// with appropriate Content-Length header.
func WriteMessage(pb proto.Message, w io.Writer) error {
	buf, err := proto.Marshal(pb)
	if err != nil {
		return fmt.Errorf("Failed marshalling proto message: %v", err)
	}
	if _, err := fmt.Fprintf(w, "\nContent-Length: %d\n\n%s", len(buf), buf); err != nil {
		return fmt.Errorf("Failed writing response: %v", err)
	}
	return nil
}

// ServeContext blocks until canceled, servicing probe requests. It
// communicates over os.Stdin and os.Stdout, so nothing else in the
// program can use them. Note that this function is provided mainly to
// help external probe server implementations. Cloudprober doesn't
// make use of it. Example usage:
//	import (
//		"context"
//		serverpb "github.com/cloudprober/cloudprober/probes/external/proto"
//		"github.com/cloudprober/cloudprober/probes/external/serverutils"
//	)
//	func runProbe(ctx Context.context, opts []*cppb.ProbeRequest_Option) {
//  	...
//	}
//	serverutils.ServeContext(ctx, func(ctx context.Context, req *serverpb.ProbeRequest, reply *serverpb.ProbeReply) {
// 		payload, errMsg, _ := runProbe(ctx, req.GetOptions())
//		reply.Payload = proto.String(payload)
//		if errMsg != "" {
//			reply.ErrorMessage = proto.String(errMsg)
//		}
//	})
func ServeContext(ctx context.Context, probeFunc func(context.Context, *serverpb.ProbeRequest, *serverpb.ProbeReply)) {
	stdin := bufio.NewReader(os.Stdin)

	go func() {
		// There is no neat way to do this in go, but there
		// are several ugly ways. The simplest is to close
		// stdin and let all the readers fail.
		<-ctx.Done()
		os.Stdin.Close()
	}()
	
	repliesChan := make(chan *serverpb.ProbeReply)

	// Write replies to stdout. These are not required to be in-order.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case rep := <-repliesChan:
				if err := WriteMessage(rep, os.Stdout); err != nil {
					log.Fatal(err)
				}
			}

		}
	}()

	// Read requests from stdin, and dispatch probes to service them.
	for {
		request, err := ReadProbeRequest(stdin)
		if ctx.Err() != nil {
			// Catches errors cascading from os.Stdin.Close - if we're canceled, nobody cares
			return
		} else if err != nil {
			log.Fatalf("Failed reading request: %v", err)
		}
		go func() {
			reply := &serverpb.ProbeReply{
				RequestId: request.RequestId,
			}
			done := make(chan bool, 1)
			pctx, cancel := context.WithTimeout(ctx, time.Duration(*request.TimeLimit) * time.Millisecond)
			defer cancel()
			go func() {
				probeFunc(pctx, request, reply)
				done <- true
			}()
			select {
			case <-done:
				repliesChan <- reply
			case <-pctx.Done():
				fmt.Fprintf(os.Stderr, "Dropping request %v: %s\n", *reply.RequestId, pctx.Err())
			}
		}()
	}
}

// Serve blocks indefinitely, servicing probe requests. Otherwise equivalent to ServeContext.
func Serve(probeFunc func(*serverpb.ProbeRequest, *serverpb.ProbeReply)) {
	ServeContext(context.Background(), func(_ context.Context, request *serverpb.ProbeRequest, reply *serverpb.ProbeReply) {
		probeFunc(request, reply)
	})
}
