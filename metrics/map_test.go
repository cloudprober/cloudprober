// Copyright 2017-2023 The Cloudprober Authors.
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
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func verify[T Number](t *testing.T, m *Map[T], expectedKeys []string, expectedMap map[string]T) {
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

func TestMap(t *testing.T) {
	m := NewMap("code").IncKeyBy("200", 4000)
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
	m1 := m.Clone().(*Map[int64])

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
	t.Helper()

	tests := []struct {
		name      string
		last      *Map[int64]
		current   *Map[int64]
		wantErr   bool
		wantReset bool
		wantMap   map[string]int64
	}{
		{
			name:    "no-reset",
			last:    NewMap("code").IncKeyBy("200", 4000).IncKeyBy("403", 2),
			current: NewMap("code").IncKeyBy("200", 4010).IncKeyBy("403", 3),
			wantMap: map[string]int64{
				"200": 10,
				"403": 1,
			},
		},
		{
			name:    "no-reset-new-keys",
			last:    NewMap("code").IncKeyBy("200", 4000).IncKeyBy("403", 2),
			current: NewMap("code").IncKeyBy("200", 4010).IncKeyBy("403", 2).IncKeyBy("502", 3),
			wantMap: map[string]int64{
				"200": 10,
				"403": 0,
				"502": 3,
			},
		},
		{
			name:    "reset-key-disappeared",
			last:    NewMap("code").IncKeyBy("200", 4000).IncKeyBy("403", 2),
			current: NewMap("code").IncKeyBy("200", 4010).IncKeyBy("500", 3),
			wantMap: map[string]int64{
				"200": 4010,
				"500": 3,
			},
			wantReset: true,
		},
		{
			name:    "reset-total-going-down",
			last:    NewMap("code").IncKeyBy("200", 4000).IncKeyBy("403", 2),
			current: NewMap("code").IncKeyBy("200", 3800).IncKeyBy("403", 3),
			wantMap: map[string]int64{
				"200": 3800,
				"403": 3,
			},
			wantReset: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotReset, err := tt.current.SubtractCounter(tt.last)
			if (err != nil) != tt.wantErr {
				t.Errorf("Map.SubtractCounter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.wantReset, gotReset, "reset")

			expectedKeys := make([]string, 0, len(tt.wantMap))
			wantTotal := int64(0)
			for k, v := range tt.wantMap {
				expectedKeys = append(expectedKeys, k)
				wantTotal += v
			}
			sort.Strings(expectedKeys)

			assert.Equal(t, wantTotal, tt.current.total, "total")
			verify(t, tt.current, expectedKeys, tt.wantMap)
		})
	}
}

func TestMapString(t *testing.T) {
	tests := []struct {
		name        string
		typ         string
		wantString  string
		parseString string
		wantErr     bool
	}{
		{
			name:       "int64",
			typ:        "int64",
			wantString: "map:code,200:4000,403:2",
		},
		{
			name:        "int64 float errror",
			typ:         "int64",
			wantString:  "map:code,200:4000,403:2",
			parseString: "map:code,200:400.2,403:2",
			wantErr:     true,
		},
		{
			name:        "int64 format errror",
			typ:         "int64",
			wantString:  "map:code,200:4000,403:2",
			parseString: "map:code,200:4000403:2",
			wantErr:     true,
		},
		{
			name:       "float64",
			typ:        "float64",
			wantString: "map:code,200:4000.000,403:2.000",
		},
		{
			name:        "float64 float parsing error",
			typ:         "float64",
			wantString:  "map:code,200:4000.000,403:2.000",
			parseString: "map:code,200:40.00.000,403:2.000",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.parseString == "" {
				tt.parseString = tt.wantString
			}

			if tt.typ == "int64" {
				m := NewMap("code").IncKeyBy("200", 4000).IncKeyBy("403", 2)
				s := m.String()
				assert.Equal(t, tt.wantString, s)

				// Parse test
				m2, err := ParseMapFromString[int64](tt.parseString)
				if err != nil {
					if !tt.wantErr {
						t.Errorf("Unexpected error: %v", err)
					}
					return
				}
				if tt.wantErr && err == nil {
					t.Errorf("Expected error, but got none")
				}
				assert.Equal(t, s, m2.String())
			} else {
				m := NewMapFloat("code").IncKeyBy("200", 4000).IncKeyBy("403", 2)

				s := m.String()
				assert.Equal(t, tt.wantString, s)
				m2, err := ParseMapFromString[float64](s)
				assert.NoError(t, err)
				assert.Equal(t, s, m2.String())
			}
		})
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
	newMap := func(numKeys int) *Map[int64] {
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
