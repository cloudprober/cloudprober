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
	"sort"
	"strconv"
	"strings"
	"sync"
)

// MapFloat implements a key-value store where keys are of type string and values
// are of type NumValue.
// It satisfies the Value interface.
type MapFloat struct {
	MapName string // Map key name
	mu      sync.RWMutex
	m       map[string]float64
	keys    []string

	// total is only used to figure out if counter is moving up or down (reset).
	total float64
}

// NewMapFloat returns a new MapFloat
func NewMapFloat(mapName string) *MapFloat {
	return &MapFloat{
		MapName: mapName,
		m:       make(map[string]float64),
	}
}

// GetKey returns the given key's value.
func (m *MapFloat) GetKey(key string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.m[key]
}

// Clone creates a clone of the Map. Clone makes sure that underlying data
// storage is properly cloned.
func (m *MapFloat) Clone() Value {
	m.mu.RLock()
	defer m.mu.RUnlock()
	newMap := &MapFloat{
		MapName: m.MapName,
		m:       make(map[string]float64),
		total:   m.total,
	}
	newMap.keys = make([]string, len(m.keys))
	for i, k := range m.keys {
		newMap.m[k] = m.m[k]
		newMap.keys[i] = m.keys[i]
	}
	return newMap
}

// Keys returns the list of keys
func (m *MapFloat) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.keys...)
}

// newKey adds a new key to the map, with its value set to defaultKeyValue
// This is an unsafe function, callers should take care of protecting the map
// from race conditions.
func (m *MapFloat) newKey(key string) {
	m.keys = append(m.keys, key)
	sort.Strings(m.keys)
	m.m[key] = float64(0)
}

// IncKeyBy increments the given key's value by NumValue.
func (m *MapFloat) IncKeyBy(key string, delta float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.m[key]; !ok {
		m.newKey(key)
	}
	m.m[key] += delta
	m.total += delta
}

// IncKey increments the given key's value by one.
func (m *MapFloat) IncKey(key string) {
	m.IncKeyBy(key, float64(1))
}

// Add adds a value (type Value) to the receiver Map. A non-Map value returns
// an error. This is part of the Value interface.
func (m *MapFloat) Add(val Value) error {
	_, err := m.addOrSubtract(val, false)
	return err
}

// SubtractCounter subtracts the provided "lastVal", assuming that value
// represents a counter, i.e. if "value" is less than "lastVal", we assume that
// counter has been reset and don't subtract.
func (m *MapFloat) SubtractCounter(lastVal Value) (bool, error) {
	return m.addOrSubtract(lastVal, true)
}

func (m *MapFloat) addOrSubtract(val Value, subtract bool) (bool, error) {
	delta, ok := val.(*MapFloat)
	if !ok {
		return false, errors.New("incompatible value to add or subtract")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	delta.mu.RLock()
	defer delta.mu.RUnlock()

	if subtract && (m.total < delta.total) {
		return true, nil
	}

	var sortRequired bool
	for k, v := range delta.m {
		if subtract {
			// If a key is there in delta (lastVal) but not in the current val,
			// assume metric has been reset.
			if _, ok := m.m[k]; !ok {
				return true, nil
			}
			m.m[k] -= v
		} else {
			if _, ok := m.m[k]; !ok {
				sortRequired = true
				m.keys = append(m.keys, k)
				m.m[k] = v
				continue
			}
			m.m[k] += v
		}
	}
	if sortRequired {
		sort.Strings(m.keys)
	}
	return false, nil
}

// AddInt64 generates a panic for the Map type. This is added only to satisfy
// the Value interface.
func (m *MapFloat) AddInt64(i int64) {
	panic("Map type doesn't implement AddInt64()")
}

// AddFloat64 generates a panic for the Map type. This is added only to
// satisfy the Value interface.
func (m *MapFloat) AddFloat64(f float64) {
	panic("Map type doesn't implement AddFloat64()")
}

// String returns the string representation of the receiver Map.
// This is part of the Value interface.
// map:key,k1:v1,k2:v2
func (m *MapFloat) String() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var b strings.Builder
	b.Grow(64)

	b.WriteString("map:")
	b.WriteString(m.MapName)

	for _, k := range m.keys {
		b.WriteByte(',')
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(FloatToString(m.m[k]))
	}
	return b.String()
}

// ParseMapFromString parses a map value string into a map object.
// Note that the values are always parsed as floats, so even a map with integer
// values will become a float map.
// For example:
// "map:code,200:10123,404:21" will be parsed as:
// "map:code 200:10123.000 404:21.000".
func ParseMapFloatFromString(mapValue string) (*MapFloat, error) {
	tokens := strings.Split(mapValue, ",")
	if len(tokens) < 1 {
		return nil, errors.New("bad map value")
	}

	kv := strings.Split(tokens[0], ":")
	if kv[0] != "map" {
		return nil, errors.New("map value doesn't start with map:<key>")
	}

	m := NewMapFloat(kv[1])

	for _, tok := range tokens[1:] {
		kv := strings.Split(tok, ":")
		if len(kv) != 2 {
			return nil, errors.New("bad map value token: " + tok)
		}
		f, err := strconv.ParseFloat(kv[1], 64)
		if err != nil {
			return nil, fmt.Errorf("could not convert map key value %s to a float: %v", kv[1], err)
		}
		m.IncKeyBy(kv[0], f)
	}

	return m, nil
}
