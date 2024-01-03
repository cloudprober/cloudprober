package aws

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/protobuf/proto"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

type mockRDSDescribeDBClustersAPIClient func(context.Context, *rds.DescribeDBClustersInput, ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error)

func (m mockRDSDescribeDBClustersAPIClient) DescribeDBClusters(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error) {
	return m(ctx, params, optFns...)
}

type testRDSClusters struct {
	id     string
	name   string
	ipAddr string
	port   int32
	tags   map[string]string
}

func TestRDSClustersExpand(t *testing.T) {
	cases := []struct {
		err         error
		instances   []*testRDSClusters
		expectCount int
	}{
		{
			instances: []*testRDSClusters{
				{
					id:     "rds-test-id",
					name:   "rds-test-cluster",
					ipAddr: "10.0.0.2",
					port:   5431,
					tags:   map[string]string{"a": "b"},
				},
				{
					id:     "rds-test-id-2",
					name:   "rds-test-cluster-2",
					ipAddr: "10.0.0.3",
					port:   5431,
					tags:   map[string]string{"a": "b"},
				},
			},
			err:         nil,
			expectCount: 2,
		},
		{
			instances:   []*testRDSClusters{},
			err:         nil,
			expectCount: 0,
		},
		{
			instances: []*testRDSClusters{
				{
					id: "rds-test-id",
				},
				{
					id: "rds-test-id-2",
				},
			},
			err:         nil,
			expectCount: 0,
		},
		{
			instances:   []*testRDSClusters{},
			err:         fmt.Errorf("some rds error"),
			expectCount: 0,
		},
	}

	for i, tt := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			client := func(t *testing.T, instances []*testRDSClusters) rds.DescribeDBClustersAPIClient {
				return mockRDSDescribeDBClustersAPIClient(func(ctx context.Context, params *rds.DescribeDBClustersInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClustersOutput, error) {
					t.Helper()

					out := &rds.DescribeDBClustersOutput{}

					for _, v := range instances {
						c := types.DBCluster{
							DBClusterIdentifier: &v.id,
							DatabaseName:        &v.name,
						}

						if v.ipAddr != "" {
							c.Endpoint = &v.ipAddr
						}

						if v.port > 0 {
							c.Port = &v.port
						}

						for tk, tv := range v.tags {
							tag := types.Tag{
								Key:   &tk,
								Value: &tv,
							}

							c.TagList = append(c.TagList, tag)
						}
						out.DBClusters = append(out.DBClusters, c)
					}

					return out, tt.err
				})
			}

			il := &rdsClustersLister{
				client:         client(t, tt.instances),
				dbClustersList: make(map[string]*rdsClusterData),
			}

			il.expand(time.Second)

			// Check for instance count
			if len(il.dbClustersList) != tt.expectCount {
				t.Errorf("Got %d instances, want %d", len(il.dbClustersList), tt.expectCount)
			}
		})
	}
}

func TestRDSClustersLister(t *testing.T) {
	cases := []struct {
		instances     []*testRDSClusters
		filter        []*pb.Filter
		expectErr     bool
		expectedCount int
	}{
		{
			instances: []*testRDSClusters{
				{
					id:     "rds-cluster-test-id",
					ipAddr: "10.0.0.2",
					tags:   map[string]string{"a": "b"},
				},
				{
					id:     "rds-cluster-test-id-2",
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
			instances: []*testRDSClusters{
				{
					id:     "rds-cluster-test-id",
					ipAddr: "10.0.0.2",
					tags:   map[string]string{"a": "b"},
				},
				{
					id:     "rds-cluster-test-id-2",
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
			instances:     []*testRDSClusters{},
			expectedCount: 0,
		},
		{
			instances: []*testRDSClusters{
				{
					id:     "rds-test-id",
					ipAddr: "10.0.0.2",
					tags:   map[string]string{"test1": "a"},
				},
				{
					id:     "rds-test-id-2",
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
			instances: []*testRDSClusters{
				{
					id:     "rds-test-id",
					ipAddr: "10.0.0.2",
					tags:   map[string]string{"a": "b"},
				},
				{
					id:     "rds-test-id-2",
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
	}

	for i, tt := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {

			names := []string{}
			cache := make(map[string]*rdsClusterData)
			for _, ti := range tt.instances {
				ii := &rdsClusterInfo{
					Name: ti.id,
					Ip:   ti.ipAddr,
					Tags: ti.tags,
				}
				cache[ti.id] = &rdsClusterData{
					ri: ii,
				}

				names = append(names, ti.id)
			}

			lister := &rdsClustersLister{
				dbClustersList: cache,
				names:          names,
				l:              &logger.Logger{},
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
