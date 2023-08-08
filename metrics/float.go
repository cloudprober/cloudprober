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
	"strconv"
)

// Float implements NumValue with float64 storage. Note that Float is not concurrency
// safe.
type Float struct {
	f float64
	// If Str is defined, this is method used to convert Float into a string.
	Str func(float64) string
}

// NewFloat returns a new Float.
func NewFloat(f float64) *Float {
	return &Float{f: f}
}

// Clone returns a copy the receiver Float
func (f *Float) Clone() Value {
	return &Float{
		f:   f.f,
		Str: f.Str,
	}
}

// Int64 returns the stored float64 as int64.
func (f *Float) Int64() int64 {
	return int64(f.f)
}

// Float64 returns the stored float64.
func (f *Float) Float64() float64 {
	return f.f
}

// Inc increments the receiver Float by one.
// It's part of the NumValue interface.
func (f *Float) Inc() {
	f.f++
}

// Add adds a Value to the receiver Float. If Value is not Float, an error is returned.
// It's part of the Value interface.
func (f *Float) Add(val Value) error {
	delta, ok := val.(*Float)
	if !ok {
		return errors.New("incompatible value to add")
	}
	f.f += delta.f
	return nil
}

// SubtractCounter subtracts the provided "lastVal", assuming that value
// represents a counter, i.e. if "value" is less than "lastVal", we assume that
// counter has been reset and don't subtract.
func (f *Float) SubtractCounter(lastVal Value) (bool, error) {
	lv, ok := lastVal.(*Float)
	if !ok {
		return false, errors.New("incompatible value to add")
	}
	if f.f < lv.f {
		return true, nil
	}

	f.f -= lv.f
	return false, nil
}

// AddFloat64 adds a float64 to the receiver Float.
func (f *Float) AddFloat64(ff float64) {
	f.f += ff
}

func FloatToString(f float64) string {
	return strconv.FormatFloat(f, 'f', 3, 64)
}

// String returns the string representation of Float.
// It's part of the Value interface.
func (f *Float) String() string {
	if f.Str != nil {
		return f.Str(f.f)
	}
	return FloatToString(f.f)
}
