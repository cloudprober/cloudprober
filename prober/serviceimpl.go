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
	"fmt"
	"os"
	"sort"

	configpb "github.com/cloudprober/cloudprober/config/proto"
	pb "github.com/cloudprober/cloudprober/prober/proto"
	probes_configpb "github.com/cloudprober/cloudprober/probes/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// saveProbesConfigToFile saves the current config to the given file path.
// This function is called whenever we change the config in response to a gRPC
// request.
func (pr *Prober) saveProbesConfigToFile(filePath string) error {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	var keys []string
	for k := range pr.Probes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	cfg := &configpb.ProberConfig{}
	for _, k := range keys {
		cfg.Probe = append(cfg.Probe, pr.Probes[k].ProbeDef)
	}

	textCfg := prototext.MarshalOptions{
		Indent: "  ",
	}.Format(cfg)

	if textCfg == "" && len(pr.Probes) != 0 {
		err := fmt.Errorf("text marshaling of probes config returned an empty string. Config: %v", cfg)
		pr.l.Warning(err.Error())
		return err
	}

	if err := os.WriteFile(filePath, []byte(textCfg), 0644); err != nil {
		pr.l.Errorf("Error saving config to disk: %v", err)
		return err
	}

	return nil
}

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

	// Start probe using the prober's start context.
	if pr.startCtx == nil {
		return &pb.AddProbeResponse{}, status.Errorf(codes.FailedPrecondition, "prober not started")
	}
	pr.startProbe(p.GetName())

	if *probesConfigSavePath != "" {
		pr.saveProbesConfigToFile(*probesConfigSavePath)
	}

	return &pb.AddProbeResponse{}, nil
}

// removeProbe removes the probe with the given name from the prober's internal
// database and cancels the probe's context.
func (pr *Prober) removeProbe(name string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.Probes[name] == nil {
		return fmt.Errorf("probe %s not found", name)
	}

	pr.probeCancelFunc[name]()
	delete(pr.Probes, name)
	return nil
}

// RemoveProbe gRPC method cancels the given probe and removes its from the
// prober's internal database.
func (pr *Prober) RemoveProbe(ctx context.Context, req *pb.RemoveProbeRequest) (*pb.RemoveProbeResponse, error) {
	pr.l.Infof("RemoveProbe called with: %s", req.GetProbeName())

	name := req.GetProbeName()
	if name == "" {
		return &pb.RemoveProbeResponse{}, status.Errorf(codes.InvalidArgument, "probe name cannot be empty")
	}

	if err := pr.removeProbe(name); err != nil {
		return &pb.RemoveProbeResponse{}, status.Errorf(codes.NotFound, "%v", err)
	}

	if *probesConfigSavePath != "" {
		pr.saveProbesConfigToFile(*probesConfigSavePath)
	}

	return &pb.RemoveProbeResponse{}, nil
}

// ListProbes gRPC method returns the list of probes from the in-memory database.
func (pr *Prober) ListProbes(ctx context.Context, req *pb.ListProbesRequest) (*pb.ListProbesResponse, error) {
	pr.l.Info("ListProbes called")
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	resp := &pb.ListProbesResponse{}

	for name, p := range pr.Probes {
		resp.Probe = append(resp.Probe, &pb.Probe{
			Name:   proto.String(name),
			Config: proto.Clone(p.ProbeDef).(*probes_configpb.ProbeDef),
		})
	}

	return resp, nil
}

func (pr *Prober) SaveProbesConfig(ctx context.Context, req *pb.SaveProbesConfigRequest) (*pb.SaveProbesConfigResponse, error) {
	filePath := req.GetFilePath()
	if filePath == "" {
		filePath = *probesConfigSavePath
	}
	if filePath == "" {
		return nil, errors.New("file_path not provided and --config_save_path flag is also not set")
	}

	if err := pr.saveProbesConfigToFile(filePath); err != nil {
		return nil, err
	}

	return &pb.SaveProbesConfigResponse{
		FilePath: &filePath,
	}, nil
}
