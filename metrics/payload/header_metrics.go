// Copyright 2017-2024 The Cloudprober Authors.
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
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
)

func (p *Parser) processHeaderMetrics(resp interface{}) *metrics.EventMetrics {
	em := p.newEM(nil)

	httpResp, ok := resp.(*http.Response)
	if !ok {
		p.l.Warningf("processHeaderMetric: expected http.Response, got %T", resp)
		return nil
	}

	for _, hm := range p.opts.GetHeaderMetric() {
		metricName := hm.GetMetricName()
		if metricName == "" {
			metricName = strings.Replace(strings.ToLower(hm.GetHeaderName()), "-", "_", -1)
		}
		val := httpResp.Header.Get(hm.GetHeaderName())
		if val == "" {
			p.l.Warningf("processHeaderMetric: header %s not found in response", hm.GetHeaderName())
			continue
		}
		switch hm.GetType() {
		case configpb.HeaderMetric_STRING:
			em.AddMetric(metricName, metrics.NewString(val))
		case configpb.HeaderMetric_INT:
			i, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				p.l.Warningf("processHeaderMetric: error parsing header %s as int: %v", hm.GetHeaderName(), err)
			}
			em.AddMetric(metricName, metrics.NewInt(i))
		case configpb.HeaderMetric_FLOAT:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				p.l.Warningf("processHeaderMetric: error parsing header %s as float: %v", hm.GetHeaderName(), err)
			}
			em.AddMetric(metricName, metrics.NewFloat(f))
		case configpb.HeaderMetric_HTTP_TIME:
			t, err := http.ParseTime(val)
			if err != nil {
				p.l.Warningf("processHeaderMetric: error parsing header %s as date: %v", hm.GetHeaderName(), err)
			}
			em.AddMetric(metricName, metrics.NewInt(t.Unix()))
		default:
			em.AddMetric(metricName, metrics.NewString(val))
		}
	}

	if len(em.MetricsKeys()) == 0 {
		return nil
	}
	return em
}
