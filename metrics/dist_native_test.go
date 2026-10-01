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

package metrics

import (
	"fmt"
	"math"
	"testing"

	distpb "github.com/cloudprober/cloudprober/metrics/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/prototext"
)

func testNativeDist(t *testing.T, schema int32, samples ...float64) *Distribution {
	t.Helper()
	d, err := NewNativeDistribution(schema)
	if err != nil {
		t.Fatalf("NewNativeDistribution(%d): %v", schema, err)
	}
	for _, s := range samples {
		d.AddSample(s)
	}
	return d
}

func TestNewNativeDistributionFromProto(t *testing.T) {
	tests := []struct {
		inputProto string
		wantSchema int32
		wantErr    bool
	}{
		{
			inputProto: "native_buckets {}",
			wantSchema: 3,
		},
		{
			inputProto: "native_buckets { schema: 0 }",
			wantSchema: 0,
		},
		{
			inputProto: "native_buckets { schema: -4 }",
			wantSchema: -4,
		},
		{
			inputProto: "native_buckets { schema: 9 }",
			wantErr:    true,
		},
		{
			inputProto: "native_buckets { schema: -5 }",
			wantErr:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.inputProto, func(t *testing.T) {
			distProto := &distpb.Dist{}
			assert.NoError(t, prototext.Unmarshal([]byte(test.inputProto), distProto))

			d, err := NewDistributionFromProto(distProto)
			if test.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.wantSchema, d.native.Schema)
		})
	}
}

func TestNativeBucketKey(t *testing.T) {
	tests := []struct {
		schema  int32
		v       float64
		wantKey int
	}{
		// Buckets are (4^(i-1), 4^i]
		{schema: -1, v: 0.25, wantKey: -1},
		{schema: -1, v: 0.26, wantKey: 0},
		{schema: -1, v: 1, wantKey: 0},
		{schema: -1, v: 3.9, wantKey: 1},
		{schema: -1, v: 4, wantKey: 1},
		{schema: -1, v: 4.1, wantKey: 2},
		{schema: -1, v: math.MaxFloat64, wantKey: 512},
		{schema: -1, v: math.Inf(1), wantKey: 513},
		// Buckets are (2^(i-1), 2^i]
		{schema: 0, v: 0.5, wantKey: -1},
		{schema: 0, v: 0.75, wantKey: 0},
		{schema: 0, v: 1, wantKey: 0},
		{schema: 0, v: 1.5, wantKey: 1},
		{schema: 0, v: 2, wantKey: 1},
		{schema: 0, v: 2.1, wantKey: 2},
		{schema: 0, v: math.MaxFloat64, wantKey: 1024},
		{schema: 0, v: math.Inf(1), wantKey: 1025},
		// Bucket boundaries are 2^(i/4): ..., 0.84, 1, 1.19, 1.41, 1.68, 2, ..
		{schema: 2, v: 0.8408964152537144, wantKey: -1},
		{schema: 2, v: 0.85, wantKey: 0},
		{schema: 2, v: 1, wantKey: 0},
		{schema: 2, v: 1.189207115002721, wantKey: 1},
		{schema: 2, v: 1.19, wantKey: 2},
		{schema: 2, v: 2, wantKey: 4},
		{schema: 2, v: math.MaxFloat64, wantKey: 4096},
		{schema: 2, v: math.Inf(1), wantKey: 4097},
		// Bucket boundaries are 2^(i/8).
		{schema: 3, v: 1.5, wantKey: 5},
		{schema: 3, v: 3, wantKey: 13},
		{schema: 3, v: 100, wantKey: 54},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("schema=%d,v=%v", test.schema, test.v), func(t *testing.T) {
			assert.Equal(t, test.wantKey, nativeBucketKey(test.schema, test.v))
		})
	}
}

// Verify for all schemas that buckets include their upper bound, and that the
// next value goes to the next bucket.
func TestNativeBucketBoundaries(t *testing.T) {
	for schema := int32(minNativeSchema); schema <= maxNativeSchema; schema++ {
		for key := -40; key <= 40; key++ {
			ub := nativeUpperBound(schema, key)
			assert.Equal(t, key, nativeBucketKey(schema, ub), "schema=%d, upper bound=%v", schema, ub)
			next := math.Nextafter(ub, math.Inf(1))
			assert.Equal(t, key+1, nativeBucketKey(schema, next), "schema=%d, value just above upper bound=%v", schema, next)
		}
	}

	// Upper bound for a few known buckets.
	assert.Equal(t, 0.25, nativeUpperBound(-1, -1))
	assert.Equal(t, 4.0, nativeUpperBound(-1, 1))
	assert.Equal(t, 0.5, nativeUpperBound(0, -1))
	assert.Equal(t, 2.0, nativeUpperBound(0, 1))
	assert.Equal(t, 0.8408964152537144, nativeUpperBound(2, -1))
	assert.Equal(t, 1.189207115002721, nativeUpperBound(2, 1))
	assert.Equal(t, 2.0, nativeUpperBound(3, 8))
}

