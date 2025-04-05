// Copyright 2023 The Cloudprober Authors.
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

package oauth

import (
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"golang.org/x/oauth2"
)

type tokenCache struct {
	tok                 *oauth2.Token
	mu                  sync.RWMutex
	refreshExpiryBuffer time.Duration
	getToken            func() (*oauth2.Token, error)
	l                   *logger.Logger

	// refreshingOnInterval means token is getting refreshed at a regular
	// interval, through refresh_interval_sec option. Cache doesn't refresh the
	// token in this case.
	refreshingOnInterval bool
}

func (tc *tokenCache) setToken(tok *oauth2.Token) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.tok = tok
}

func (tc *tokenCache) Token() (*oauth2.Token, error) {
	tc.mu.RLock()
	tok := tc.tok
	tc.mu.RUnlock()

	if tc.refreshingOnInterval {
		return tok, nil
	}

	if tok != nil {
		// Dealing with simple non-JSON tokens. We (cache) don't refresh them.
		// We'll mostly not be here in this case, as refreshingOnInterval will
		// be set to true for such token sources.
		if tok.Expiry.IsZero() {
			return tok, nil
		}
		if time.Until(tok.Expiry) > tc.refreshExpiryBuffer {
			return tok, nil
		}
	}

	newTok, err := tc.getToken()
	if err != nil {
		if tok != nil {
			tc.l.Errorf("oauth: failed to refresh the token: %v, returning stale token", err)
			return tok, nil
		}
		return nil, err
	}
	tc.setToken(newTok)
	return newTok, nil
}
