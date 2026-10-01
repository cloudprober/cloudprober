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
	"maps"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
)

const (
	minNativeSchema     = -4
	maxNativeSchema     = 8
	defaultNativeSchema = 3
)

// NativeBuckets holds Prometheus-style native histogram buckets. Buckets are
// sparse and their boundaries are powers of base = 2^(2^-Schema). Bucket i
// covers (base^(i-1), base^i] in Positive and [-base^i, -base^(i-1)) in
// Negative. Only the buckets with values are present. See
// https://prometheus.io/docs/specs/native_histograms/.
type NativeBuckets struct {
	Schema    int32
	ZeroCount int64 // count of zeros
	Positive  map[int]int64
	Negative  map[int]int64
}

func newNativeBuckets(schema int32) *NativeBuckets {
	return &NativeBuckets{
		Schema:   schema,
		Positive: make(map[int]int64),
		Negative: make(map[int]int64),
	}
}

// NewNativeDistribution returns a new distribution container with
// Prometheus-style native histogram buckets. Bucket boundaries are powers of
// 2^(2^-schema).
func NewNativeDistribution(schema int32) (*Distribution, error) {
	d := &Distribution{native: newNativeBuckets(schema)}
	if err := d.Verify(); err != nil {
		return nil, err
	}
	return d, nil
}

// nativeBucketKey returns the index of the bucket that v, a positive number,
// belongs to. It's adapted from prometheus/client_golang's
// histogramCounts.observe.
//
// Frexp splits v into frac and exp such that v = frac * 2^exp, with frac in
// [0.5, 1). For example, 3 = 0.75 * 2^2 and 100 = 0.78125 * 2^7. So exp tells
// us which power of 2 range v is in, [2^(exp-1), 2^exp), and frac tells us
// where v is within that range.
func nativeBucketKey(schema int32, v float64) int {
	isInf := math.IsInf(v, 1)
	if isInf {
		// Pretend v is MaxFloat64 but later increment key by one.
		v = math.MaxFloat64
	}

	var key int
	frac, exp := math.Frexp(v)
	if schema > 0 {
		// Each power of 2 range is split into n = len(bounds) buckets, the same
		// way in every range, so one table of bucket upper bounds for the range
		// [0.5, 1) works for all of them. (exp-1)*n is the number of buckets
		// before this range, and Search gives us the bucket's position within
		// the range (0 to n). For example, for schema 3 (n=8) and v=3, exp is 2
		// and frac=0.75 is at position 5, so the key is 1*8 + 5 = 13.
		bounds := nativeHistogramBounds[schema]
		key = (exp-1)*len(bounds) + sort.SearchFloat64s(bounds, frac)
	} else {
		// For schema 0, each power of 2 range is one bucket, (2^(exp-1), 2^exp],
		// and exp is its key. Exact powers of 2 (frac is 0.5) belong to the
		// previous bucket, as buckets include their upper bound.
		key = exp
		if frac == 0.5 {
			key--
		}
		// For negative schemas, each bucket covers 2^-schema of the schema 0
		// buckets. This is ceil(key / 2^-schema).
		offset := (1 << -schema) - 1
		key = (key + offset) >> -schema
	}

	if isInf {
		key++
	}
	return key
}

// nativeUpperBound returns the upper bound of the bucket with the given key.
func nativeUpperBound(schema int32, key int) float64 {
	if schema < 0 {
		return math.Ldexp(1, key<<-schema)
	}
	frac := nativeHistogramBounds[schema][key&((1<<schema)-1)]
	return math.Ldexp(frac, (key>>schema)+1)
}

func (nb *NativeBuckets) addSample(v float64) {
	switch {
	case v > 0:
		nb.Positive[nativeBucketKey(nb.Schema, v)]++
	case v < 0:
		nb.Negative[nativeBucketKey(nb.Schema, -v)]++
	default:
		nb.ZeroCount++
	}
}

// compatible returns true if distributions with nb and other buckets can be
// added to or subtracted from each other. Either of them can be nil.
func (nb *NativeBuckets) compatible(other *NativeBuckets) bool {
	if nb == nil || other == nil {
		return nb == other
	}
	return nb.Schema == other.Schema
}

func (nb *NativeBuckets) addOrSubtract(delta *NativeBuckets, subtract bool) {
	sign := int64(1)
	if subtract {
		sign = -1
	}
	nb.ZeroCount += sign * delta.ZeroCount

	merge := func(dst, src map[int]int64) {
		for k, c := range src {
			dst[k] += sign * c
			// Keep buckets sparse.
			if dst[k] == 0 {
				delete(dst, k)
			}
		}
	}
	merge(nb.Positive, delta.Positive)
	merge(nb.Negative, delta.Negative)
}