func TestNativeDistAddSample(t *testing.T) {
	d := testNativeDist(t, 0, 0, 0.75, 1, 1.5, 2, 3, 17, -3)

	assert.Equal(t, int64(8), d.count)
	assert.Equal(t, 22.25, d.sum)
	assert.Equal(t, int64(1), d.native.ZeroCount)
	assert.Equal(t, map[int]int64{0: 2, 1: 2, 2: 1, 5: 1}, d.native.Positive)
	assert.Equal(t, map[int]int64{2: 1}, d.native.Negative)
	assert.NoError(t, d.Verify())

	// Smallest and largest possible values are valid for all schemas.
	for schema := int32(minNativeSchema); schema <= maxNativeSchema; schema++ {
		d := testNativeDist(t, schema, math.SmallestNonzeroFloat64, math.MaxFloat64, math.Inf(1), math.Inf(-1))
		assert.NoError(t, d.Verify(), "schema=%d", schema)
		assert.Len(t, d.native.Positive, 3, "schema=%d", schema)
	}
}

func TestNativeDistAddAndSubtract(t *testing.T) {
	d := testNativeDist(t, 0, 0, 1, 1.5, -3)

	d2 := d.Clone().(*Distribution)
	assert.Equal(t, d.String(), d2.String())
	for _, s := range []float64{0, 1.5, 17, -3, -100} {
		d2.AddSample(s)
	}
	// Clone doesn't share buckets with the original.
	assert.Equal(t, "dist:sum:-0.5|count:4|schema:0|zc:1|pb:0=1,1=1|nb:2=1", d.String())
	total := d2.String()
	assert.Equal(t, "dist:sum:-85|count:9|schema:0|zc:2|pb:0=1,1=2,5=1|nb:2=2,7=1", total)

	// Subtract: buckets that become empty are removed.
	wasReset, err := d2.SubtractCounter(d)
	assert.NoError(t, err)
	assert.False(t, wasReset)
	assert.Equal(t, "dist:sum:-84.5|count:5|schema:0|zc:1|pb:1=1,5=1|nb:2=1,7=1", d2.String())
	assert.NoError(t, d2.Verify())

	// Add it back
	assert.NoError(t, d2.Add(d))
	assert.Equal(t, total, d2.String())
	assert.NoError(t, d2.Verify())

	// Reset: last value has higher count.
	last := d2.Clone()
	d3 := testNativeDist(t, 0, 4)
	wasReset, err = d3.SubtractCounter(last)
	assert.NoError(t, err)
	assert.True(t, wasReset)
	assert.Equal(t, "dist:sum:4|count:1|schema:0|zc:0|pb:2=1", d3.String())
}

func TestNativeDistAddIncompatible(t *testing.T) {
	d := testNativeDist(t, 3, 1)

	for name, other := range map[string]Value{
		"regular_buckets":    NewDistribution([]float64{1, 2}),
		"different_schema":   testNativeDist(t, 2),
		"not_a_distribution": NewInt(1),
	} {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, d.Add(other))
			_, err := d.SubtractCounter(other)
			assert.Error(t, err)
			if od, ok := other.(*Distribution); ok {
				assert.Error(t, od.Add(d))
			}
		})
	}
	assert.Equal(t, "dist:sum:1|count:1|schema:3|zc:0|pb:0=1", d.String())
}

