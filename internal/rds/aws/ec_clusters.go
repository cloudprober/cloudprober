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

// ecClusterInfo represents cache cluster items that we fetch from the elasticache API.
type ecClusterInfo struct {
	ID         string
	IP         string
	Port       int32
	TLSEnabled bool
	Engine     string
	Tags       map[string]string
}

// ecClusterLocalCacheData represents objects that we store in local cache.
type ecClusterLocalCacheData struct {
	ci          *ecClusterInfo
	lastUpdated int64
}

/*
ElastiCacheClustersFilters defines filters supported by the ec_cluster_instances resource
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

var ElastiCacheClustersFilters = struct {
	RegexFilterKeys []string
	LabelsFilter    bool
}{
	[]string{"name", "engine"},
	true,
}

// elastiCacheClusterLister is a AWS ElastiCache cluster lister. It implements a cache,
// that's populated at a regular interval by making the AWS API calls.
// Listing actually only returns the current contents of that cache.
type elastiCacheClusterLister struct {
	c         *configpb.ElastiCacheClusters
	client    elasticache.DescribeCacheClustersAPIClient
	tagclient *elasticache.Client
	l         *logger.Logger
	mu        sync.RWMutex
	names     []string
	cacheList map[string]*ecClusterLocalCacheData
	// This is mainly for unit testing, should be taken out if/when there is a respective
	// interface in AWS SDK for go v2 to replace this.
	discoverTags bool
}

// listResources returns the list of resource records, where each record
// consists of an cluster name and the endpoint associated with it.
func (cl *elastiCacheClusterLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	var resources []*pb.Resource

	allFilters, err := filter.ParseFilters(req.GetFilter(), ElastiCacheClustersFilters.RegexFilterKeys, "")
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
			Id:          proto.String(ins.ID),
			Name:        proto.String(name),
			Ip:          proto.String(ins.IP),
			Port:        proto.Int32(ins.Port),
			Labels:      ins.Tags,
			LastUpdated: proto.Int64(cl.cacheList[name].lastUpdated),
			Info:        []byte("clustered"),
		})
	}

	cl.l.Infof("ec_clusters.listResources: returning %d instances", len(resources))
	return resources, nil
}

// expand runs equivalent API calls as "aws elasticache describe-cache-clusters",
// and is used to populate the cache. More details about this call is available in
// https://docs.aws.amazon.com/cli/latest/reference/elasticache/describe-cache-clusters.html
func (cl *elastiCacheClusterLister) expand(reEvalInterval time.Duration) {
	cl.l.Infof("ec_clusters.expand: expanding AWS targets")

	resp, err := cl.client.DescribeCacheClusters(context.TODO(), nil)
	if err != nil {
		cl.l.Errorf("ec_clusters.expand: error while listing cache clusters: %v", err)
		return
	}

	var ids = make([]string, 0)
	var cacheList = make(map[string]*ecClusterLocalCacheData)
	ts := time.Now().Unix()
	for _, c := range resp.CacheClusters {
		if len(c.CacheNodes) == 0 {
			continue
		}
		ci := &ecClusterInfo{
			ID:         *c.CacheClusterId,
			TLSEnabled: *c.TransitEncryptionEnabled,
			IP:         *c.CacheNodes[0].Endpoint.Address,
			Port:       *c.CacheNodes[0].Endpoint.Port,
			Engine:     *c.Engine,
			Tags:       make(map[string]string),
		}

		if cl.discoverTags {
			// AWS doesn't return Tag information in the response, we'll need to request it separately
			// NB: This might get throttled by AWS, if we make too many requests, see if we can batch or slow down
			// Add sleep if needed to the end of the loop
			tagsResp, err := cl.tagclient.ListTagsForResource(context.TODO(), &elasticache.ListTagsForResourceInput{
				ResourceName: c.ARN,
			})
			if err != nil {
				cl.l.Errorf("ec_clusters.expand: error getting tags for cluster %s: %v", *c.CacheClusterId, err)
				continue
			}

			// Convert to map
			for _, t := range tagsResp.TagList {
				ci.Tags[*t.Key] = *t.Value
			}
		}

		cacheList[*c.CacheClusterId] = &ecClusterLocalCacheData{ci, ts}
		ids = append(ids, *c.CacheClusterId)
	}

	cl.mu.Lock()
	cl.names = ids
	cl.cacheList = cacheList
	cl.mu.Unlock()

	cl.l.Infof("ec_clusters.expand: got %d caches", len(ids))
}

func newElastiCacheClusterLister(c *configpb.ElastiCacheClusters, region string, l *logger.Logger) (*elastiCacheClusterLister, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("AWS configuration error: %v", err)
	}

	client := elasticache.NewFromConfig(cfg)

	cl := &elastiCacheClusterLister{
		c:            c,
		client:       client,
		tagclient:    client,
		cacheList:    make(map[string]*ecClusterLocalCacheData),
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