func (nb *NativeBuckets) clone() *NativeBuckets {
	return &NativeBuckets{
		Schema:    nb.Schema,
		ZeroCount: nb.ZeroCount,
		Positive:  maps.Clone(nb.Positive),
		Negative:  maps.Clone(nb.Negative),
	}
}

func (nb *NativeBuckets) verify(count int64) error {
	if nb.Schema < minNativeSchema || nb.Schema > maxNativeSchema {
		return fmt.Errorf("invalid native buckets schema (%d), valid range: %d to %d", nb.Schema, minNativeSchema, maxNativeSchema)
	}

	// Bucket keys can come from outside (ParseDistFromString). Make sure they
	// are in the range that float64 values can map to, as the regular buckets
	// view in data() gets as big as the range of the keys.
	minKey := nativeBucketKey(nb.Schema, math.SmallestNonzeroFloat64)
	maxKey := nativeBucketKey(nb.Schema, math.Inf(1))

	countSum := nb.ZeroCount
	for _, buckets := range []map[int]int64{nb.Positive, nb.Negative} {
		for k, c := range buckets {
			if k < minKey || k > maxKey {
				return fmt.Errorf("invalid native bucket key (%d) for schema %d, valid range: %d to %d", k, nb.Schema, minKey, maxKey)
			}
			countSum += c
		}
	}
	if count != countSum {
		return fmt.Errorf("sum of bucket counts (%d) don't match with the overall count (%d)", countSum, count)
	}
	return nil
}

// writeString writes native buckets in the following format:
// |schema:<schema>|zc:<zero count>|pb:<key>=<count>,..|nb:<key>=<count>,..
// pb (positive buckets) and nb (negative buckets) are skipped if empty.
func (nb *NativeBuckets) writeString(b *strings.Builder) {
	b.WriteString("|schema:")
	b.WriteString(strconv.Itoa(int(nb.Schema)))
	b.WriteString("|zc:")
	b.WriteString(strconv.FormatInt(nb.ZeroCount, 10))

	writeBuckets := func(name string, buckets map[int]int64) {
		if len(buckets) == 0 {
			return
		}
		b.WriteString(name)
		for i, k := range slices.Sorted(maps.Keys(buckets)) {
			if i != 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.Itoa(k))
			b.WriteByte('=')
			b.WriteString(strconv.FormatInt(buckets[k], 10))
		}
	}
	writeBuckets("|pb:", nb.Positive)
	writeBuckets("|nb:", nb.Negative)
}

// parseToken parses a native buckets token, written by writeString, into nb.
func (nb *NativeBuckets) parseToken(key, val string) (err error) {
	parseBuckets := func(buckets map[int]int64) error {
		for _, kc := range strings.Split(val, ",") {
			k, c, _ := strings.Cut(kc, "=")
			bucketKey, err := strconv.Atoi(k)
			if err != nil {
				return err
			}
			if buckets[bucketKey], err = strconv.ParseInt(c, 10, 64); err != nil {
				return err
			}
		}
		return nil
	}

	switch key {
	case "schema":
		var schema int64
		schema, err = strconv.ParseInt(val, 10, 32)
		nb.Schema = int32(schema)
	case "zc":
		nb.ZeroCount, err = strconv.ParseInt(val, 10, 64)
	case "pb":
		err = parseBuckets(nb.Positive)
	case "nb":
		err = parseBuckets(nb.Negative)
	}
	return err
}

// data returns distribution data for the native buckets. Along with the
// native buckets, it includes a view of the same data as regular buckets for
// the surfacers that don't handle native buckets: (-Inf, 0) for all the
// negative values, [0, ..) for zeros, and then all the positive buckets from
// the smallest to the largest one, including the empty ones.
//
// Note that native buckets include their upper bound, while regular buckets
// include their lower bound. We ignore that in this view; it matters only for
// the values that are exactly at a bucket boundary.
func (nb *NativeBuckets) data(count int64, sum float64) *DistributionData {
	var negCount int64
	for _, c := range nb.Negative {
		negCount += c
	}
	dd := &DistributionData{
		LowerBounds:  []float64{math.Inf(-1), 0},
		BucketCounts: []int64{negCount, nb.ZeroCount},
		Count:        count,
		Sum:          sum,
		Native:       nb.clone(),
	}

	if len(nb.Positive) == 0 {
		return dd
	}
	keys := slices.Collect(maps.Keys(nb.Positive))
	maxKey := slices.Max(keys)
	for k := slices.Min(keys); k <= maxKey; k++ {
		dd.LowerBounds = append(dd.LowerBounds, nativeUpperBound(nb.Schema, k-1))
		dd.BucketCounts = append(dd.BucketCounts, nb.Positive[k])
	}
	// Overflow bucket, always empty.
	dd.LowerBounds = append(dd.LowerBounds, nativeUpperBound(nb.Schema, maxKey))
	dd.BucketCounts = append(dd.BucketCounts, 0)
	return dd
}
