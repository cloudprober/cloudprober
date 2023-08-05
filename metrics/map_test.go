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
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func verify(t *testing.T, m *Map, expectedKeys []string, expectedMap map[string]int64) {
	t.Helper()

	if !reflect.DeepEqual(m.Keys(), expectedKeys) {
		t.Errorf("Map doesn't have expected keys. Got: %q, Expected: %q", m.Keys(), expectedKeys)
	}
	for k, v := range expectedMap {
		if m.GetKey(k) != v {
			t.Errorf("Key values not as expected. Key: %s, Got: %d, Expected: %d", k, m.GetKey(k), v)
		}
	}
}

func TestMap(t *testing.T) {
	m := NewMap("code")
	m.IncKeyBy("200", 4000)

	verify(t, m, []string{"200"}, map[string]int64{"200": 4000})

	m.IncKey("500")
	verify(t, m, []string{"200", "500"}, map[string]int64{
		"200": 4000,
		"500": 1,
	})

	// Verify that keys are ordered
	m.IncKey("404")
	verify(t, m, []string{"200", "404", "500"}, map[string]int64{
		"200": 4000,
		"404": 1,
		"500": 1,
	})

	// Clone m for verification later
	m1 := m.Clone().(*Map)

	// Verify add works as expected
	m2 := NewMap("code")
	m2.IncKeyBy("403", 2)
	err := m.Add(m2)
	if err != nil {
		t.Errorf("Add two maps produced error. Err: %v", err)
	}
	verify(t, m, []string{"200", "403", "404", "500"}, map[string]int64{
		"200": 4000,
		"403": 2,
		"404": 1,
		"500": 1,
	})

	// Verify that clones value has not changed
	verify(t, m1, []string{"200", "404", "500"}, map[string]int64{
		"200": 4000,
		"404": 1,
		"500": 1,
	})
}

func TestMapSubtractCounter(t *testing.T) {
	m1 := NewMap("code")
	m1.IncKeyBy("200", 4000)
	m1.IncKeyBy("403", 2)

	m2 := m1.Clone().(*Map)
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
	verify(t, m2, []string{"200", "403", "500"}, map[string]int64{
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
	verify(t, m3.(*Map), []string{"200", "403"}, map[string]int64{
		"200": 4000,
		"403": 2,
	})

}

func TestMapString(t *testing.T) {
	m := NewMap("code")
	m.IncKeyBy("403", 20)
	m.IncKeyBy("200", 4000)

	s := m.String()
	expectedString := "map:code,200:4000,403:20"
	if s != expectedString {
		t.Errorf("m.String()=%s, expected=%s", s, expectedString)
	}

	m2, err := ParseMapFromString(s)
	if err != nil {
		t.Errorf("ParseMapFromString(%s) returned error: %v", s, err)
	}

	s1 := m2.String()
	if s1 != s {
		t.Errorf("ParseMapFromString(%s).String() = %s, expected = %s", s, s1, s)
	}
}

func BenchmarkMapSetGet(b *testing.B) {
	// run the em.String() function b.N times
	for n := 0; n < b.N; n++ {
		m := NewMap("code")
		for i := 1; i <= 20; i++ {
			m.IncKeyBy(strconv.Itoa(100*i), int64(i))
		}
		// 3000 incKey operations per op
		for i := 0; i < 1000; i++ {
			for _, k := range m.Keys() {
				m.IncKey(k)
			}
		}

		for _, k := range m.Keys() {
			m.GetKey(k)
		}
	}
}

func BenchmarkMapClone(b *testing.B) {
	m := NewMap("code")
	for i := 1; i <= 20; i++ {
		m.IncKeyBy(strconv.Itoa(100*i), int64(i))
	}

	for n := 0; n < b.N; n++ {
		// clone 1000 times
		for i := 0; i < 1000; i++ {
			m.Clone()
		}
	}
}

func TestMapAllocsPerRun(t *testing.T) {
	newMap := func(numKeys int) *Map {
		m := NewMap("code")
		for i := 1; i <= numKeys; i++ {
			m.IncKeyBy(strconv.Itoa(100*i), int64(i))
		}
		return m
	}

	for _, n := range []int{5, 10, 20} {
		m := newMap(n)
		cloneAvg := testing.AllocsPerRun(100, func() {
			_ = m.Clone()
		})
		stringAvg := testing.AllocsPerRun(100, func() {
			_ = m.String()
		})

		assert.LessOrEqual(t, int(cloneAvg), 8)

		t.Logf("Average allocations per run (numKeys=%d): ForMapClone=%v, ForMapString=%v", n, cloneAvg, stringAvg)
	}
}
