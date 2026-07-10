// Copyright 2026 The Cloudprober Authors.
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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/common/oauth/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func rsaKeyPEM(t *testing.T) string {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

func jwtClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	cb, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(cb, &claims))
	return claims
}

func TestJWTSource(t *testing.T) {
	cfg := &configpb.Config{
		Source: &configpb.Config_Jwt{
			Jwt: &configpb.JWTSource{
				PrivateKey:  proto.String(rsaKeyPEM(t)),
				LifetimeSec: proto.Int32(3600),
				Claims:      map[string]string{"iss": "acct.user", "sub": "acct.user"},
				Header:      map[string]string{"kid": "k1"},
			},
		},
	}

	ts, err := TokenSourceFromConfig(cfg, &logger.Logger{})
	require.NoError(t, err)

	tok, err := ts.Token()
	require.NoError(t, err)
	assert.NotEmpty(t, tok.AccessToken)
	// Expiry is stamped so the cache re-mints before it lapses.
	assert.WithinDuration(t, time.Now().Add(3600*time.Second), tok.Expiry, 30*time.Second)

	claims := jwtClaims(t, tok.AccessToken)
	assert.Equal(t, "acct.user", claims["iss"])
	assert.Equal(t, "acct.user", claims["sub"])
	assert.NotNil(t, claims["iat"])
	assert.NotNil(t, claims["exp"])
}

// TestJWTSource_ReservedTimeClaims pins that iat/exp in the configured claims
// are ignored: they're derived from lifetime_sec so they stay in sync with the
// token's Expiry, and a string value would otherwise produce a spec-invalid exp.
func TestJWTSource_ReservedTimeClaims(t *testing.T) {
	cfg := &configpb.Config{
		Source: &configpb.Config_Jwt{
			Jwt: &configpb.JWTSource{
				PrivateKey:  proto.String(rsaKeyPEM(t)),
				LifetimeSec: proto.Int32(3600),
				Claims:      map[string]string{"sub": "x", "exp": "not-a-number", "iat": "nope"},
			},
		},
	}
	ts, err := TokenSourceFromConfig(cfg, &logger.Logger{})
	require.NoError(t, err)
	tok, err := ts.Token()
	require.NoError(t, err)

	claims := jwtClaims(t, tok.AccessToken)
	// exp/iat must be the numeric derived values, not the user's strings.
	_, expIsNum := claims["exp"].(float64)
	_, iatIsNum := claims["iat"].(float64)
	assert.True(t, expIsNum, "exp must be numeric, got %T", claims["exp"])
	assert.True(t, iatIsNum, "iat must be numeric, got %T", claims["iat"])
}

// TestJWTSource_BadKey pins that a bad key fails at construction (verifyToken
// mints an initial token), so config errors surface early.
func TestJWTSource_BadKey(t *testing.T) {
	cfg := &configpb.Config{
		Source: &configpb.Config_Jwt{
			Jwt: &configpb.JWTSource{
				PrivateKey: proto.String("not-a-pem"),
				Claims:     map[string]string{"sub": "x"},
			},
		},
	}
	_, err := TokenSourceFromConfig(cfg, &logger.Logger{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no PEM block")
}
