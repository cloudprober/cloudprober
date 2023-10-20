// Copyright 2020-2021 The Cloudprober Authors.
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
Package file implements a file-based targets for cloudprober.
*/
package file

import (
	"context"

	"github.com/cloudprober/cloudprober/internal/rds/client"
	client_configpb "github.com/cloudprober/cloudprober/internal/rds/client/proto"
	"github.com/cloudprober/cloudprober/internal/rds/file"
	file_configpb "github.com/cloudprober/cloudprober/internal/rds/file/proto"
	rdspb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/targets/file/proto"
	dnsRes "github.com/cloudprober/cloudprober/targets/resolver"
	"google.golang.org/protobuf/proto"
)

// New returns new file targets.
func New(opts *configpb.TargetsConf, res *dnsRes.Resolver, l *logger.Logger) (*client.Client, error) {
	lister, err := file.New(&file_configpb.ProviderConfig{
		FilePath:  []string{opts.GetFilePath()},
		ReEvalSec: proto.Int32(opts.GetReEvalSec()),
	}, l)
	if err != nil {
		return nil, err
	}

	// We can be really aggressive about client refresh interval as "file" RDS
	// provider is smart about using the last_modified field of the response,
	// and client checking for targets before file has been reloaded will not
	// trigger unnecessary recreation of target objects.
	// Ref: https://github.com/cloudprober/cloudprober/blob/5bec0db1ac908e69bff0fbca3182415c4e267d64/rds/client/client.go#L103
	clientConf := &client_configpb.ClientConf{
		Request:   &rdspb.ListResourcesRequest{Filter: opts.GetFilter()},
		ReEvalSec: proto.Int32(1),
	}

	return client.New(clientConf, func(_ context.Context, req *rdspb.ListResourcesRequest) (*rdspb.ListResourcesResponse, error) {
		return lister.ListResources(req)
	}, l)
}
