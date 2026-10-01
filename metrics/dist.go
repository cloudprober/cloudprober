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
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	distpb "github.com/cloudprober/cloudprober/metrics/proto"
	"google.golang.org/api/googleapi"
	monitoring "google.golang.org/api/monitoring/v3"
)

// Distribution metrics type implements a histogram of values distributed over
// a set of pre-defined buckets, or over native buckets.
type Distribution struct {
	mu           sync.RWMutex
	lowerBounds  []float64 // bucket lower bounds
	bucketCounts []int64
	count        int64   // count of all values
	sum          float64 // sum of all samples.

	// Set only for the distributions with native buckets. Such distributions
	// don't use lowerBounds and bucketCounts.
	native *NativeBuckets
}

// NewDistribution returns a new distribution container.
func NewDistribution(lowerBounds []float64) *Distribution {
	return &Distribution{
		lowerBounds:  append([]float64{math.Inf(-1)}, lowerBounds...),
		bucketCounts: make([]int64, len(lowerBounds)+1),
	}
}

// NewExponentialDistribution returns a distribution container with
// exponentially growing bucket sizes. Buckets' lower bounds are determined as
// follows:
// -Inf,
// 0,
// scale_factor,
// scale_factor * base,
// scale_factor * base^2,
// ...
// scale_factor * base^(i-1).., ith bucket
// ...
// scale_factor * base^(numBuckets), last element (numBuckets+1-th)
func NewExponentialDistribution(base, scaleFactor float64, numBuckets int) (*Distribution, error) {
	if base < 1.01 {
		return nil, fmt.Errorf("exponential distribution's base (%f) should be at least 1.01", base)
	}
	lowerBounds := make([]float64, numBuckets+1)
	lowerBounds[0] = 0
	for i := 1; i < len(lowerBounds); i++ {
		lowerBounds[i] = scaleFactor * math.Pow(base, float64(i-1))
	}
	return NewDistribution(lowerBounds), nil
}

// NewDistributionFromProto returns a new distribution based on the provided
// protobuf.
func NewDistributionFromProto(distProto *distpb.Dist) (*Distribution, error) {

	switch distProto.Buckets.(type) {

	case *distpb.Dist_ExplicitBuckets:
		lbStringA := strings.Split(distProto.GetExplicitBuckets(), ",")
		lowerBounds := make([]float64, len(lbStringA))
		for i, tok := range lbStringA {
			lb, err := strconv.ParseFloat(tok, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid lower bound for bucket: %s. Err: %v", tok, err)
			}
			lowerBounds[i] = lb
		}
		return NewDistribution(lowerBounds), nil

	case *distpb.Dist_ExponentialBuckets:
		expb := distProto.GetExponentialBuckets()
		if expb.Base == 0 {
			expb.Base = 2
		}
		if expb.Base < 1.01 {
			return nil, fmt.Errorf("exponential distribution's base (%f) should be at least 1.01", expb.Base)
		}
		if expb.ScaleFactor == 0 {
			expb.ScaleFactor = 1
		}
		if expb.NumBuckets == 0 {
			expb.NumBuckets = 20
		}
		return NewExponentialDistribution(float64(expb.GetBase()), float64(expb.GetScaleFactor()), int(expb.GetNumBuckets()))

	case *distpb.Dist_NativeBuckets:
		nb := distProto.GetNativeBuckets()
		schema := int32(defaultNativeSchema)
		if nb.Schema != nil {
			schema = nb.GetSchema()
		}
		return NewNativeDistribution(schema)
	}

	return nil, fmt.Errorf("unknown buckets type: %v", distProto.Buckets)
}

func (d *Distribution) bucketIndex(sample float64) int {
	return sort.Search(len(d.lowerBounds), func(i int) bool { return sample < d.lowerBounds[i] }) - 1
}

// AddSample adds a sample to the receiver distribution.
func (d *Distribution) AddSample(sample float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.native != nil {
		d.native.addSample(sample)
	} else {
		d.bucketCounts[d.bucketIndex(sample)]++
	}
	d.sum += sample
	d.count++
}

// AddFloat64 adds an float64 to the receiver distribution.
func (d *Distribution) AddFloat64(f float64) {
	d.AddSample(f)
}

