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

// cacheInfo represents instance items that we fetch from the elasticache API.
type cacheInfo struct {
	ID         string
	Ip         string
	Port       int32
	Clustered  bool
	TLSEnabled bool
	Engine     string
	Tags       map[string]string
}

// cacheData represents objects that we store in cache.
type cacheData struct {
	ci          *cacheInfo
	lastUpdated int64
}

/*
AWSInstancesFilters defines filters supported by the ec2_instances resource
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

var ElastiCacheFilters = struct {
	RegexFilterKeys []string
	LabelsFilter    bool
}{
	[]string{"name", "engine"},
	true,
}

// elastiCacheLister is a AWS ElastiCache cluster lister. It implements a cache,
// that's populated at a regular interval by making the AWS API calls.
// Listing actually only returns the current contents of that cache.
type elastiCacheLister struct {
	c             *configpb.ElastiCaches
	clusterclient elasticache.DescribeCacheClustersAPIClient
	rgclient      elasticache.DescribeReplicationGroupsAPIClient
	tagclient     *elasticache.Client
	l             *logger.Logger
	mu            sync.RWMutex
	names         []string
	cacheList     map[string]*cacheData
	// This is for unit testing, should be taken out if/when there is a respective
	// interface in AWS SDK for go v2 to replace this logs
	discoverTags bool
}

// listResources returns the list of resource records, where each record
// consists of an cluster name and the endpoint associated with it.
func (cl *elastiCacheLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	var resources []*pb.Resource

	allFilters, err := filter.ParseFilters(req.GetFilter(), ElastiCacheFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, err
	}

	nameFilter, engineFilter, labelsFilter := allFilters.RegexFilters["name"], allFilters.RegexFilters["engine"], allFilters.LabelsFilter

	cl.mu.RLock()
	defer cl.mu.RUnlock()

	for _, name := range cl.names {
		ins := cl.cacheList[name].ci
		if ins == nil {
			cl.l.Errorf("elacticaches: cached info missing for %s", name)
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

		resources = append(resources, &pb.Resource{
			Name:        proto.String(name),
			Ip:          proto.String(ins.Ip),
			Port:        proto.Int32(ins.Port),
			Labels:      ins.Tags,
			LastUpdated: proto.Int64(cl.cacheList[name].lastUpdated),
		})
	}

	cl.l.Infof("elasticaches.listResources: returning %d instances", len(resources))
	return resources, nil
}

// expand runs equivalent API calls as "aws describe-instances",
// and is used to populate the cache.
func (cl *elastiCacheLister) expand(reEvalInterval time.Duration) {
	cl.l.Infof("elasticaches.expand: expanding AWS targets")

	resCC, err := cl.clusterclient.DescribeCacheClusters(context.TODO(), nil)
	if err != nil {
		cl.l.Errorf("elasticaches.expand: error while listing cache clusters: %v", err)
		return
	}

	var ids = make([]string, 0)
	var cacheList = make(map[string]*cacheData)

	ts := time.Now().Unix()
	for _, r := range resCC.CacheClusters {
		if len(r.CacheNodes) == 0 {
			continue
		}
		ci := &cacheInfo{
			ID:         *r.CacheClusterId,
			TLSEnabled: *r.TransitEncryptionEnabled,
			Ip:         *r.CacheNodes[0].Endpoint.Address,
			Port:       *r.CacheNodes[0].Endpoint.Port,
			Engine:     *r.Engine,
			Clustered:  false,
			Tags:       make(map[string]string),
		}

		if cl.discoverTags {
			// AWS doesn't return Tag information in the response, we'll need to request it separately
			// NB: This might get throttled by AWS, if we make too many requests, see if we can batch or slow down
			// Add sleep if needed to the end of the loop
			tagsResp, err := cl.tagclient.ListTagsForResource(context.TODO(), &elasticache.ListTagsForResourceInput{
				ResourceName: r.ARN,
			})
			if err != nil {
				cl.l.Errorf("elasticaches.expand: error getting tags for cluster %s: %v", *r.CacheClusterId, err)
				continue
			}

			// Convert to map
			for _, t := range tagsResp.TagList {
				ci.Tags[*t.Key] = *t.Value
			}
		}

		cacheList[*r.CacheClusterId] = &cacheData{ci, ts}
		ids = append(ids, *r.CacheClusterId)
	}

	resRG, err := cl.rgclient.DescribeReplicationGroups(context.TODO(), nil)
	if err != nil {
		cl.l.Errorf("elasticaches.expand: error while listing replication groups: %v", err)
		return
	}
	for _, r := range resRG.ReplicationGroups {
		var ci *cacheInfo
		if r.ConfigurationEndpoint != nil { //clustered
			ci = &cacheInfo{
				ID:         *r.ReplicationGroupId,
				Ip:         *r.ConfigurationEndpoint.Address,
				Port:       *r.ConfigurationEndpoint.Port,
				TLSEnabled: *r.TransitEncryptionEnabled,
				Clustered:  true,
				Tags:       make(map[string]string),
			}
		} else if len(r.NodeGroups) > 0 && r.NodeGroups[0].PrimaryEndpoint != nil {
			ci = &cacheInfo{
				ID:         *r.ReplicationGroupId,
				Ip:         *r.NodeGroups[0].PrimaryEndpoint.Address,
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
				cl.l.Errorf("elasticaches.expand: error getting tags for replication group %s: %v", *r.ReplicationGroupId, err)
				continue
			}

			// Convert to map
			for _, t := range tagsResp.TagList {
				ci.Tags[*t.Key] = *t.Value
			}
		}

		cacheList[*r.ReplicationGroupId] = &cacheData{ci, ts}
		ids = append(ids, *r.ReplicationGroupId)
	}

	cl.mu.Lock()
	cl.names = ids
	cl.cacheList = cacheList
	cl.mu.Unlock()

	cl.l.Infof("elasticaches.expand: got %d caches", len(ids))
}

func newElastiCacheLister(c *configpb.ElastiCaches, region string, l *logger.Logger) (*elastiCacheLister, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("AWS configuration error : %v", err)
	}

	client := elasticache.NewFromConfig(cfg)

	cl := &elastiCacheLister{
		c:             c,
		clusterclient: client,
		rgclient:      client,
		tagclient:     client,
		cacheList:     make(map[string]*cacheData),
		l:             l,
		discoverTags:  true,
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
