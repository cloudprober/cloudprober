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
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"google.golang.org/protobuf/proto"
)

// elastiCacheClusterLister represents replication group items that we fetch from the elasticache API.
type ecReplicationGroupInfo struct {
	ID         string
	IP         string
	Port       int32
	TLSEnabled bool
	Clustered  bool
	Engine     string
	Tags       map[string]string
}

// ecReplicationGroupCacheData represents objects that we store in the local cache.
type ecReplicationGroupCacheData struct {
	ci          *ecReplicationGroupInfo
	lastUpdated int64
}

/*
ElastiCacheRGFilters defines filters supported by the ec_replicationgroups_instances resource
type.

	 Example:
	 filter {
		 key: "name"
		 value: "service.*"
	 }
	 filter {
		 key: "engine"
		 value: "redis"
	 }
	 filter {
		 key: "labels.app"
		 value: "service-a"
	 }
*/

var ElastiCacheRGFilters = struct {
	RegexFilterKeys []string
	LabelsFilter    bool
}{
	[]string{"name", "engine"},
	true,
}

// elastiCacheRGLister is a AWS ElastiCache replication group lister. It implements a cache,
// that's populated at a regular interval by making the AWS API calls.
// Listing actually only returns the current contents of that cache.
type elastiCacheRGLister struct {
	c         *configpb.ElastiCacheReplicationGroups
	client    elasticache.DescribeReplicationGroupsAPIClient
	tagclient *elasticache.Client
	l         *logger.Logger
	mu        sync.RWMutex
	names     []string
	cacheList map[string]*ecReplicationGroupCacheData
	// This is for unit testing, should be taken out if/when there is a respective
	// interface in AWS SDK for go v2 to replace this logs
	discoverTags bool
}

// listResources returns the list of resource records, where each record
// consists of an cluster name and the endpoint associated with it.
func (cl *elastiCacheRGLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	var resources []*pb.Resource

	allFilters, err := filter.ParseFilters(req.GetFilter(), ElastiCacheRGFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, err
	}

	nameFilter, engineFilter, labelsFilter := allFilters.RegexFilters["name"], allFilters.RegexFilters["engine"], allFilters.LabelsFilter

	cl.mu.RLock()
	defer cl.mu.RUnlock()

	for _, name := range cl.names {
		ins := cl.cacheList[name].ci
		if ins == nil {
			cl.l.Errorf("ec_replicationgroups: cached info missing for %s", name)
			continue
		}

		if nameFilter != nil && !nameFilter.Match(name, cl.l) {
			continue
		}
		if labelsFilter != nil && !labelsFilter.Match(ins.Tags, cl.l) {
			continue
		}

		if engineFilter != nil && !engineFilter.Match(ins.Engine, cl.l) {
			continue
		}

		var info string
		if ins.Clustered {
			info = "clustered"
		}
		resources = append(resources, &pb.Resource{
			Id:          proto.String(ins.ID),
			Name:        proto.String(name),
			Ip:          proto.String(ins.IP),
			Port:        proto.Int32(ins.Port),
			Labels:      ins.Tags,
			LastUpdated: proto.Int64(cl.cacheList[name].lastUpdated),
			Info:        []byte(info),
		})
	}

	cl.l.Infof("ec_replicationgroups.listResources: returning %d instances", len(resources))
	return resources, nil
}

// expand runs equivalent API calls as "aws elasticache describe-replication-groups",
// and is used to populate the cache. More details about this API call can be found in
// https://docs.aws.amazon.com/AmazonElastiCache/latest/APIReference/API_DescribeReplicationGroups.html
func (cl *elastiCacheRGLister) expand(reEvalInterval time.Duration) {
	cl.l.Infof("ec_replicationgroups.expand: expanding AWS targets")

	resp, err := cl.client.DescribeReplicationGroups(context.TODO(), nil)
	if err != nil {
		cl.l.Errorf("ec_replicationgroups.expand: error while listing replication groups: %v", err)
		return
	}

	var ids = make([]string, 0)
	var cacheList = make(map[string]*ecReplicationGroupCacheData)
	ts := time.Now().Unix()

	for _, r := range resp.ReplicationGroups {
		var ci *ecReplicationGroupInfo
		if r.ConfigurationEndpoint != nil { //clustered
			ci = &ecReplicationGroupInfo{
				ID:         *r.ReplicationGroupId,
				IP:         *r.ConfigurationEndpoint.Address,
				Port:       *r.ConfigurationEndpoint.Port,
				TLSEnabled: *r.TransitEncryptionEnabled,
				Clustered:  true,
				Tags:       make(map[string]string),
			}
		} else if len(r.NodeGroups) > 0 && r.NodeGroups[0].PrimaryEndpoint != nil {
			ci = &ecReplicationGroupInfo{
				ID:         *r.ReplicationGroupId,
				IP:         *r.NodeGroups[0].PrimaryEndpoint.Address,
				Port:       *r.NodeGroups[0].PrimaryEndpoint.Port,
				TLSEnabled: *r.TransitEncryptionEnabled,
				Clustered:  false,
				Tags:       make(map[string]string),
			}
		} else {
			continue
		}

		if cl.discoverTags {
			// Same comments as the same calls above
			tagsResp, err := cl.tagclient.ListTagsForResource(context.TODO(), &elasticache.ListTagsForResourceInput{
				ResourceName: r.ARN,
			})
			if err != nil {
				cl.l.Errorf("ec_replicationgroups.expand: error getting tags for replication group %s: %v", *r.ReplicationGroupId, err)
				continue
			}

			// Convert to map
			for _, t := range tagsResp.TagList {
				ci.Tags[*t.Key] = *t.Value
			}
		}

		cacheList[*r.ReplicationGroupId] = &ecReplicationGroupCacheData{ci, ts}
		ids = append(ids, *r.ReplicationGroupId)
	}

	cl.mu.Lock()
	cl.names = ids
	cl.cacheList = cacheList
	cl.mu.Unlock()

	cl.l.Infof("ec_replicationgroups.expand: got %d caches", len(ids))
}

func newElastiCacheRGLister(c *configpb.ElastiCacheReplicationGroups, region string, l *logger.Logger) (*elastiCacheRGLister, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("AWS configuration error: %v", err)
	}

	client := elasticache.NewFromConfig(cfg)

	cl := &elastiCacheRGLister{
		c:            c,
		client:       client,
		tagclient:    client,
		cacheList:    make(map[string]*ecReplicationGroupCacheData),
		l:            l,
		discoverTags: true,
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
