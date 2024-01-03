package aws

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/protobuf/proto"
)

type mockECClusterCache struct {
	output elasticache.DescribeCacheClustersOutput
	err    error
}

func (m mockECClusterCache) DescribeCacheClusters(ctx context.Context, params *elasticache.DescribeCacheClustersInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheClustersOutput, error) {
	return &m.output, m.err
}

type testECCluster struct {
	instances []testECClusterInstance
	id        string
}

type testECClusterInstance struct {
	id     string
	ipAddr string
	port   int32
	tags   map[string]string
	engine string
}

func TestECClusterExpand(t *testing.T) {
	cases := []struct {
		err         error
		cluster     *testECCluster
		expectCount int
	}{

		{
			cluster: &testECCluster{
				id: "test-cluster-id",

				instances: []testECClusterInstance{
					{
						id: "test-id",
					},
					{
						id: "test-id-2",
					},
				},
			},
			err:         nil,
			expectCount: 1,
		},
	}

	for i, tt := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			mock := mockECClusterCache{
				err: tt.err,
			}

			tlsenabled := false
			engine := "redis"

			if tt.cluster != nil {
				mock.output = elasticache.DescribeCacheClustersOutput{
					CacheClusters: []types.CacheCluster{
						{
							CacheClusterId:           &tt.cluster.id,
							TransitEncryptionEnabled: &tlsenabled,
							Engine:                   &engine,
						},
					},
				}
				for _, v := range tt.cluster.instances {
					c := types.CacheNode{
						Endpoint: &types.Endpoint{
							Address: &v.ipAddr,
							Port:    &v.port,
						},
					}
					mock.output.CacheClusters[0].CacheNodes = append(mock.output.CacheClusters[0].CacheNodes, c)
				}
			}

			il := &elastiCacheClusterLister{
				client:       mock,
				tagclient:    &elasticache.Client{},
				cacheList:    make(map[string]*ecClusterLocalCacheData),
				discoverTags: false, // tag discovery to be tested once the client can be mocked
			}
			il.expand(time.Second)

			// Check for instance count
			if len(il.cacheList) != tt.expectCount {
				t.Errorf("Got %d instances, want %d", len(il.cacheList), tt.expectCount)
			}
		})
	}
}

func TestECClusterLister(t *testing.T) {
	cases := []struct {
		instances     []*testECClusterInstance
		filter        []*pb.Filter
		expectErr     bool
		expectedCount int
	}{
		{
			instances: []*testECClusterInstance{
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
			instances: []*testECClusterInstance{
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
			instances:     []*testECClusterInstance{},
			expectedCount: 0,
		},
		{
			instances: []*testECClusterInstance{
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
			instances: []*testECClusterInstance{
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
			instances: []*testECClusterInstance{
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
			instances: []*testECClusterInstance{
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
			cache := make(map[string]*ecClusterLocalCacheData)
			for _, ti := range tt.instances {
				ci := &ecClusterInfo{
					ID:     ti.id,
					IP:     ti.ipAddr,
					Tags:   ti.tags,
					Engine: ti.engine,
				}
				cache[ti.id] = &ecClusterLocalCacheData{
					ci: ci,
				}

				names = append(names, ti.id)
			}

			lister := &elastiCacheClusterLister{
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
