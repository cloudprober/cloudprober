package aws

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/protobuf/proto"
)

type mockECCache struct {
	ccoutput elasticache.DescribeCacheClustersOutput
	rgoutput elasticache.DescribeReplicationGroupsOutput
	ccerr    error
	rgerr    error
}

func (m mockECCache) DescribeCacheClusters(ctx context.Context, params *elasticache.DescribeCacheClustersInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheClustersOutput, error) {
	return &m.ccoutput, m.ccerr
}

func (m mockECCache) DescribeReplicationGroups(ctx context.Context, params *elasticache.DescribeReplicationGroupsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeReplicationGroupsOutput, error) {
	return &m.rgoutput, m.rgerr
}

type testECCluster struct {
	instances []testECInstance
	id        string
}

type testECInstance struct {
	id     string
	ipAddr string
	port   int32
	tags   map[string]string
	engine string
}

func TestECExpand(t *testing.T) {
	cases := []struct {
		rgerr       error
		rg          *testECCluster
		ccerr       error
		cc          *testECCluster
		expectCount int
	}{
		{
			rg: &testECCluster{
				id: "test-cluster-id",
				instances: []testECInstance{
					{
						id:     "test-id",
						ipAddr: "10.0.0.2",
						port:   1000,
						tags:   map[string]string{"a": "b"},
					},
					{
						id:     "test-id-2",
						ipAddr: "10.0.0.3",
						port:   1000,
						tags:   map[string]string{"a": "b"},
					},
				},
			},
			rgerr:       nil,
			cc:          &testECCluster{},
			ccerr:       nil,
			expectCount: 1,
		},
		{
			rg:          &testECCluster{},
			rgerr:       nil,
			expectCount: 0,
		},
		{
			rg:          &testECCluster{},
			rgerr:       fmt.Errorf("some error"),
			expectCount: 0,
		},
		{
			cc: &testECCluster{
				id: "test-cluster-id",

				instances: []testECInstance{
					{
						id: "test-id",
					},
					{
						id: "test-id-2",
					},
				},
			},
			rgerr:       nil,
			ccerr:       nil,
			expectCount: 1,
		},
	}

	for i, tt := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			mock := mockECCache{
				rgerr: tt.rgerr,
				ccerr: tt.ccerr,
			}

			tlsenabled := false
			engine := "redis"

			if tt.cc != nil {
				mock.ccoutput = elasticache.DescribeCacheClustersOutput{
					CacheClusters: []types.CacheCluster{
						{
							CacheClusterId:           &tt.cc.id,
							TransitEncryptionEnabled: &tlsenabled,
							Engine:                   &engine,
						},
					},
				}
				for _, v := range tt.cc.instances {
					c := types.CacheNode{
						Endpoint: &types.Endpoint{
							Address: &v.ipAddr,
							Port:    &v.port,
						},
					}
					mock.ccoutput.CacheClusters[0].CacheNodes = append(mock.ccoutput.CacheClusters[0].CacheNodes, c)
				}
			}

			if tt.rg != nil {
				mock.rgoutput = elasticache.DescribeReplicationGroupsOutput{}
				for _, v := range tt.rg.instances {
					g := types.ReplicationGroup{
						ReplicationGroupId:       &v.id,
						TransitEncryptionEnabled: &tlsenabled,
						ConfigurationEndpoint: &types.Endpoint{
							Address: &v.ipAddr,
							Port:    &v.port,
						},
					}
					mock.rgoutput.ReplicationGroups = append(mock.rgoutput.ReplicationGroups, g)
				}
			}

			il := &elastiCacheLister{
				clusterclient: mock,
				rgclient:      mock,
				tagclient:     &elasticache.Client{},
				cacheList:     make(map[string]*cacheData),
				discoverTags:  false, // tag discovery to be tested once the client can be mocked
			}
			il.expand(time.Second)

			// Check for instance count
			if len(il.cacheList) != tt.expectCount {
				t.Errorf("Got %d instances, want %d", len(il.cacheList), tt.expectCount)
			}
		})
	}
}

