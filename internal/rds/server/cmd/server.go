// This binary implements a stand-alone ResourceDiscovery server.
package main

import (
	"context"
	"log"
	"net"
	"os"

	"flag"

	"github.com/cloudprober/cloudprober/internal/rds/server"
	configpb "github.com/cloudprober/cloudprober/internal/rds/server/proto"
	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/prototext"
)

var (
	config      = flag.String("config_file", "", "Config file (ServerConf)")
	addr        = flag.String("addr", ":0", "Port for the gRPC server")
	tlsCertFile = flag.String("tls_cert_file", ":0", "Port for the gRPC server")
	tlsKeyFile  = flag.String("tls_key_file", ":0", "Port for the gRPC server")
)

func main() {
	flag.Parse()

	c := &configpb.ServerConf{}

	// If we are given a config file, read it. If not, use defaults.
	if *config != "" {
		b, err := os.ReadFile(*config)
		if err != nil {
			log.Fatal(err)
		}
		if err := prototext.Unmarshal(b, c); err != nil {
			log.Fatalf("Error while parsing config protobuf %s: Err: %v", string(b), err)
		}
	}

	grpcLn, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("error while creating listener for default gRPC server: %v", err)
	}

	// Create a gRPC server for RDS service.
	var serverOpts []grpc.ServerOption

	if *tlsCertFile != "" {
		creds, err := credentials.NewServerTLSFromFile(*tlsCertFile, *tlsKeyFile)
		if err != nil {
			log.Fatalf("error initializing gRPC server TLS credentials: %v", err)
		}
		serverOpts = append(serverOpts, grpc.Creds(creds))
	}

	grpcServer := grpc.NewServer(serverOpts...)
	srv, err := server.New(context.Background(), c, nil, &logger.Logger{})
	if err != nil {
		log.Fatal(err)
	}
	srv.RegisterWithGRPC(grpcServer)

	grpcServer.Serve(grpcLn)
}
