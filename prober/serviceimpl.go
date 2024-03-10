// Copyright 2019 The Cloudprober Authors.
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

package prober

import (
	"context"
	"errors"

	pb "github.com/cloudprober/cloudprober/prober/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// AddProbe adds the given probe to cloudprober.
func (pr *Prober) AddProbe(ctx context.Context, req *pb.AddProbeRequest) (*pb.AddProbeResponse, error) {
	pr.l.Info("AddProbe called")

	p := req.GetProbeConfig()

	if p == nil {
		return &pb.AddProbeResponse{}, status.Errorf(codes.InvalidArgument, "probe config cannot be nil")
	}

	if err := pr.addProbe(p); err != nil {
		return &pb.AddProbeResponse{}, err
	}

	// Send probe to the start probe channel to be started by a goroutine started
	// at the prober start time.
	pr.grpcStartProbeCh <- p.GetName()

	if *probesConfigSavePath != "" {
		pr.saveProbesConfigUnprotected(*probesConfigSavePath)
	}

	return &pb.AddProbeResponse{}, nil
}

// RemoveProbe gRPC method cancels the given probe and removes its from the
// prober's internal database.
func (pr *Prober) RemoveProbe(ctx context.Context, req *pb.RemoveProbeRequest) (*pb.RemoveProbeResponse, error) {
	pr.l.Infof("RemoveProbe called with: %s", req.GetProbeName())

	pr.mu.Lock()
	defer pr.mu.Unlock()

	name := req.GetProbeName()

	if name == "" {
		return &pb.RemoveProbeResponse{}, status.Errorf(codes.InvalidArgument, "probe name cannot be empty")
	}

	if pr.Probes[name] == nil {
		return &pb.RemoveProbeResponse{}, status.Errorf(codes.NotFound, "probe %s not found", name)
	}

	pr.probeCancelFunc[name]()
	delete(pr.Probes, name)

	if *probesConfigSavePath != "" {
		pr.saveProbesConfigUnprotected(*probesConfigSavePath)
	}

	return &pb.RemoveProbeResponse{}, nil
}

// ListProbes gRPC method returns the list of probes from the in-memory database.
func (pr *Prober) ListProbes(ctx context.Context, req *pb.ListProbesRequest) (*pb.ListProbesResponse, error) {
	pr.l.Info("ListProbes called")
	pr.mu.Lock()
	defer pr.mu.Unlock()

	resp := &pb.ListProbesResponse{}

	for name, p := range pr.Probes {
		resp.Probe = append(resp.Probe, &pb.Probe{
			Name:   proto.String(name),
			Config: p.ProbeDef,
		})
	}

	return resp, nil
}

func (pr *Prober) SaveProbesConfig(ctx context.Context, req *pb.SaveProbesConfigRequest) (*pb.SaveProbesConfigResponse, error) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	filePath := req.GetFilePath()
	if filePath == "" {
		filePath = *probesConfigSavePath
	}
	if filePath == "" {
		return nil, errors.New("file_path not provided and --config_save_path flag is also not set")
	}

	if err := pr.saveProbesConfigUnprotected(filePath); err != nil {
		return nil, err
	}

	return &pb.SaveProbesConfigResponse{
		FilePath: &filePath,
	}, nil
}
