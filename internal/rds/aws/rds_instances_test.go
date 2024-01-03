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

type mockRDSDescribeDBInstancesAPIClient func(context.Context, *rds.DescribeDBInstancesInput, ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error)

func (m mockRDSDescribeDBInstancesAPIClient) DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
	return m(ctx, params, optFns...)
}

type testRDSInstances struct {
	name      string
	ipAddr    string
	port      int32
	isReplica bool
	tags      map[string]string
}

func TestRDSInstancesExpand(t *testing.T) {
	cases := []struct {
		err         error
		instances   []*testRDSInstances
		expectCount int
	}{
		{
			instances: []*testRDSInstances{
				{
					name:   "rds-test-instance",
					ipAddr: "10.0.0.2",
					port:   5431,
					tags:   map[string]string{"a": "b"},
				},
			},
			err:         nil,
			expectCount: 1,
		},
		{
			instances: []*testRDSInstances{
				{
					name:   "rds-test-instance",
					ipAddr: "10.0.0.2",
					port:   5431,
					tags:   map[string]string{"a": "b"},
				},
				{
					name:   "rds-test-instance-2",
					ipAddr: "10.0.0.3",
					port:   5431,
					tags:   map[string]string{"a": "b"},
				},
			},
			err:         nil,
			expectCount: 2,
		},
		{
			instances:   []*testRDSInstances{},
			err:         nil,
			expectCount: 0,
		},
		{
			instances: []*testRDSInstances{
				{
					name: "rds-test-name",
				},
				{
					name: "rds-test-name-2",
				},
			},
			err:         nil,
			expectCount: 0,
		},
		{
			instances:   []*testRDSInstances{},
			err:         fmt.Errorf("some rds error"),
			expectCount: 0,
		},
	}

	for i, tt := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			client := func(t *testing.T, instances []*testRDSInstances) rds.DescribeDBInstancesAPIClient {
				return mockRDSDescribeDBInstancesAPIClient(func(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
					t.Helper()

					out := &rds.DescribeDBInstancesOutput{}

					for _, v := range instances {
						c := types.DBInstance{
							DBInstanceIdentifier: &v.name,
							DBName:               &v.name,
						}

						if v.ipAddr != "" {
							c.Endpoint = &types.Endpoint{
								Address: &v.ipAddr,
								Port:    &v.port,
							}
						}

						for tk, tv := range v.tags {
							tag := types.Tag{
								Key:   &tk,
								Value: &tv,
							}

							c.TagList = append(c.TagList, tag)
						}
						out.DBInstances = append(out.DBInstances, c)
					}

					return out, tt.err
				})
			}

			il := &rdsInstancesLister{
				client:          client(t, tt.instances),
				dbInstancesList: make(map[string]*rdsInstanceData),
			}

			il.expand(time.Second)

			// Check for instance count
			if len(il.dbInstancesList) != tt.expectCount {
				t.Errorf("Got %d instances, want %d", len(il.dbInstancesList), tt.expectCount)
			}
		})
	}
}

func TestRDSInstancesLister(t *testing.T) {
	cases := []struct {
		instances     []*testRDSInstances
		filter        []*pb.Filter
		expectErr     bool
		expectedCount int
	}{
		{
			instances: []*testRDSInstances{
				{
					name:   "rds-cluster-test-id",
					ipAddr: "10.0.0.2",
					tags:   map[string]string{"a": "b"},
				},
				{
					name:   "rds-cluster-test-id-2",
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
			instances: []*testRDSInstances{
				{
					name:   "rds-instance-test-name",
					ipAddr: "10.0.0.2",
					tags:   map[string]string{"a": "b"},
				},
				{
					name:   "rds-instance-test-name-2",
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
			instances:     []*testRDSInstances{},
			expectedCount: 0,
		},
		{
			instances: []*testRDSInstances{
				{
					name:   "rds-test-name",
					ipAddr: "10.0.0.2",
					tags:   map[string]string{"test1": "a"},
				},
				{
					name:   "rds-test-name-2",
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
			instances: []*testRDSInstances{
				{
					name:   "rds-test-name",
					ipAddr: "10.0.0.2",
					tags:   map[string]string{"a": "b"},
				},
				{
					name:   "rds-test-name-2",
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
			cache := make(map[string]*rdsInstanceData)
			for _, ti := range tt.instances {
				ii := &rdsInstanceInfo{
					Name: ti.name,
					Ip:   ti.ipAddr,
					Tags: ti.tags,
				}
				cache[ti.name] = &rdsInstanceData{
					ri: ii,
				}

				names = append(names, ti.name)
			}

			lister := &rdsInstancesLister{
				dbInstancesList: cache,
				names:           names,
				l:               &logger.Logger{},
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
