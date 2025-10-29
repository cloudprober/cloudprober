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
	resTypes := map[string]string{
		ResourceTypes.EC2Instances: "",
	}

	c := DefaultProviderConfig(resTypes, 10)
	testAWSConfig(t, c, true, "", "", 10)

	// RDS instances and clusters
	testRDSInstancesConfig := "rds_instances"
	testRDSClustersConfig := "rds_clusters"

	resTypes = map[string]string{
		ResourceTypes.RDSClusters:  testRDSClustersConfig,
		ResourceTypes.RDSInstances: testRDSInstancesConfig,
	}
	c = DefaultProviderConfig(resTypes, 10)
	testAWSConfig(t, c, false, testRDSInstancesConfig, testRDSClustersConfig, 10)

	// EC2 and RDS instances
	resTypes = map[string]string{
		ResourceTypes.EC2Instances: "",
		ResourceTypes.RDSInstances: testRDSInstancesConfig,
	}
	c = DefaultProviderConfig(resTypes, 10)
	testAWSConfig(t, c, true, testRDSInstancesConfig, "", 10)
}
