// Copyright 2017-2019 The Cloudprober Authors.
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

import "errors"

// String implements a value type with string storage.
// It satisfies the Value interface.
type String struct {
	s string
}

// NewString returns a new String with the given string value.
func NewString(s string) String {
	return String{s: s}
}

// Add isn't supported for the String type, this is only to satisfy the Value
// interface.
func (s String) Add(val Value) error {
	return errors.New("string value type doesn't support Add() operation")
}

// SubtractCounter isn't supported for the String type, this is only to satisfy
// the Value interface.
func (s String) SubtractCounter(val Value) (bool, error) {
	return false, errors.New("string value type doesn't support SubtractCounter() operation")
}

// String simply returns the stored string.
func (s String) String() string {
	return "\"" + s.s + "\""
}

// Clone returns the copy of receiver String.
func (s String) Clone() Value {
	return String{s: s.s}
}

// IsString checks if the given value is a string.
func IsString(v Value) bool {
	if v == nil {
		return false
	}
	_, ok := v.(String)
	return ok
}
