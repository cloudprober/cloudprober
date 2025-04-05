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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

var (
	calledOnce   bool
	calledOnceMu sync.RWMutex
)

func tcSetCalledOnce(b bool) {
	calledOnceMu.Lock()
	defer calledOnceMu.Unlock()
	calledOnce = b
}

func tcCalledOnce() bool {
	calledOnceMu.RLock()
	defer calledOnceMu.RUnlock()
	return calledOnce
}

func getTokenFunc(exp time.Time, failSecond bool) func() (*oauth2.Token, error) {
	return func() (*oauth2.Token, error) {
		suffix := ""
		if tcCalledOnce() {
			if failSecond {
				return nil, fmt.Errorf("failed to renew")
			}
			suffix = "_new"
		}
		tcSetCalledOnce(true)
		return &oauth2.Token{AccessToken: "test_token" + suffix, Expiry: exp}, nil
	}
}

func TestTokenCacheToken(t *testing.T) {
	tests := []struct {
		name         string
		expiresIn    time.Duration
		buffer       time.Duration
		wait         time.Duration
		failSecond   bool
		wantNewToken bool
	}{
		{
			name:         "expired_token_1ms_expiry",
			expiresIn:    time.Millisecond,
			buffer:       0,
			wait:         10 * time.Millisecond,
			wantNewToken: true,
		},
		{
			name:         "zero_expiry",
			expiresIn:    -1, // Sets expiry to 0.
			wait:         1 * time.Millisecond,
			wantNewToken: false,
		},
		{
			name:         "unexpired_but_out_of_default_expiry_buffer_60s",
			expiresIn:    time.Second,
			buffer:       time.Minute,
			wait:         time.Millisecond, // < expiry(1s)
			wantNewToken: true,
		},
		{
			name:         "unexpired_and_within_expiry_buffer",
			expiresIn:    time.Second,
			buffer:       0,
			wait:         time.Millisecond, // < expiry(1s)
			wantNewToken: false,
		},
		{
			name:         "unexpired_but_out_of_expiry_buffer_1s",
			expiresIn:    time.Second,
			buffer:       time.Second,
			wait:         time.Millisecond, // < expiry(1s)
			wantNewToken: true,
		},
		{
			name:         "renew_fails_return_cached_token",
			failSecond:   true,
			expiresIn:    time.Millisecond,
			wait:         10 * time.Millisecond,
			wantNewToken: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcSetCalledOnce(false)

			exp := time.Now().Add(tt.expiresIn)
			if tt.expiresIn == -1 {
				exp = time.Time{}
			}
			tc := &tokenCache{
				getToken:            getTokenFunc(exp, tt.failSecond),
				refreshExpiryBuffer: tt.buffer,
				//ignoreExpiryIfZero:  tt.ignoreExpiryIfZero,
			}

			got, err := tc.Token()
			assert.NoError(t, err, "Token()")

			wantToken := "test_token"
			assert.Equal(t, wantToken, got.AccessToken, "access token (1st call)")

			// Prepare for second call
			if tt.wantNewToken {
				wantToken = "test_token_new"
			}
			time.Sleep(tt.wait)
			got, _ = tc.Token()
			assert.Equal(t, wantToken, got.AccessToken, "access token (2nd call)")
		})
	}
}
