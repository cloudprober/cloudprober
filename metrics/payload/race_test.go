package payload

import (
	"sync"
	"testing"

	configpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
	"google.golang.org/protobuf/proto"
)

func TestPayloadMetricsConcurrent(t *testing.T) {
	c := &configpb.OutputMetricsOptions{
		AggregateInCloudprober: proto.Bool(true),
	}
	p, err := NewParser(c, nil)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	numIterations := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(targetID int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				p.PayloadMetrics(&Input{Text: []byte("m1 1\nm2 2")}, "target-1")
			}
		}(i)
	}
	wg.Wait()
}