func TestECLister(t *testing.T) {
	cases := []struct {
		instances     []*testECInstance
		filter        []*pb.Filter
		expectErr     bool
		expectedCount int
	}{
		{
			instances: []*testECInstance{
				{
					id:     "test-id",
					ipAddr: "10.0.0.2",
					tags:   map[string]string{"a": "b"},
				},
				{
					id:     "test-id-2",
					ipAddr: "10.0.0.3",
					tags:   map[string]string{"a": "b"},
				},
			},
			filter: []*pb.Filter{
				{
					Key:   proto.String("labels.a"),
					Value: proto.String("b"),
				},
			},
			expectedCount: 2,
		},
		{
			instances: []*testECInstance{
				{
					id:     "test-id",
					ipAddr: "10.0.0.2",
					tags:   map[string]string{"a": "b"},
				},
				{
					id:     "test-id-2",
					ipAddr: "10.0.0.3",
					tags:   map[string]string{"a": "b"},
				},
			},
			filter: []*pb.Filter{
				{
					Key:   proto.String("ins."),
					Value: proto.String("b"),
				},
			},
			expectErr: true,
		},
		{
			instances:     []*testECInstance{},
			expectedCount: 0,
		},
		{
			instances: []*testECInstance{
				{
					id:     "test-id",
					ipAddr: "10.0.0.2",
					tags:   map[string]string{"test1": "a"},
				},
				{
					id:     "test-id-2",
					ipAddr: "10.0.0.3",
					tags:   map[string]string{"test2": "b"},
				},
			},
			filter: []*pb.Filter{
				{
					Key:   proto.String("labels.a"),
					Value: proto.String("b"),
				},
			},
			expectedCount: 0,
		},
		{
			instances: []*testECInstance{
				{
					id:     "test-id",
					ipAddr: "10.0.0.2",
					tags:   map[string]string{"a": "b"},
				},
				{
					id:     "test-id-2",
					ipAddr: "10.0.0.3",
					tags:   map[string]string{"a": "b"},
				},
			},
			filter: []*pb.Filter{
				{
					Key:   proto.String("name"),
					Value: proto.String("nonexistent"),
				},
			},
			expectedCount: 0,
		},
		{
			instances: []*testECInstance{
				{
					id:     "test-id",
					ipAddr: "10.0.0.2",
					tags:   map[string]string{"a": "b"},
					engine: "memcached",
				},
				{
					id:     "test-id-2",
					ipAddr: "10.0.0.3",
					tags:   map[string]string{"a": "b"},
					engine: "memcached",
				},
			},
			filter: []*pb.Filter{
				{
					Key:   proto.String("engine"),
					Value: proto.String("redis"),
				},
			},
			expectedCount: 0,
		},
		{
			instances: []*testECInstance{
				{
					id:     "test-id",
					ipAddr: "10.0.0.2",
					tags:   map[string]string{"a": "b"},
					engine: "redis",
				},
				{
					id:     "test-id-2",
					ipAddr: "10.0.0.3",
					tags:   map[string]string{"a": "b"},
					engine: "redis",
				},
			},
			filter: []*pb.Filter{
				{
					Key:   proto.String("engine"),
					Value: proto.String("redis"),
				},
			},
			expectedCount: 2,
		},
	}

	for i, tt := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {

			names := []string{}
			cache := make(map[string]*cacheData)
			for _, ti := range tt.instances {
				ci := &cacheInfo{
					ID:     ti.id,
					Ip:     ti.ipAddr,
					Tags:   ti.tags,
					Engine: ti.engine,
				}
				cache[ti.id] = &cacheData{
					ci: ci,
				}

				names = append(names, ti.id)
			}

			lister := &elastiCacheLister{
				cacheList: cache,
				names:     names,
				l:         &logger.Logger{},
			}

			var filters []*pb.Filter
			if tt.filter != nil {
				filters = append(filters, tt.filter...)
			}

			resources, err := lister.listResources(&pb.ListResourcesRequest{
				Filter: filters,
			})

			if err != nil {
				if !tt.expectErr {
					t.Errorf("Got error while listing resources: %v, expected no errors", err)
				}
				return
			}

			if len(resources) != tt.expectedCount {
				t.Errorf("Got wrong number of targets. Expected: %d, Got: %d", tt.expectedCount, len(resources))
			}
		})
	}
}
