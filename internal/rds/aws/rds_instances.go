package aws

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/rds/aws/proto"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/internal/rds/server/filter"
	"github.com/cloudprober/cloudprober/logger"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"google.golang.org/protobuf/proto"
)

// rdsInstanceInfo represents instance items that we fetch from the RDS API.
type rdsInstanceInfo struct {
	Name      string
	Ip        string
	Port      int32
	IsReplica bool
	Tags      map[string]string
}

// rdsData represents objects that we store in cache.
type rdsInstanceData struct {
	ri          *rdsInstanceInfo
	lastUpdated int64
}

var RDSInstancesFilters = struct {
	RegexFilterKeys []string
	LabelsFilter    bool
}{
	[]string{"name", "engine"},
	true,
}

// rdsInstancesLister is an AWS Relational Database Service Instances lister. It implements a cache,
// that's populated at a regular interval by making the AWS API calls.
// Listing actually only returns the current contents of that cache.
type rdsInstancesLister struct {
	c               *configpb.RDSInstances
	client          rds.DescribeDBInstancesAPIClient
	l               *logger.Logger
	mu              sync.RWMutex
	names           []string
	dbInstancesList map[string]*rdsInstanceData
}

// listResources returns the list of resource records, where each record
// consists of an cluster name and the endpoint associated with it.
func (rl *rdsInstancesLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	var resources []*pb.Resource

	allFilters, err := filter.ParseFilters(req.GetFilter(), RDSInstancesFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, err
	}

	nameFilter, labelsFilter := allFilters.RegexFilters["name"], allFilters.LabelsFilter

	rl.mu.RLock()
	defer rl.mu.RUnlock()

	for _, name := range rl.names {
		ins := rl.dbInstancesList[name].ri
		if ins == nil {
			rl.l.Errorf("rds_instances.listResources: db info missing for %s", name)
			continue
		}

		if nameFilter != nil && !nameFilter.Match(name, rl.l) {
			continue
		}
		if labelsFilter != nil && !labelsFilter.Match(ins.Tags, rl.l) {
			continue
		}

		resources = append(resources, &pb.Resource{
			Name:        proto.String(name),
			Ip:          proto.String(ins.Ip),
			Port:        proto.Int32(ins.Port),
			Labels:      ins.Tags,
			LastUpdated: proto.Int64(rl.dbInstancesList[name].lastUpdated),
		})
	}

	rl.l.Infof("rds_instances.listResources: returning %d instances", len(resources))
	return resources, nil
}

// expand runs equivalent API calls as "aws rds describe-db-instances",
// and is used to populate the cache. It returns the instance information
// for instances provisioned for RDS.
// More details can be found in
// https://docs.aws.amazon.com/AmazonRDS/latest/APIReference/API_DescribeDBInstances.html
func (rl *rdsInstancesLister) expand(reEvalInterval time.Duration) {
	rl.l.Infof("rds_instances.expand: expanding AWS targets")

	result, err := rl.client.DescribeDBInstances(context.TODO(), nil)
	if err != nil {
		rl.l.Errorf("rds_instances.expand: error while listing database instances: %v", err)
		return
	}

	var ids = make([]string, 0)
	var dbInstancesList = make(map[string]*rdsInstanceData)

	ts := time.Now().Unix()
	for _, r := range result.DBInstances {
		if r.DBInstanceIdentifier == nil || r.DBName == nil || r.Endpoint == nil {
			continue
		}
		isReplica := false
		if r.DBClusterIdentifier != nil || r.ReadReplicaSourceDBInstanceIdentifier != nil {
			isReplica = true
		}

		ci := &rdsInstanceInfo{
			Name:      *r.DBName,
			Ip:        *r.Endpoint.Address,
			Port:      *r.Endpoint.Port,
			IsReplica: isReplica,
			Tags:      make(map[string]string),
		}

		// Convert to map
		for _, t := range r.TagList {
			ci.Tags[*t.Key] = *t.Value
		}

		dbInstancesList[*r.DBName] = &rdsInstanceData{ci, ts}
		ids = append(ids, *r.DBName)
	}

	rl.mu.Lock()
	rl.names = ids
	rl.dbInstancesList = dbInstancesList
	rl.mu.Unlock()

	rl.l.Infof("rds_instances.expand: got %d databases", len(ids))
}

func newRdsInstancesLister(c *configpb.RDSInstances, region string, l *logger.Logger) (*rdsInstancesLister, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("AWS configuration error: %v", err)
	}

	client := rds.NewFromConfig(cfg)

	cl := &rdsInstancesLister{
		c:               c,
		client:          client,
		dbInstancesList: make(map[string]*rdsInstanceData),
		l:               l,
	}

	reEvalInterval := time.Duration(c.GetReEvalSec()) * time.Second
	go func() {
		cl.expand(0)
		// Introduce a random delay between 0-reEvalInterval before
		// starting the refresh loop. If there are multiple cloudprober
		// awsInstances, this will make sure that each instance calls AWS
		// API at a different point of time.
		randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
		time.Sleep(time.Duration(randomDelaySec) * time.Second)
		for range time.Tick(reEvalInterval) {
			cl.expand(reEvalInterval)
		}
	}()
	return cl, nil
}
