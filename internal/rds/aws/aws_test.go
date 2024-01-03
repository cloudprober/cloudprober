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

func testAWSConfig(t *testing.T, pc *serverconfigpb.Provider, awsInstances bool, rdsInstancesConfig, rdsClustersConfig, elasticCacheClustersConfig, elasticCacheRGConfig string, reEvalSec int) {
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
			t.Fatal("c.GetGceInstances() is nil, wanted=not-nil")
		}
		if c.GetEc2Instances().GetReEvalSec() != int32(reEvalSec) {
			t.Errorf("AWS instance reEvalSec=%d, wanted=%d", c.GetEc2Instances().GetReEvalSec(), reEvalSec)
		}
	}

	// Verify that ElastiCacheClusters is set correctly.
	if elasticCacheClustersConfig == "" {
		if c.GetElasticacheClusters() != nil {
			t.Errorf("c.GetElasticacheClusters()=%v, wanted=nil", c.GetElasticacheClusters())
		}
	} else {
		if c.GetElasticacheClusters() == nil {
			t.Fatalf("c.GetElasticaches()=nil, wanted=not-nil")
		}
		if c.GetElasticacheClusters().GetReEvalSec() != int32(reEvalSec) {
			t.Errorf("Elasticacheclusters config reEvalSec=%d, wanted=%d", c.GetElasticacheClusters().GetReEvalSec(), reEvalSec)
		}
	}

	// Verify that ElastiCacheReplicationGroups is set correctly.
	if elasticCacheRGConfig == "" {
		if c.GetElasticacheReplicationgroups() != nil {
			t.Errorf("c.GetElasticacheReplicationgroups()=%v, wanted=nil", c.GetElasticacheReplicationgroups())
		}
	} else {
		if c.GetElasticacheReplicationgroups() == nil {
			t.Fatalf("c.GetElasticacheReplicationgroups()=nil, wanted=not-nil")
		}
		if c.GetElasticacheReplicationgroups().GetReEvalSec() != int32(reEvalSec) {
			t.Errorf("Elasticachereplicationgroups config reEvalSec=%d, wanted=%d", c.GetElasticacheReplicationgroups().GetReEvalSec(), reEvalSec)
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
	testAWSConfig(t, c, true, "", "", "", "", 10)

	// Elasticache cluster, replication groups and RDS
	testElastiCacheClustersConfig := "elasticache_clusters"
	testElastiCacheReplicationGroupsConfig := "elasticache_replicationgroups"
	testRDSInstancesConfig := "rds_instances"
	testRDSClustersConfig := "rds_clusters"

	resTypes = map[string]string{
		ResourceTypes.ElastiCacheClusters:          testElastiCacheClustersConfig,
		ResourceTypes.ElastiCacheReplicationGroups: testElastiCacheReplicationGroupsConfig,
		ResourceTypes.RDSClusters:                  testRDSClustersConfig,
		ResourceTypes.RDSInstances:                 testRDSInstancesConfig,
	}
	c = DefaultProviderConfig(resTypes, 10)
	testAWSConfig(t, c, false, testRDSInstancesConfig, testRDSClustersConfig, testElastiCacheClustersConfig, testElastiCacheReplicationGroupsConfig, 10)

	// EC2 and RDS instances
	resTypes = map[string]string{
		ResourceTypes.EC2Instances:                 "",
		ResourceTypes.ElastiCacheReplicationGroups: testElastiCacheReplicationGroupsConfig,
		ResourceTypes.RDSInstances:                 testRDSInstancesConfig,
	}
	c = DefaultProviderConfig(resTypes, 10)
	testAWSConfig(t, c, true, testRDSInstancesConfig, "", "", testElastiCacheReplicationGroupsConfig, 10)
}