func TestNativeDistStringAndParse(t *testing.T) {
	tests := []struct {
		name string
		d    *Distribution
		want string
	}{
		{
			name: "empty",
			d:    testNativeDist(t, 3),
			want: "dist:sum:0|count:0|schema:3|zc:0",
		},
		{
			name: "positive_only",
			d:    testNativeDist(t, 3, 1.5, 3, 3, 100),
			want: "dist:sum:107.5|count:4|schema:3|zc:0|pb:5=1,13=2,54=1",
		},
		{
			name: "negative_schema_and_keys",
			d:    testNativeDist(t, -1, 0, 0.0625, 0.25, 3, -0.25),
			want: "dist:sum:3.0625|count:5|schema:-1|zc:1|pb:-2=1,-1=1,1=1|nb:-1=1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, test.d.String())

			parsed, err := ParseDistFromString(test.want)
			assert.NoError(t, err)
			assert.Equal(t, test.d.native, parsed.native)
			assert.Equal(t, test.want, parsed.String())

			// Verify that parsed distribution is usable.
			assert.NoError(t, parsed.Add(test.d))

			// Through the generic value parser.
			v, err := ParseValueFromString(test.want)
			assert.NoError(t, err)
			assert.Equal(t, test.want, v.String())
		})
	}

	// Tokens can come in any order.
	d, err := ParseDistFromString("dist:pb:1=2|zc:1|schema:0|count:3|sum:3")
	assert.NoError(t, err)
	assert.Equal(t, "dist:sum:3|count:3|schema:0|zc:1|pb:1=2", d.String())

	for name, s := range map[string]string{
		"no_schema":            "dist:sum:3|count:2|zc:0|pb:1=2",
		"invalid_schema":       "dist:sum:3|count:2|schema:9|pb:1=2",
		"non_integer_schema":   "dist:sum:3|count:2|schema:a|pb:1=2",
		"count_mismatch":       "dist:sum:3|count:3|schema:0|zc:0|pb:1=2",
		"bucket_without_count": "dist:sum:3|count:2|schema:0|pb:1",
		"non_integer_key":      "dist:sum:3|count:2|schema:0|pb:1.5=2",
		"non_integer_count":    "dist:sum:3|count:2|schema:0|pb:1=a",
		"invalid_zero_count":   "dist:sum:3|count:2|schema:0|zc:a|pb:1=2",
		"empty_buckets":        "dist:sum:3|count:0|schema:0|pb:",
		"key_too_large":        "dist:sum:3|count:2|schema:0|pb:1026=2",
		"key_too_small":        "dist:sum:3|count:2|schema:0|nb:-1075=2",
		"keys_far_apart":       "dist:sum:3|count:2|schema:0|pb:-20000000=1,20000000=1",
		"mixed_buckets":        "dist:sum:3|count:2|schema:0|pb:1=2|lb:-Inf,1|bc:0,2",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseDistFromString(s)
			assert.Error(t, err)
		})
	}
}

func TestNativeDistData(t *testing.T) {
	tests := []struct {
		name string
		d    *Distribution
		want *DistributionData
	}{
		{
			name: "empty",
			d:    testNativeDist(t, 3),
			want: &DistributionData{
				LowerBounds:  []float64{math.Inf(-1), 0},
				BucketCounts: []int64{0, 0},
				Native: &NativeBuckets{
					Schema:   3,
					Positive: map[int]int64{},
					Negative: map[int]int64{},
				},
			},
		},
		{
			name: "no_positive_values",
			d:    testNativeDist(t, 0, 0, 0, -3),
			want: &DistributionData{
				LowerBounds:  []float64{math.Inf(-1), 0},
				BucketCounts: []int64{1, 2},
				Count:        3,
				Sum:          -3,
				Native: &NativeBuckets{
					Schema:    0,
					ZeroCount: 2,
					Positive:  map[int]int64{},
					Negative:  map[int]int64{2: 1},
				},
			},
		},
		{
			// Regular buckets have all the buckets between the smallest and the
			// largest positive bucket, and an empty overflow bucket.
			name: "positive_and_negative_values",
			d:    testNativeDist(t, 0, 0, 0.75, 1, 1.5, 2, 17, -3, -100),
			want: &DistributionData{
				LowerBounds:  []float64{math.Inf(-1), 0, 0.5, 1, 2, 4, 8, 16, 32},
				BucketCounts: []int64{2, 1, 2, 2, 0, 0, 0, 1, 0},
				Count:        8,
				Sum:          -80.75,
				Native: &NativeBuckets{
					Schema:    0,
					ZeroCount: 1,
					Positive:  map[int]int64{0: 2, 1: 2, 5: 1},
					Negative:  map[int]int64{2: 1, 7: 1},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, test.d.Data())
		})
	}

	// Data doesn't share buckets with the distribution.
	d := testNativeDist(t, 0, 1)
	dd := d.Data()
	d.AddSample(1)
	assert.Equal(t, map[int]int64{0: 1}, dd.Native.Positive)

	// Regular distributions have no native data.
	assert.Nil(t, NewDistribution([]float64{1, 2}).Data().Native)
}
