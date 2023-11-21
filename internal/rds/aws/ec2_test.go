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

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type mockEC2DescribeInstances func(context.Context, *ec2.DescribeInstancesInput, ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)

func (m mockEC2DescribeInstances) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return m(ctx, params, optFns...)
}

type testInstance struct {
	id     string
	ipAddr string
	tags   map[string]string
}

func TestExpand(t *testing.T) {
	cases := []struct {
		err         error
		instances   []*testInstance
		expectCount int
	}{
		{
			instances: []*testInstance{
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
			err:         nil,
			expectCount: 2,
		},
		{
			instances:   []*testInstance{},
			err:         nil,
			expectCount: 0,
		},
		{
			instances: []*testInstance{
				{
					id: "test-id",
				},
				{
					id: "test-id-2",
				},
			},
			err:         nil,
			expectCount: 0,
		},
		{
			instances:   []*testInstance{},
			err:         fmt.Errorf("some error"),
			expectCount: 0,
		},
	}

	for i, tt := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			client := func(t *testing.T, instances []*testInstance) ec2.DescribeInstancesAPIClient {
				return mockEC2DescribeInstances(func(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
					t.Helper()

					r := types.Reservation{}

					for _, v := range instances {
						i := types.Instance{
							InstanceId: &v.id,
						}

						if v.ipAddr != "" {
							i.PrivateIpAddress = &v.ipAddr
						}

						for tk, tv := range v.tags {
							tag := types.Tag{
								Key:   &tk,
								Value: &tv,
							}

							i.Tags = append(i.Tags, tag)

						}
						r.Instances = append(r.Instances, i)
					}

					out := &ec2.DescribeInstancesOutput{
						Reservations: []types.Reservation{r},
					}

					return out, tt.err
				})
			}

			il := &ec2InstancesLister{
				client: client(t, tt.instances),
				cache:  make(map[string]*instanceData),
			}

			il.expand(time.Second)

			// Check for instance count
			if len(il.cache) != tt.expectCount {
				t.Errorf("Got %d instances, want %d", len(il.cache), tt.expectCount)
			}
		})
	}
}

func TestLister(t *testing.T) {
	cases := []struct {
		instances     []*testInstance
		filter        []*pb.Filter
		expectErr     bool
		expectedCount int
	}{
		{
			instances: []*testInstance{
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
			instances: []*testInstance{
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
			instances:     []*testInstance{},
			expectedCount: 0,
		},
		{
			instances: []*testInstance{
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
			instances: []*testInstance{
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
	}

	for i, tt := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {

			names := []string{}
			cache := make(map[string]*instanceData)
			for _, ti := range tt.instances {
				ii := &instanceInfo{
					ID:     ti.id,
					IPAddr: ti.ipAddr,
					Tags:   ti.tags,
				}
				cache[ti.id] = &instanceData{
					ii: ii,
				}

				names = append(names, ti.id)
			}

			lister := &ec2InstancesLister{
				cache: cache,
				names: names,
				l:     &logger.Logger{},
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
