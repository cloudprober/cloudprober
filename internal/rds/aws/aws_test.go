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

func testAWSConfig(t *testing.T, pc *serverconfigpb.Provider, awsInstances bool, rdsConfig, elasticCachesConfig string, reEvalSec int) {
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

	// Verify that RDS config is set correctly.
	if rdsConfig == "" {
		if c.GetRds() != nil {
			t.Errorf("c.GetRds()=%v, wanted=nil", c.GetRds())
		}
	} else {
		if c.GetRds() == nil {
			t.Fatalf("c.GetRds()=nil, wanted=not-nil")
		}
		if c.GetRds().GetReEvalSec() != int32(reEvalSec) {
			t.Errorf("RDS config reEvalSec=%d, wanted=%d", c.GetRds().GetReEvalSec(), reEvalSec)
		}
	}

	// Verify that Elasticache is set correctly.
	if elasticCachesConfig == "" {
		if c.GetElasticaches() != nil {
			t.Errorf("c.GetElasticaches()=%v, wanted=nil", c.GetElasticaches())
		}
	} else {
		if c.GetElasticaches() == nil {
			t.Fatalf("c.GetElasticaches()=nil, wanted=not-nil")
		}
		if c.GetElasticaches().GetReEvalSec() != int32(reEvalSec) {
			t.Errorf("Elasticaches config reEvalSec=%d, wanted=%d", c.GetElasticaches().GetReEvalSec(), reEvalSec)
		}
	}
}

func TestDefaultProviderConfig(t *testing.T) {
	resTypes := map[string]string{
		ResourceTypes.EC2Instances: "",
	}

	c := DefaultProviderConfig(resTypes, 10)
	testAWSConfig(t, c, true, "", "", 10)

	// Elasticache and RDS
	testElastiCacheConfig := "elasticaches"
	testRDSConfig := "rds"

	resTypes = map[string]string{
		ResourceTypes.ElastiCaches: testElastiCacheConfig,
		ResourceTypes.RDS:          testRDSConfig,
	}
	c = DefaultProviderConfig(resTypes, 10)
	testAWSConfig(t, c, false, testRDSConfig, testElastiCacheConfig, 10)

	// EC2 instances, RTC and pub-sub
	resTypes = map[string]string{
		ResourceTypes.EC2Instances: "",
		ResourceTypes.ElastiCaches: testElastiCacheConfig,
		ResourceTypes.RDS:          testRDSConfig,
	}
	c = DefaultProviderConfig(resTypes, 10)
	testAWSConfig(t, c, true, testRDSConfig, testElastiCacheConfig, 10)
}
