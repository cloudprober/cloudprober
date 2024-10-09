// Copyright 2017 The Cloudprober Authors.
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

package metrics

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newEventMetrics(sent, rcvd, rtt int64, respCodes map[string]int64) *EventMetrics {
	respCodesVal := NewMap("code")
	for k, v := range respCodes {
		respCodesVal.IncKeyBy(k, v)
	}
	em := NewEventMetrics(time.Now()).
		AddMetric("sent", NewInt(sent)).
		AddMetric("rcvd", NewInt(rcvd)).
		AddMetric("rtt", NewInt(rtt)).
		AddMetric("resp-code", respCodesVal)
	return em
}

func verifyOrder(em *EventMetrics, names ...string) error {
	keys := em.MetricsKeys()
	for i := range names {
		if keys[i] != names[i] {
			return fmt.Errorf("Metrics not in order. At Index: %d, Expected: %s, Got: %s", i, names[i], keys[i])
		}
	}
	return nil
}

func verifyEventMetrics(t *testing.T, m *EventMetrics, sent, rcvd, rtt int64, respCodes map[string]int64) {
	// Verify that metrics are ordered correctly.
	if err := verifyOrder(m, "sent", "rcvd", "rtt", "resp-code"); err != nil {
		t.Error(err)
	}

	expectedMetrics := map[string]int64{
		"sent": sent,
		"rcvd": rcvd,
		"rtt":  rtt,
	}
	for k, eVal := range expectedMetrics {
		if m.Metric(k).(NumValue).Int64() != eVal {
			t.Errorf("Unexpected metric value. Expected: %d, Got: %d", eVal, m.Metric(k).(*Int).Int64())
		}
	}
	for k, eVal := range respCodes {
		if m.Metric("resp-code").(*Map[int64]).GetKey(k) != eVal {
			t.Errorf("Unexpected metric value. Expected: %d, Got: %d", eVal, m.Metric("resp-code").(*Map[int64]).GetKey(k))
		}
	}
}

func TestEventMetricsSubtractCounters(t *testing.T) {
	m := newEventMetrics(10, 10, 1000, make(map[string]int64))
	m.AddLabel("ptype", "http")

	// First run
	m2 := newEventMetrics(32, 22, 220100, map[string]int64{
		"200": 22,
	})
	gEM, err := m2.SubtractLast(m)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	verifyEventMetrics(t, gEM, 22, 12, 219100, map[string]int64{
		"200": 22,
	})

	// Second run
	m3 := newEventMetrics(42, 31, 300100, map[string]int64{
		"200": 24,
		"204": 8,
	})

	gEM, err = m3.SubtractLast(m2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	verifyEventMetrics(t, gEM, 10, 9, 80000, map[string]int64{
		"200": 2,
		"204": 8,
	})

	// Third run, expect reset
	m4 := newEventMetrics(10, 8, 1100, map[string]int64{
		"200": 8,
	})
	gEM, err = m4.SubtractLast(m3)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	verifyEventMetrics(t, gEM, 10, 8, 1100, map[string]int64{
		"200": 8,
	})
}

func TestKey(t *testing.T) {
	m := newEventMetrics(42, 31, 300100, map[string]int64{
		"200": 24,
		"204": 8,
	}).AddLabel("probe", "google-homepage")

	key := m.Key()
	wantKey := "sent,rcvd,rtt,resp-code,probe=google-homepage"

	if key != wantKey {
		t.Errorf("Got key: %s, wanted: %s", key, wantKey)
	}
}

func BenchmarkEventMetricsStringer(b *testing.B) {
	em := newEventMetrics(32, 22, 220100, map[string]int64{
		"200": 22,
		"404": 4500,
		"403": 4500,
	})
	// run the em.String() function b.N times
	for n := 0; n < b.N; n++ {
		_ = em.String()
	}
}

func TestAllocsPerRun(t *testing.T) {
	respCodesVal := NewMap("code")
	for k, v := range map[string]int64{
		"200": 22,
		"404": 4500,
		"403": 4500,
	} {
		respCodesVal.IncKeyBy(k, v)
	}

	var em *EventMetrics
	newAvg := testing.AllocsPerRun(100, func() {
		em = NewEventMetrics(time.Now()).
			AddMetric("sent", NewInt(32)).
			AddMetric("rcvd", NewInt(22)).
			AddMetric("rtt", NewInt(220100)).
			AddMetric("resp-code", respCodesVal)
	})

	stringAvg := testing.AllocsPerRun(100, func() {
		_ = em.String()
	})

	t.Logf("Average allocations per run: ForNew=%v, ForString=%v", newAvg, stringAvg)
}

func TestLatencyUnitToString(t *testing.T) {
	tests := map[time.Duration]string{
		0:                "us",
		time.Second:      "s",
		time.Millisecond: "ms",
		time.Microsecond: "us",
		time.Nanosecond:  "ns",
	}
	for latencyUnit, want := range tests {
		t.Run(want, func(t *testing.T) {
			assert.Equal(t, want, LatencyUnitToString(latencyUnit), "LatencyUnitToString()")
		})
	}
}

func TestEventMetricsString(t *testing.T) {
	em := newEventMetrics(20, 18, 1400, map[string]int64{"200": 18})
	em.AddLabel("dst", "cloudprober.org")

	tsString := strconv.FormatInt(em.Timestamp.Unix(), 10)

	tests := []struct {
		name string
		opts []StringerOption
		want string
	}{
		{
			name: "default",
			want: fmt.Sprintf("%s labels=dst=cloudprober.org sent=20 rcvd=18 rtt=1400 resp-code=map:code,200:18", tsString),
		},
		{
			name: "no timstamp",
			opts: []StringerOption{StringerNoTimestamp()},
			want: "labels=dst=cloudprober.org sent=20 rcvd=18 rtt=1400 resp-code=map:code,200:18",
		},
		{
			name: "ignore metric rcvd",
			opts: []StringerOption{StringerIgnoreMetric(func(m string) bool {
				return m == "rcvd"
			})},
			want: fmt.Sprintf("%s labels=dst=cloudprober.org sent=20 rtt=1400 resp-code=map:code,200:18", tsString),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, em.String(tt.opts...))
		})
	}
}
