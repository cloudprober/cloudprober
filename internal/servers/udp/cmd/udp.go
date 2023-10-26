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

/*
This binary implements a stand-alone UDP server using the
cloudprober/internal/servers/udp/udp package.
*/
package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/cloudprober/cloudprober/internal/servers/udp"
	configpb "github.com/cloudprober/cloudprober/internal/servers/udp/proto"
	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/protobuf/proto"

	"flag"
)

var (
	port         = flag.Int("port", 31122, "Port to listen on")
	responseType = flag.String("type", "echo", "Server type: echo|discard")
)

func main() {
	flag.Parse()

	config := &configpb.ServerConf{
		Port: proto.Int32(int32(*port)),
		Type: configpb.ServerConf_DISCARD.Enum(),
	}
	server, err := udp.New(context.Background(), config, logger.NewWithAttrs(slog.String("component", "UDP_"+*responseType)))
	if err != nil {
		log.Fatalf("Error creating a new UDP server: %v", err)
	}
	log.Fatal(server.Start(context.Background(), nil))
}