// Add adds a distribution to the receiver distribution. If both distributions
// don't have the same buckets (or the same schema in case of native buckets),
// an error is returned.
func (d *Distribution) Add(val Value) error {
	_, err := d.addOrSubtract(val, false)
	return err
}

// SubtractCounter subtracts the provided "lastVal", assuming that value
// represents a counter, i.e. if "value" is less than "lastVal", we assume that
// counter has been reset and don't subtract.
func (d *Distribution) SubtractCounter(lastVal Value) (bool, error) {
	return d.addOrSubtract(lastVal, true)
}

func (d *Distribution) addOrSubtract(val Value, subtract bool) (bool, error) {
	delta, ok := val.(*Distribution)
	if !ok {
		return false, errors.New("dist: incompatible value to add or subtract")
	}

	if !d.native.compatible(delta.native) {
		return false, errors.New("incompatible delta value, receiver and delta distributions should both have either regular buckets, or native buckets with the same schema")
	}
	if !reflect.DeepEqual(d.lowerBounds, delta.lowerBounds) {
		return false, fmt.Errorf("incompatible delta value, Bucket lower bounds in receiver distribution: %v, and in delta distribution: %v", d.lowerBounds, delta.lowerBounds)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	delta.mu.RLock()
	defer delta.mu.RUnlock()

	if subtract {
		// If receiver count is less than lastVal' count, assume reset and return.
		if d.count < delta.count {
			return true, nil
		}
		d.count -= delta.count
		d.sum -= delta.sum
	} else {
		d.count += delta.count
		d.sum += delta.sum
	}

	for i := 0; i < len(d.bucketCounts); i++ {
		if subtract {
			d.bucketCounts[i] -= delta.bucketCounts[i]
		} else {
			d.bucketCounts[i] += delta.bucketCounts[i]
		}
	}
	if d.native != nil {
		d.native.addOrSubtract(delta.native, subtract)
	}

	return false, nil
}

// String returns a string representation of the distribution:
// "dist:sum:<sum>|count:<count>|lb:<lower bounds>|bc:<bucket counts>"
// For example for a distribution with lower bounds 0.5, 2.0, 7.5 and
// bucket counts 34, 54, 121, 12, string representation will look like the
// following:
// dist:sum:899|count:221|lb:-Inf,0.5,2,7.5|bc:34,54,121,12
//
// Distributions with native buckets have schema, zero count, and positive and
// negative buckets (as <bucket index>=<count>) instead of lower bounds and
// bucket counts:
// dist:sum:899|count:221|schema:3|zc:2|pb:-2=5,3=214
func (d *Distribution) String() string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var b strings.Builder

	b.WriteString("dist:sum:")
	b.WriteString(strconv.FormatFloat(d.sum, 'f', -1, 64))
	b.WriteString("|count:")
	b.WriteString(strconv.FormatInt(d.count, 10))

	if d.native != nil {
		d.native.writeString(&b)
		return b.String()
	}

	b.WriteString("|lb:")
	for i, lb := range d.lowerBounds {
		if i != 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(lb, 'f', -1, 64))
	}

	b.WriteString("|bc:")
	for i, c := range d.bucketCounts {
		if i != 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatInt(c, 10))
	}

	return b.String()
}

// Verify verifies that the distribution is valid.
func (d *Distribution) Verify() error {
	if d.native != nil {
		if len(d.lowerBounds) != 0 || len(d.bucketCounts) != 0 {
			return errors.New("distribution can't have both regular and native buckets")
		}
		return d.native.verify(d.count)
	}
	if len(d.lowerBounds) == 0 {
		return errors.New("no distribution buckets found")
	}
	if len(d.lowerBounds) != len(d.bucketCounts) {
		return fmt.Errorf("size mismatch between buckets array (%v) and bucket counts array (%v)", d.lowerBounds, d.bucketCounts)
	}
	var countSum int64
	for _, c := range d.bucketCounts {
		countSum += c
	}
	if d.count != countSum {
		return fmt.Errorf("sum of bucket counts (%d) don't match with the overall count (%d)", countSum, d.count)
	}
	return nil
}

