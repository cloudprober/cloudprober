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
	"reflect"
	"testing"
)

func verifyMapFloat(t *testing.T, m *MapFloat, expectedKeys []string, expectedMap map[string]float64) {
	t.Helper()

	if !reflect.DeepEqual(m.Keys(), expectedKeys) {
		t.Errorf("Map doesn't have expected keys. Got: %q, Expected: %q", m.Keys(), expectedKeys)
	}
	for k, v := range expectedMap {
		if m.GetKey(k) != v {
			t.Errorf("Key values not as expected. Key: %s, Got: %v, Expected: %v", k, m.GetKey(k), v)
		}
	}
}

func TestMapFloat(t *testing.T) {
	m := NewMapFloat("code")
	m.IncKeyBy("200", 4000)

	verifyMapFloat(t, m, []string{"200"}, map[string]float64{"200": 4000})

	m.IncKey("500")
	verifyMapFloat(t, m, []string{"200", "500"}, map[string]float64{
		"200": 4000,
		"500": 1,
	})

	// Verify that keys are ordered
	m.IncKey("404")
	verifyMapFloat(t, m, []string{"200", "404", "500"}, map[string]float64{
		"200": 4000,
		"404": 1,
		"500": 1,
	})

	// Clone m for verification later
	m1 := m.Clone().(*MapFloat)

	// Verify add works as expected
	m2 := NewMapFloat("code")
	m2.IncKeyBy("403", 2)
	err := m.Add(m2)
	if err != nil {
		t.Errorf("Add two maps produced error. Err: %v", err)
	}
	verifyMapFloat(t, m, []string{"200", "403", "404", "500"}, map[string]float64{
		"200": 4000,
		"403": 2,
		"404": 1,
		"500": 1,
	})

	// Verify that clones value has not changed
	verifyMapFloat(t, m1, []string{"200", "404", "500"}, map[string]float64{
		"200": 4000,
		"404": 1,
		"500": 1,
	})
}

func TestMapFloatSubtractCounter(t *testing.T) {
	m1 := NewMapFloat("code")
	m1.IncKeyBy("200", 4000)
	m1.IncKeyBy("403", 2)

	m2 := m1.Clone().(*MapFloat)
	m2.IncKeyBy("200", 400)
	m2.IncKey("500")
	m2Clone := m2.Clone() // We'll use this for reset testing below.

	expectReset := false
	wasReset, err := m2.SubtractCounter(m1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if wasReset != expectReset {
		t.Errorf("wasReset=%v, expected=%v", wasReset, expectReset)
	}
	verifyMapFloat(t, m2, []string{"200", "403", "500"}, map[string]float64{
		"200": 400,
		"403": 0,
		"500": 1,
	})

	// Expect a reset this time, as m3 (m) will be smaller than m2Clone.
	m3 := m1.Clone()
	expectReset = true
	wasReset, err = m3.SubtractCounter(m2Clone)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if wasReset != expectReset {
		t.Errorf("wasReset=%v, expected=%v", wasReset, expectReset)
	}
	verifyMapFloat(t, m3.(*MapFloat), []string{"200", "403"}, map[string]float64{
		"200": 4000,
		"403": 2,
	})

}

func TestMapFloatString(t *testing.T) {
	m := NewMapFloat("code")
	m.IncKeyBy("403", 20)
	m.IncKeyBy("200", 4000)

	s := m.String()
	expectedString := "map:code,200:4000.000,403:20.000"
	if s != expectedString {
		t.Errorf("m.String()=%s, expected=%s", s, expectedString)
	}

	m2, err := ParseMapFloatFromString(s)
	if err != nil {
		t.Errorf("ParseMapFloatFromString(%s) returned error: %v", s, err)
	}

	s1 := m2.String()
	if s1 != s {
		t.Errorf("ParseMapFloatFromString(%s).String() = %s, expected = %s", s, s1, s)
	}
}
