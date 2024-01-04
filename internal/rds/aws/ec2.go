// Copyright 2017-2023 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aws

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/rds/aws/proto"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/internal/rds/server/filter"
	"github.com/cloudprober/cloudprober/logger"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"google.golang.org/protobuf/proto"
)

// instanceInfo represents instance items that we fetch from the API.
type instanceInfo struct {
	ID     string
	Tags   map[string]string
	IPAddr string
}

// instanceData represents objects that we store in cache.
type instanceData struct {
	ii          *instanceInfo
	lastUpdated int64
}

/*
AWSInstancesFilters defines filters supported by the ec2_instances resource
type.

	 Example:
	 filter {
		 key: "name"
		 value: "cloudprober.*"
	 }
	 filter {
		 key: "labels.app"
		 value: "service-a"
	 }
*/
var AWSInstancesFilters = struct {
	RegexFilterKeys []string
	LabelsFilter    bool
}{
	[]string{"name"},
	true,
}

// ec2InstancesLister is a AWS EC2 instances lister. It implements a cache,
// that's populated at a regular interval by making the AWS API calls.
// Listing actually only returns the current contents of that cache.
type ec2InstancesLister struct {
	c      *configpb.EC2Instances
	client ec2.DescribeInstancesAPIClient
	l      *logger.Logger
	mu     sync.RWMutex
	names  []string
	cache  map[string]*instanceData
}

// listResources returns the list of resource records, where each record
// consists of an instance name and the IP address associated with it. IP address
// to return is selected based on the provided ipConfig.
func (il *ec2InstancesLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	var resources []*pb.Resource

	allFilters, err := filter.ParseFilters(req.GetFilter(), AWSInstancesFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, err
	}

	nameFilter, labelsFilter := allFilters.RegexFilters["name"], allFilters.LabelsFilter

	il.mu.RLock()
	defer il.mu.RUnlock()

	for _, name := range il.names {
		ins := il.cache[name].ii
		if ins == nil {
			il.l.Errorf("ec2_instances: cached info missing for %s", name)
			continue
		}

		if nameFilter != nil && !nameFilter.Match(name, il.l) {
			continue
		}
		if labelsFilter != nil && !labelsFilter.Match(ins.Tags, il.l) {
			continue
		}

		resources = append(resources, &pb.Resource{
			Name:        proto.String(name),
			Ip:          proto.String(ins.IPAddr),
			Labels:      ins.Tags,
			LastUpdated: proto.Int64(il.cache[name].lastUpdated),
		})
	}

	il.l.Infof("ec2_instances.listResources: returning %d instances", len(resources))
	return resources, nil
}

// expand runs equivalent API calls as "aws describe-instances",
// and is used to populate the cache. It will list the EC2 instances
// in the target account with some basic networking information
func (il *ec2InstancesLister) expand(reEvalInterval time.Duration) {
	il.l.Infof("ec2_instances.expand: expanding AWS EC2 targets")

	result, err := il.client.DescribeInstances(context.TODO(), nil)
	if err != nil {
		il.l.Errorf("ec2_instances.expand: error while listing instances: %v", err)
		return
	}

	var ids = make([]string, 0)
	var cache = make(map[string]*instanceData)

	ts := time.Now().Unix()
	for _, r := range result.Reservations {
		for _, i := range r.Instances {

			if i.PrivateIpAddress == nil {
				continue
			}

			ii := &instanceInfo{
				ID:     *i.InstanceId,
				IPAddr: *i.PrivateIpAddress,
				Tags:   make(map[string]string),
			}

			// Convert to map
			for _, t := range i.Tags {
				ii.Tags[*t.Key] = *t.Value
			}

			cache[*i.InstanceId] = &instanceData{ii, ts}
			ids = append(ids, *i.InstanceId)
		}
	}

	il.mu.Lock()
	il.names = ids
	il.cache = cache
	il.mu.Unlock()

	il.l.Infof("ec2_instances.expand: got %d instances", len(ids))
}

func newEC2InstancesLister(c *configpb.EC2Instances, region string, l *logger.Logger) (*ec2InstancesLister, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("AWS configuration error: %v", err)
	}

	client := ec2.NewFromConfig(cfg)

	il := &ec2InstancesLister{
		c:      c,
		client: client,
		cache:  make(map[string]*instanceData),
		l:      l,
	}

	reEvalInterval := time.Duration(c.GetReEvalSec()) * time.Second
	go func() {
		il.expand(0)
		// Introduce a random delay between 0-reEvalInterval before
		// starting the refresh loop. If there are multiple cloudprober
		// awsInstances, this will make sure that each instance calls AWS
		// API at a different point of time.
		randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
		time.Sleep(time.Duration(randomDelaySec) * time.Second)
		for range time.Tick(reEvalInterval) {
			il.expand(reEvalInterval)
		}
	}()
	return il, nil
}
