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

type Number interface {
	int64 | float64
}

// Map implements a key-value store where keys are of type string and values
// are of type Number.
// It satisfies the Value interface.
type Map[T Number] struct {
	MapName string // Map key name
	mu      sync.RWMutex
	m       map[string]T
	keys    []string

	// total is only used to figure out if counter is moving up or down (reset).
	total T
}

func newMap[T Number](mapName string) *Map[T] {
	return &Map[T]{
		MapName: mapName,
		m:       make(map[string]T),
	}
}

// NewMap returns a new Map
func NewMap(mapName string) *Map[int64] {
	return newMap[int64](mapName)
}

func NewMapFloat(mapName string) *Map[float64] {
	return newMap[float64](mapName)
}

// GetKey returns the given key's value.
func (m *Map[T]) GetKey(key string) T {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.m[key]
}

// Clone creates a clone of the Map. Clone makes sure that underlying data
// storage is properly cloned.
func (m *Map[T]) Clone() Value {
	m.mu.RLock()
	defer m.mu.RUnlock()
	newMap := &Map[T]{
		MapName: m.MapName,
		m:       make(map[string]T, len(m.m)),
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
func (m *Map[T]) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string{}, m.keys...)
}

// newKey adds a new key to the map, with its value set to defaultKeyValue
// This is an unsafe function, callers should take care of protecting the map
// from race conditions.
func (m *Map[T]) newKey(key string) {
	m.keys = append(m.keys, key)
	sort.Strings(m.keys)
	m.m[key] = 0
}

// IncKeyBy increments the given key's value by Number.
func (m *Map[T]) IncKeyBy(key string, delta T) *Map[T] {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.m[key]; !ok {
		m.newKey(key)
	}
	m.m[key] += delta
	m.total += delta
	return m
}

// IncKey increments the given key's value by one.
func (m *Map[T]) IncKey(key string) *Map[T] {
	return m.IncKeyBy(key, 1)
}

// Add adds a value (type Value) to the receiver Map. A non-Map value returns
// an error. This is part of the Value interface.
func (m *Map[T]) Add(val Value) error {
	_, err := m.addOrSubtract(val, false)
	return err
}

// SubtractCounter subtracts the provided "lastVal", assuming that value
// represents a counter, i.e. if "value" is less than "lastVal", we assume that
// counter has been reset and don't subtract.
func (m *Map[T]) SubtractCounter(lastVal Value) (bool, error) {
	return m.addOrSubtract(lastVal, true)
}

func (m *Map[T]) addOrSubtract(val Value, subtract bool) (bool, error) {
	delta, ok := val.(*Map[T])
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
	// We use this map to restore the modified keys in case of a reset.
	subtractedKeys := make(map[string]T)

	for k, v := range delta.m {
		if subtract {
			// If a key is there in delta (lastVal) but not in the current val,
			// assume metric has been reset.
			if _, ok := m.m[k]; !ok {
				// Fix the keys modified so far and return.
				for k, v := range subtractedKeys {
					m.m[k] += v
					m.total += v
				}
				return true, nil
			}
			m.m[k] -= v
			m.total -= v
			subtractedKeys[k] = v
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

func MapValueToString[T Number](v T) string {
	if f, ok := any(v).(float64); ok {
		return FloatToString(f)
	}
	return strconv.FormatInt(any(v).(int64), 10)
}

// String returns the string representation of the receiver Map.
// This is part of the Value interface.
// map:key,k1:v1,k2:v2
func (m *Map[T]) String() string {
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
		b.WriteString(MapValueToString(m.m[k]))
	}
	return b.String()
}

// ParseMapFromString parses a map value string into a map object.
// Note that the values are always parsed as floats, so even a map with integer
// values will become a float map.
// For example:
// "map:code,200:10123,404:21" will be parsed as:
// "map:code 200:10123.000 404:21.000".
func ParseMapFromString[T Number](mapValue string) (*Map[T], error) {
	var v T // We use this to determine the actual type

	tokens := strings.Split(mapValue, ",")
	if len(tokens) < 1 {
		return nil, errors.New("bad map value")
	}

	kv := strings.Split(tokens[0], ":")
	if kv[0] != "map" {
		return nil, errors.New("map value doesn't start with map:<key>")
	}

	m := newMap[T](kv[1])

	for _, tok := range tokens[1:] {
		kv := strings.Split(tok, ":")
		if len(kv) != 2 {
			return nil, errors.New("bad map value token: " + tok)
		}
		if _, ok := any(v).(int64); ok {
			i, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("could not convert map key value %s to a int64: %v", kv[1], err)
			}
			m.IncKeyBy(kv[0], T(i))
		} else {
			f, err := strconv.ParseFloat(kv[1], 64)
			if err != nil {
				return nil, fmt.Errorf("could not convert map key value %s to a float64: %v", kv[1], err)
			}
			m.IncKeyBy(kv[0], T(f))
		}
	}

	return m, nil
}
