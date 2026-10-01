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

// nativeBuckets holds Prometheus-style native histogram buckets. Buckets are
// sparse and their boundaries are powers of base = 2^(2^-schema). Bucket i
// covers (base^(i-1), base^i] in positive and [-base^i, -base^(i-1)) in
// negative. See https://prometheus.io/docs/specs/native_histograms/.
type nativeBuckets struct {
	schema        int32
	zeroThreshold float64
	zeroCount     int64 // samples in [-zeroThreshold, zeroThreshold]
	positive      map[int]int64
	negative      map[int]int64
}

func newNativeBuckets(schema int32, zeroThreshold float64) *nativeBuckets {
	return &nativeBuckets{
		schema:        schema,
		zeroThreshold: zeroThreshold,
		positive:      make(map[int]int64),
		negative:      make(map[int]int64),
	}
}

// NewNativeDistribution returns a new distribution container with
// Prometheus-style native histogram buckets. Bucket boundaries are powers of
// 2^(2^-schema), and samples in [-zeroThreshold, zeroThreshold] are counted in
// a separate zero bucket.
func NewNativeDistribution(schema int32, zeroThreshold float64) (*Distribution, error) {
	d := &Distribution{native: newNativeBuckets(schema, zeroThreshold)}
	if err := d.Verify(); err != nil {
		return nil, err
	}
	return d, nil
}