// ParseDistFromString parses a distribution value from a string that's in a
// format that's generated by the String() method:
// Example strings:
// dist:sum:899|count:221|lb:-Inf,0.5,2,7.5|bc:34,54,121,12
// dist:sum:899|count:221|schema:3|zc:2|pb:-2=5,3=214
func ParseDistFromString(str string) (*Distribution, error) {
	tokens := strings.SplitN(str, ":", 2)
	if len(tokens) != 2 || tokens[0] != "dist" {
		return nil, fmt.Errorf("invalid distribution string: %s", str)
	}

	d := &Distribution{}

	var f float64
	var i int64
	var err error

	errF := func(kv []string, err error) (*Distribution, error) {
		return nil, fmt.Errorf("invalid token (%s:%s) in the distribution string: %s. Err: %v", kv[0], kv[1], str, err)
	}

	for _, tok := range strings.Split(tokens[1], "|") {
		kv := strings.Split(tok, ":")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid distribution string: %s", str)
		}
		switch kv[0] {
		case "sum":
			if f, err = strconv.ParseFloat(kv[1], 64); err != nil {
				return errF(kv, err)
			}
			d.sum = f
		case "count":
			if i, err = strconv.ParseInt(kv[1], 10, 64); err != nil {
				return errF(kv, err)
			}
			d.count = i
		case "lb":
			for _, vs := range strings.Split(kv[1], ",") {
				if f, err = strconv.ParseFloat(vs, 64); err != nil {
					return errF(kv, err)
				}
				d.lowerBounds = append(d.lowerBounds, f)
			}
		case "bc":
			for _, vs := range strings.Split(kv[1], ",") {
				if i, err = strconv.ParseInt(vs, 10, 64); err != nil {
					return errF(kv, err)
				}
				d.bucketCounts = append(d.bucketCounts, i)
			}
		case "schema", "zc", "pb", "nb":
			if d.native == nil {
				// Start with an invalid schema so that Verify fails if the
				// string has no schema.
				d.native = newNativeBuckets(minNativeSchema - 1)
			}
			if err = d.native.parseToken(kv[0], kv[1]); err != nil {
				return errF(kv, err)
			}
		default:
			return errF(kv, nil)
		}
	}
	if err := d.Verify(); err != nil {
		return nil, err
	}
	return d, nil
}

// DistributionData stuct, along with Data() function, provides a way to
// readily share the Distribution data with other packages.
type DistributionData struct {
	LowerBounds  []float64 // bucket lower bounds
	BucketCounts []int64
	Count        int64   // count of all values
	Sum          float64 // sum of all samples.

	// Native is set only for the distributions with native buckets. For such
	// distributions, LowerBounds and BucketCounts provide a view of the native
	// buckets as regular buckets.
	Native *NativeBuckets
}

// Data returns a DistributionData object, built using Distribution's current
// state.
func (d *Distribution) Data() *DistributionData {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.native != nil {
		return d.native.data(d.count, d.sum)
	}
	return &DistributionData{
		LowerBounds:  d.lowerBounds,
		BucketCounts: d.bucketCounts,
		Count:        d.count,
		Sum:          d.sum,
	}
}

// StackdriverTypedValue returns a Stackdriver typed value corresponding to the
// receiver distribution. This routine is used by stackdriver surfacer. It
// returns nil for the distributions with native buckets as we don't support
// them for stackdriver.
func (d *Distribution) StackdriverTypedValue() *monitoring.TypedValue {
	if d.native != nil {
		return nil
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	distVal := &monitoring.Distribution{
		BucketCounts: googleapi.Int64s(append([]int64{}, d.bucketCounts...)),
		BucketOptions: &monitoring.BucketOptions{
			ExplicitBuckets: &monitoring.Explicit{
				Bounds: append([]float64{}, d.lowerBounds[1:]...),
			},
		},
		Count: d.count,
	}
	return &monitoring.TypedValue{
		DistributionValue: distVal,
	}
}

// Clone returns a copy of the receiver distribution.
func (d *Distribution) CloneDist() *Distribution {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.native != nil {
		return &Distribution{sum: d.sum, count: d.count, native: d.native.clone()}
	}
	newD := NewDistribution(d.lowerBounds[1:])
	newD.sum = d.sum
	newD.count = d.count
	copy(newD.bucketCounts, d.bucketCounts)
	return newD
}

// Clone returns a copy of the receiver distribution.
func (d *Distribution) Clone() Value {
	return d.CloneDist()
}
