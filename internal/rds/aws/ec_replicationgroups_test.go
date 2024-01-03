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

type mockECRGCache struct {
	output elasticache.DescribeReplicationGroupsOutput
	err    error
}

func (m mockECRGCache) DescribeReplicationGroups(ctx context.Context, params *elasticache.DescribeReplicationGroupsInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeReplicationGroupsOutput, error) {
	return &m.output, m.err
}

type testECReplicationGroup struct {
	instances []testECRGInstance
	id        string
}

type testECRGInstance struct {
	id     string
	ipAddr string
	port   int32
	tags   map[string]string
	engine string
}

func TestECRGExpand(t *testing.T) {
	cases := []struct {
		err         error
		group       *testECReplicationGroup
		expectCount int
	}{
		{
			group: &testECReplicationGroup{
				id: "test-cluster-id",
				instances: []testECRGInstance{
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
			err:         nil,
			expectCount: 1,
		},
		{
			group:       &testECReplicationGroup{},
			err:         nil,
			expectCount: 0,
		},
		{
			group:       &testECReplicationGroup{},
			err:         fmt.Errorf("some error"),
			expectCount: 0,
		},
	}

	for i, tt := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			mock := mockECRGCache{
				err: tt.err,
			}

			tlsenabled := false

			if tt.group != nil {
				mock.output = elasticache.DescribeReplicationGroupsOutput{}
				for _, v := range tt.group.instances {
					g := types.ReplicationGroup{
						ReplicationGroupId:       &v.id,
						TransitEncryptionEnabled: &tlsenabled,
						ConfigurationEndpoint: &types.Endpoint{
							Address: &v.ipAddr,
							Port:    &v.port,
						},
					}
					mock.output.ReplicationGroups = append(mock.output.ReplicationGroups, g)
				}
			}

			il := &elastiCacheRGLister{
				client:       mock,
				tagclient:    &elasticache.Client{},
				cacheList:    make(map[string]*ecReplicationGroupCacheData),
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

func TestECLister(t *testing.T) {
	cases := []struct {
		instances     []*testECRGInstance
		filter        []*pb.Filter
		expectErr     bool
		expectedCount int
	}{
		{
			instances: []*testECRGInstance{
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
			instances: []*testECRGInstance{
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
			instances:     []*testECRGInstance{},
			expectedCount: 0,
		},
		{
			instances: []*testECRGInstance{
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
			instances: []*testECRGInstance{
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
			instances: []*testECRGInstance{
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
			instances: []*testECRGInstance{
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
			cache := make(map[string]*ecReplicationGroupCacheData)
			for _, ti := range tt.instances {
				ci := &ecReplicationGroupInfo{
					ID:     ti.id,
					IP:     ti.ipAddr,
					Tags:   ti.tags,
					Engine: ti.engine,
				}
				cache[ti.id] = &ecReplicationGroupCacheData{
					ci: ci,
				}

				names = append(names, ti.id)
			}

			lister := &elastiCacheRGLister{
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
