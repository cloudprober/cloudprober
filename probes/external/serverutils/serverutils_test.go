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

package serverutils

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	serverpb "github.com/cloudprober/cloudprober/probes/external/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestReadProbeReply(t *testing.T) {
	reply := &serverpb.ProbeReply{
		RequestId: proto.Int32(123),
		Payload:   proto.String("test payload"),
	}

	buf := new(bytes.Buffer)
	err := WriteMessage(reply, buf)
	assert.NoError(t, err)

	r := bufio.NewReader(buf)
	gotReply := &serverpb.ProbeReply{}
	assert.NoError(t, ReadMessage(context.Background(), gotReply, r))
	assert.True(t, proto.Equal(reply, gotReply), "proto equal check")
}

func TestReadProbeRequest(t *testing.T) {
	request := &serverpb.ProbeRequest{
		RequestId: proto.Int32(123),
		TimeLimit: proto.Int32(5000),
		Options:   []*serverpb.ProbeRequest_Option{{Name: proto.String("opt1"), Value: proto.String("val1")}},
	}

	buf := new(bytes.Buffer)
	err := WriteMessage(request, buf)
	assert.NoError(t, err)

	r := bufio.NewReader(buf)
	gotRequest := &serverpb.ProbeRequest{}
	assert.NoError(t, ReadMessage(context.Background(), gotRequest, r))
	assert.True(t, proto.Equal(request, gotRequest))
}

func TestWriteMessage(t *testing.T) {
	// Create a mock proto message
	message := &serverpb.ProbeReply{
		RequestId: proto.Int32(123),
		Payload:   proto.String("test payload"),
	}

	// Create a buffer to write the message to
	buf := new(bytes.Buffer)

	// Call the WriteMessage function
	err := WriteMessage(message, buf)
	assert.NoError(t, err)

	// Verify the output
	fmt.Printf("buf: %x\n", buf.String())
	expectedOutput := "\nContent-Length: 16\n\n\b{\x1a\ftest payload"
	assert.Equal(t, expectedOutput, buf.String())
}

func TestServe(t *testing.T) {
	// Create a mock probe function
	mockProbeFunc := func(request *serverpb.ProbeRequest, reply *serverpb.ProbeReply) {
		reply.RequestId = request.RequestId
		reply.Payload = proto.String("test payload")
	}

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	// Call the Serve function
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go serve(ctx, mockProbeFunc, stdinR, stdoutW, stdoutW)

	// Create a mock probe request
	mockRequest := &serverpb.ProbeRequest{
		RequestId: proto.Int32(123),
		TimeLimit: proto.Int32(5000),
		Options:   []*serverpb.ProbeRequest_Option{{Name: proto.String("opt1"), Value: proto.String("val1")}},
	}

	// Write the mock probe request to the mock stdin
	WriteMessage(mockRequest, stdinW)

	// Wait for the reply to be written to the mock stdout
	reply := &serverpb.ProbeReply{}
	assert.NoError(t, ReadMessage(context.Background(), reply, bufio.NewReader(stdoutR)))

	// Verify the reply
	expectedReply := &serverpb.ProbeReply{
		RequestId: proto.Int32(123),
		Payload:   proto.String("test payload"),
	}
	assert.True(t, proto.Equal(expectedReply, reply))
}
