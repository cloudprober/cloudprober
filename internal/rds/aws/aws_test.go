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

package aws

import (
	"reflect"
	"testing"

	serverconfigpb "github.com/cloudprober/cloudprober/internal/rds/server/proto"
)

func testAWSConfig(t *testing.T, pc *serverconfigpb.Provider, awsInstances bool, rdsInstancesConfig, rdsClustersConfig string, reEvalSec int) {
	t.Helper()

	if pc.GetId() != DefaultProviderID {
		t.Errorf("pc.GetId()=%s, wanted=%s", pc.GetId(), DefaultProviderID)
	}
	c := pc.GetAwsConfig()

	if !awsInstances {
		if c.GetEc2Instances() != nil {
			t.Errorf("c.GetEc2Instances()=%v, wanted=nil", c.GetEc2Instances())
		}
	} else {
		if c.GetEc2Instances() == nil {
			t.Fatal("c.GetEc2Instances() is nil, wanted=not-nil")
		}
		if c.GetEc2Instances().GetReEvalSec() != int32(reEvalSec) {
			t.Errorf("AWS instance reEvalSec=%d, wanted=%d", c.GetEc2Instances().GetReEvalSec(), reEvalSec)
		}
	}

	// Verify that RDS Clusters config is set correctly.
	if rdsClustersConfig == "" {
		if c.GetRdsClusters() != nil {
			t.Errorf("c.GetRdsClusters()=%v, wanted=nil", c.GetRdsClusters())
		}
	} else {
		if c.GetRdsClusters() == nil {
			t.Fatalf("c.GetRdsClusters()=nil, wanted=not-nil")
		}
		if c.GetRdsClusters().GetReEvalSec() != int32(reEvalSec) {
			t.Errorf("RDSClusters config reEvalSec=%d, wanted=%d", c.GetRdsClusters().GetReEvalSec(), reEvalSec)
		}
	}

	// Verify that RDS Instances config is set correctly.
	if rdsInstancesConfig == "" {
		if c.GetRdsInstances() != nil {
			t.Errorf("c.GetRdsInstances()=%v, wanted=nil", c.GetRdsInstances())
		}
	} else {
		if c.GetRdsInstances() == nil {
			t.Fatalf("c.GetRdsInstances()=nil, wanted=not-nil")
		}
		if c.GetRdsInstances().GetReEvalSec() != int32(reEvalSec) {
			t.Errorf("RDSInstances config reEvalSec=%d, wanted=%d", c.GetRdsInstances().GetReEvalSec(), reEvalSec)
		}
	}

}

func TestDefaultProviderConfig(t *testing.T) {
	tests := []struct {
		name           string
		resTypes       map[string]string
		reEvalSec      int
		wantEC2        bool
		wantRDSInst    string
		wantRDSCluster string
	}{
		{
			name: "EC2 instances only",
			resTypes: map[string]string{
				ResourceTypes.EC2Instances: "",
			},
			reEvalSec:      10,
			wantEC2:        true,
			wantRDSInst:    "",
			wantRDSCluster: "",
		},
		{
			name: "RDS instances and clusters",
			resTypes: map[string]string{
				ResourceTypes.RDSClusters:  "rds_clusters",
				ResourceTypes.RDSInstances: "rds_instances",
			},
			reEvalSec:      10,
			wantEC2:        false,
			wantRDSInst:    "rds_instances",
			wantRDSCluster: "rds_clusters",
		},
		{
			name: "EC2 and RDS instances",
			resTypes: map[string]string{
				ResourceTypes.EC2Instances: "",
				ResourceTypes.RDSInstances: "rds_instances",
			},
			reEvalSec:      10,
			wantEC2:        true,
			wantRDSInst:    "rds_instances",
			wantRDSCluster: "",
		},
		{
			name: "all resource types",
			resTypes: map[string]string{
				ResourceTypes.EC2Instances: "",
				ResourceTypes.RDSInstances: "rds_instances",
				ResourceTypes.RDSClusters:  "rds_clusters",
			},
			reEvalSec:      20,
			wantEC2:        true,
			wantRDSInst:    "rds_instances",
			wantRDSCluster: "rds_clusters",
		},
		{
			name:           "empty resource types",
			resTypes:       map[string]string{},
			reEvalSec:      10,
			wantEC2:        false,
			wantRDSInst:    "",
			wantRDSCluster: "",
		},
		{
			name: "unknown resource type ignored",
			resTypes: map[string]string{
				ResourceTypes.EC2Instances: "",
				"unknown_resource":         "test",
			},
			reEvalSec:      15,
			wantEC2:        true,
			wantRDSInst:    "",
			wantRDSCluster: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DefaultProviderConfig(tt.resTypes, tt.reEvalSec)
			testAWSConfig(t, c, tt.wantEC2, tt.wantRDSInst, tt.wantRDSCluster, tt.reEvalSec)
		})
	}
}

func TestDefaultProviderConfigProviderID(t *testing.T) {
	resTypes := map[string]string{
		ResourceTypes.EC2Instances: "",
	}

	c := DefaultProviderConfig(resTypes, 10)

	if c.GetId() != DefaultProviderID {
		t.Errorf("DefaultProviderConfig().GetId() = %q, want %q", c.GetId(), DefaultProviderID)
	}

	if c.GetAwsConfig() == nil {
		t.Fatal("DefaultProviderConfig().GetAwsConfig() is nil, want non-nil")
	}
}

func TestResourceTypes(t *testing.T) {
	// Test that ResourceTypes constant values are as expected
	expectedTypes := map[string]string{
		"EC2Instances": "ec2_instances",
		"RDSClusters":  "rds_clusters",
		"RDSInstances": "rds_instances",
	}

	actualTypes := map[string]string{
		"EC2Instances": ResourceTypes.EC2Instances,
		"RDSClusters":  ResourceTypes.RDSClusters,
		"RDSInstances": ResourceTypes.RDSInstances,
	}

	if !reflect.DeepEqual(actualTypes, expectedTypes) {
		t.Errorf("ResourceTypes mismatch:\ngot:  %+v\nwant: %+v", actualTypes, expectedTypes)
	}
}

func TestConfigSetters(t *testing.T) {
	// Test that all resource types have corresponding config setters
	for _, resType := range []string{
		ResourceTypes.EC2Instances,
		ResourceTypes.RDSInstances,
		ResourceTypes.RDSClusters,
	} {
		if _, ok := resourceConfigSetters[resType]; !ok {
			t.Errorf("missing config setter for resource type %q", resType)
		}
	}

	// Test that unknown resource types don't have setters
	unknownType := "unknown_resource_type"
	if _, ok := resourceConfigSetters[unknownType]; ok {
		t.Errorf("unexpected config setter found for unknown resource type %q", unknownType)
	}
}
