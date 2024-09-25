// Copyright 2017-2024 The Cloudprober Authors.
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

// Package resolver provides a caching, non-blocking DNS resolver. All requests
// for cached resources are returned immediately and if cache has expired, an
// offline goroutine is fired to update it.
package resolver

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type retryableError struct {
	error
}

type cacheRecord struct {
	mu               sync.RWMutex
	name             string
	version          ipVersion
	ip               net.IP
	err              error
	lastUpdatedAt    time.Time
	updateInProgress bool
	callInit         sync.Once
}

type cacheRecordKey struct {
	name    string
	version ipVersion
}

type resolveFunc func(string, ipVersion) ([]net.IP, error)

// getCacheRecord returns the cache record for the target.
// It must be kept light, as it blocks the main mutex of the map.
func (r *resolverImpl) getCacheRecord(name string, version ipVersion) *cacheRecord {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := cacheRecordKey{name: name, version: version}
	cr := r.cache[key]
	// This will happen only once for a given name.
	if cr == nil {
		cr = &cacheRecord{
			name:    name,
			version: version,
			err:     &retryableError{errors.New("cache record not initialized yet")},
		}
		r.cache[key] = cr
	}
	return cr
}

// refresh refreshes the cacheRecord by making a call to the provided "resolve" function.
func (cr *cacheRecord) refresh(resolveF resolveFunc, refreshed chan<- bool) {
	// Note that we call backend's resolve outside of the mutex locks and take the lock again
	// to update the cache record once we have the results from the backend.
	ips, err := resolveF(cr.name, cr.version)

	cr.mu.Lock()
	defer cr.mu.Unlock()
	if refreshed != nil {
		refreshed <- true
	}
	if err != nil {
		cr.err = &retryableError{err}
	} else {
		cr.err = nil
	}
	cr.updateInProgress = false
	// If we have an error, we don't update the cache record so that callers
	// can use cached IP addresses if they want.
	if err != nil {
		return
	}
	cr.lastUpdatedAt = time.Now()
	if len(ips) == 0 {
		cr.err = fmt.Errorf("no %s IPs found for %s", cr.version.Network(), cr.name)
		return
	}
	cr.ip = ips[0]
}

func (cr *cacheRecord) shouldUpdateNow(maxAge time.Duration) bool {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	if cr.updateInProgress {
		return false
	}
	if time.Since(cr.lastUpdatedAt) >= maxAge {
		return true
	}
	if cr.err != nil {
		if _, ok := cr.err.(*retryableError); ok {
			return true
		}
	}
	return false
}

// refreshIfRequired does most of the work. Overall goal is to minimize the
// lock period of the cache record. To that end, if the cache record needs
// updating, we do that with the mutex unlocked.
//
// If cache record is new, blocks until it's resolved for the first time.
// If cache record needs updating, kicks off refresh asynchronously.
// If cache record is already being updated or fresh enough, returns immediately.
func (cr *cacheRecord) refreshIfRequired(resolveF resolveFunc, maxAge time.Duration, refreshedCh chan<- bool) {
	cr.callInit.Do(func() { cr.refresh(resolveF, refreshedCh) })

	// Cache record is old and no update in progress, issue a request to update.
	if cr.shouldUpdateNow(maxAge) {
		cr.mu.Lock()
		cr.updateInProgress = true
		cr.mu.Unlock()
		go cr.refresh(resolveF, refreshedCh)
	} else if refreshedCh != nil {
		refreshedCh <- false
	}
}
