// Copyright 2017-2024 The Cloudprober Authors.
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

	serverpb "github.com/cloudprober/cloudprober/probes/external/proto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// readPayload reads the payload from the given bufio.Reader. It expects the
// payload to be preceded by a header line with the content length:
// "\nContent-Length: %d\n\n"
// Note: This function takes a context, but canceling the context doesn't
// cancel the ongoing read call.
func readPayload(ctx context.Context, r *bufio.Reader) ([]byte, error) {
	// header format is: "\nContent-Length: %d\n\n"
	const prefix = "Content-Length: "
	var line string
	var length int
	var err error

	// Read lines until header line is found
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
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

// ReadMessage reads protocol buffers from the given bufio.Reader.
func ReadMessage(ctx context.Context, msg protoreflect.ProtoMessage, r *bufio.Reader) error {
	buf, err := readPayload(ctx, r)
	if err != nil {
		return err
	}
	return proto.Unmarshal(buf, msg)
}

// WriteMessage marshals the a proto message and writes it to the writer "w"
// with appropriate Content-Length header.
func WriteMessage(pb proto.Message, w io.Writer) error {
	buf, err := proto.Marshal(pb)
	if err != nil {
		return fmt.Errorf("failed marshalling proto message: %v", err)
	}
	if _, err := fmt.Fprintf(w, "\nContent-Length: %d\n\n%s", len(buf), buf); err != nil {
		return fmt.Errorf("failed writing response: %v", err)
	}
	return nil
}

func serve(ctx context.Context, probeFunc func(*serverpb.ProbeRequest, *serverpb.ProbeReply), stdin io.Reader, stdout, stderr io.Writer) {
	repliesChan := make(chan *serverpb.ProbeReply)

	// Write replies to stdout. These are not required to be in-order.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case rep := <-repliesChan:
				if err := WriteMessage(rep, stdout); err != nil {
					log.Fatal(err)
				}
			}
		}
	}()

	// Read requests from stdin, and dispatch probes to service them.
	for {
		if ctx.Err() != nil {
			return
		}

		request := new(serverpb.ProbeRequest)
		err := ReadMessage(ctx, request, bufio.NewReader(stdin))
		if err != nil {
			log.Fatalf("Failed reading request: %v", err)
		}

		go func() {
			reply := &serverpb.ProbeReply{
				RequestId: request.RequestId,
			}

			probeDone := make(chan bool, 1)
			timeout := time.After(time.Duration(*request.TimeLimit) * time.Millisecond)

			go func() {
				probeFunc(request, reply)
				probeDone <- true
			}()

			select {
			case <-probeDone:
				repliesChan <- reply
			case <-timeout:
				// drop the request on the floor.
				if len(request.GetOptions()) > 0 {
					var sb strings.Builder

					for _, option := range request.Options {
						sb.WriteString(fmt.Sprintf("%s: %s,", *option.Name, *option.Value))
					}

					fmt.Fprintf(stderr, "Timeout for request %v (%s)\n", *reply.RequestId, sb.String())
				} else {
					fmt.Fprintf(stderr, "Timeout for request %v\n", *reply.RequestId)
				}

			}
		}()
	}
}

// ServeContext blocks indefinitely, servicing probe requests. Note that this function is
// provided mainly to help external probe server implementations. Cloudprober doesn't
// make use of it. Example usage:
//
//		import (
//			serverpb "github.com/cloudprober/cloudprober/probes/external/proto"
//			"github.com/cloudprober/cloudprober/probes/external/serverutils"
//		)
//		func runProbe(opts []*cppb.ProbeRequest_Option) {
//	 	...
//		}
//		serverutils.ServeContext(ctx, func(req *serverpb.ProbeRequest, reply *serverpb.ProbeReply) {
//			payload, errMsg, _ := runProbe(req.GetOptions())
//			reply.Payload = proto.String(payload)
//			if errMsg != "" {
//				reply.ErrorMessage = proto.String(errMsg)
//			}
//		})
func ServeContext(ctx context.Context, probeFunc func(*serverpb.ProbeRequest, *serverpb.ProbeReply)) {
	serve(ctx, probeFunc, bufio.NewReader(os.Stdin), os.Stdout, os.Stderr)
}

// Serve is similar to ServeContext but uses the background context.
func Serve(probeFunc func(*serverpb.ProbeRequest, *serverpb.ProbeReply)) {
	serve(context.Background(), probeFunc, bufio.NewReader(os.Stdin), os.Stdout, os.Stderr)
}
