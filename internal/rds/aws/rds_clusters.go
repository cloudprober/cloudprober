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

// rdsClusterInfo represents cluster items that we fetch from the RDS API.
type rdsClusterInfo struct {
	Name      string
	Ip        string
	Port      int32
	IsCluster bool
	Tags      map[string]string
}

// rdsClusterData represents objects that we store in cache.
type rdsClusterData struct {
	ri          *rdsClusterInfo
	lastUpdated int64
}

var RDSClustersFilters = struct {
	RegexFilterKeys []string
	LabelsFilter    bool
}{
	[]string{"name", "engine"},
	true,
}

// rdsClustersLister is a AWS Relational Database Service lister. It implements a cache,
// that's populated at a regular interval by making the AWS API calls.
// Listing actually only returns the current contents of that cache.
type rdsClustersLister struct {
	c              *configpb.RDSClusters
	client         rds.DescribeDBClustersAPIClient
	l              *logger.Logger
	mu             sync.RWMutex
	names          []string
	dbClustersList map[string]*rdsClusterData
}

// listResources returns the list of resource records, where each record
// consists of an cluster name and the endpoint associated with it.
func (rl *rdsClustersLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	var resources []*pb.Resource

	allFilters, err := filter.ParseFilters(req.GetFilter(), RDSClustersFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, err
	}

	nameFilter, labelsFilter := allFilters.RegexFilters["name"], allFilters.LabelsFilter

	rl.mu.RLock()
	defer rl.mu.RUnlock()

	for _, name := range rl.names {
		ins := rl.dbClustersList[name].ri
		if ins == nil {
			rl.l.Errorf("rds_clusters.listResources: db info missing for %s", name)
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
			LastUpdated: proto.Int64(rl.dbClustersList[name].lastUpdated),
		})
	}

	rl.l.Infof("rds_clusters.listResources: returning %d instances", len(resources))
	return resources, nil
}

// expand runs equivalent API calls as "aws rds describe-db-clusters",
// and is used to populate the cache. It is used to obtain RDS cluster information
// More details are available in
// https://docs.aws.amazon.com/AmazonRDS/latest/APIReference/API_DescribeDBClusters.html
func (rl *rdsClustersLister) expand(reEvalInterval time.Duration) {
	rl.l.Infof("rds_clusters.expand: expanding AWS targets")

	resCluster, err := rl.client.DescribeDBClusters(context.TODO(), nil)
	if err != nil {
		rl.l.Errorf("rds_clusters.expand: error while listing database clusters: %v", err)
		return
	}

	var ids = make([]string, 0)
	var dbList = make(map[string]*rdsClusterData)

	ts := time.Now().Unix()
	for _, d := range resCluster.DBClusters {
		if d.DBClusterIdentifier == nil || d.DatabaseName == nil || d.Endpoint == nil || d.Port == nil {
			continue
		}

		ci := &rdsClusterInfo{
			Name: *d.DBClusterIdentifier,
			Ip:   *d.Endpoint,
			Port: *d.Port,
			Tags: make(map[string]string),
		}

		// Convert to map
		for _, t := range d.TagList {
			ci.Tags[*t.Key] = *t.Value
		}

		dbList[*d.DBClusterIdentifier] = &rdsClusterData{ci, ts}
		ids = append(ids, *d.DBClusterIdentifier)
	}

	rl.mu.Lock()
	rl.names = ids
	rl.dbClustersList = dbList
	rl.mu.Unlock()

	rl.l.Infof("rds_clusters.expand: got %d databases", len(ids))
}

func newRdsClustersLister(c *configpb.RDSClusters, region string, l *logger.Logger) (*rdsClustersLister, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("AWS configuration error: %v", err)
	}

	client := rds.NewFromConfig(cfg)

	cl := &rdsClustersLister{
		c:              c,
		client:         client,
		dbClustersList: make(map[string]*rdsClusterData),
		l:              l,
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