// nativeBucketKey returns the index of the bucket that v, a positive number,
// belongs to. It's adapted from prometheus/client_golang's
// histogramCounts.observe.
func nativeBucketKey(schema int32, v float64) int {
	isInf := math.IsInf(v, 1)
	if isInf {
		// Pretend v is MaxFloat64 but later increment key by one.
		v = math.MaxFloat64
	}

	var key int
	frac, exp := math.Frexp(v)
	if schema > 0 {
		bounds := nativeHistogramBounds[schema]
		key = sort.SearchFloat64s(bounds, frac) + (exp-1)*len(bounds)
	} else {
		key = exp
		if frac == 0.5 {
			key--
		}
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

func (nb *nativeBuckets) addSample(v float64) {
	switch {
	case v > nb.zeroThreshold:
		nb.positive[nativeBucketKey(nb.schema, v)]++
	case v < -nb.zeroThreshold:
		nb.negative[nativeBucketKey(nb.schema, -v)]++
	default:
		nb.zeroCount++
	}
}

// compatible returns true if distributions with nb and other buckets can be
// added to or subtracted from each other. Either of them can be nil.
func (nb *nativeBuckets) compatible(other *nativeBuckets) bool {
	if nb == nil || other == nil {
		return nb == other
	}
	return nb.schema == other.schema && nb.zeroThreshold == other.zeroThreshold
}

func (nb *nativeBuckets) addOrSubtract(delta *nativeBuckets, subtract bool) {
	sign := int64(1)
	if subtract {
		sign = -1
	}
	nb.zeroCount += sign * delta.zeroCount

	merge := func(dst, src map[int]int64) {
		for k, c := range src {
			dst[k] += sign * c
			// Keep buckets sparse.
			if dst[k] == 0 {
				delete(dst, k)
			}
		}
	}
	merge(nb.positive, delta.positive)
	merge(nb.negative, delta.negative)
}

func (nb *nativeBuckets) clone() *nativeBuckets {
	return &nativeBuckets{
		schema:        nb.schema,
		zeroThreshold: nb.zeroThreshold,
		zeroCount:     nb.zeroCount,
		positive:      maps.Clone(nb.positive),
		negative:      maps.Clone(nb.negative),
	}
}

func (nb *nativeBuckets) verify(count int64) error {
	if nb.schema < minNativeSchema || nb.schema > maxNativeSchema {
		return fmt.Errorf("invalid native buckets schema (%d), valid range: %d to %d", nb.schema, minNativeSchema, maxNativeSchema)
	}
	if !(nb.zeroThreshold >= 0) {
		return fmt.Errorf("invalid native buckets zero threshold (%v), it can't be negative", nb.zeroThreshold)
	}
	countSum := nb.zeroCount
	for _, buckets := range []map[int]int64{nb.positive, nb.negative} {
		for _, c := range buckets {
			countSum += c
		}
	}
	if count != countSum {
		return fmt.Errorf("sum of bucket counts (%d) don't match with the overall count (%d)", countSum, count)
	}
	return nil
}

// writeString writes native buckets in the following format:
// |schema:<schema>|zt:<zero threshold>|zc:<zero count>|pb:<key>=<count>,..|nb:<key>=<count>,..
// pb (positive buckets) and nb (negative buckets) are skipped if empty.
func (nb *nativeBuckets) writeString(b *strings.Builder) {
	b.WriteString("|schema:")
	b.WriteString(strconv.Itoa(int(nb.schema)))
	b.WriteString("|zt:")
	b.WriteString(strconv.FormatFloat(nb.zeroThreshold, 'f', -1, 64))
	b.WriteString("|zc:")
	b.WriteString(strconv.FormatInt(nb.zeroCount, 10))

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
	writeBuckets("|pb:", nb.positive)
	writeBuckets("|nb:", nb.negative)
}

// parseToken parses a native buckets token, written by writeString, into nb.
func (nb *nativeBuckets) parseToken(key, val string) (err error) {
	parseBuckets := func(buckets map[int]int64) error {
		if val == "" {
			return nil
		}
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
		nb.schema = int32(schema)
	case "zt":
		nb.zeroThreshold, err = strconv.ParseFloat(val, 64)
	case "zc":
		nb.zeroCount, err = strconv.ParseInt(val, 10, 64)
	case "pb":
		err = parseBuckets(nb.positive)
	case "nb":
		err = parseBuckets(nb.negative)
	}
	return err
}

// NativeDistributionData is the native buckets part of DistributionData.
type NativeDistributionData struct {
	Schema        int32
	ZeroThreshold float64
	ZeroCount     int64 // count of values in [-ZeroThreshold, ZeroThreshold]

	// Bucket counts by bucket index. With base = 2^(2^-Schema), positive bucket
	// i covers (base^(i-1), base^i] and negative bucket i covers
	// [-base^i, -base^(i-1)). Only the buckets with values are present.
	PositiveBuckets map[int]int64
	NegativeBuckets map[int]int64
}

// data returns distribution data for the native buckets. Along with the
// native buckets, it includes a view of the same data as regular buckets for
// the surfacers that don't handle native buckets: (-Inf, 0) for all the
// negative values, [0, ..) for the zero bucket, and then all the positive
// buckets from the smallest to the largest one, including the empty ones.
//
// Note that native buckets include their upper bound, while regular buckets
// include their lower bound. We ignore that in this view; it matters only for
// the values that are exactly at a bucket boundary.
func (nb *nativeBuckets) data(count int64, sum float64) *DistributionData {
	var negCount int64
	for _, c := range nb.negative {
		negCount += c
	}
	dd := &DistributionData{
		LowerBounds:  []float64{math.Inf(-1), 0},
		BucketCounts: []int64{negCount, nb.zeroCount},
		Count:        count,
		Sum:          sum,
		Native: &NativeDistributionData{
			Schema:          nb.schema,
			ZeroThreshold:   nb.zeroThreshold,
			ZeroCount:       nb.zeroCount,
			PositiveBuckets: maps.Clone(nb.positive),
			NegativeBuckets: maps.Clone(nb.negative),
		},
	}

	if len(nb.positive) == 0 {
		return dd
	}
	keys := slices.Sorted(maps.Keys(nb.positive))
	minKey, maxKey := keys[0], keys[len(keys)-1]
	for k := minKey; k <= maxKey; k++ {
		dd.LowerBounds = append(dd.LowerBounds, nativeUpperBound(nb.schema, k-1))
		dd.BucketCounts = append(dd.BucketCounts, nb.positive[k])
	}
	// Overflow bucket, always empty.
	dd.LowerBounds = append(dd.LowerBounds, nativeUpperBound(nb.schema, maxKey))
	dd.BucketCounts = append(dd.BucketCounts, 0)
	return dd
}
