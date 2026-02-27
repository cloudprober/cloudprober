// Copyright 2026 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
