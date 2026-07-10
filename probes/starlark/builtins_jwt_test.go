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

package starlark

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/probes/common/sched"
	configpb "github.com/cloudprober/cloudprober/probes/starlark/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	starlarklib "go.starlark.net/starlark"
)

// The signing core (RS256/HS256, PEM parsing, base64url, the alg-conflict
// guard) is tested in common/jwt. These tests cover only what the builtin adds
// on top: arg handling, the iat/exp auto-fill (scripts have no clock), and the
// end-to-end script path.

// callJWTEncode invokes the jwt.encode builtin directly (it needs no
// thread-locals) and returns the token string.
func callJWTEncode(t *testing.T, payload *starlarklib.Dict, key string, kwargs ...starlarklib.Tuple) (string, error) {
	t.Helper()
	v, err := jwtEncode(nil, nil, starlarklib.Tuple{payload, starlarklib.String(key)}, kwargs)
	if err != nil {
		return "", err
	}
	s, ok := starlarklib.AsString(v)
	require.True(t, ok, "jwt.encode returned non-string %s", v.Type())
	return s, nil
}

// decodeJWT splits a compact JWT and returns the parsed header, parsed claims,
// the signing input, and the raw signature bytes.
func decodeJWT(t *testing.T, token string) (header, claims map[string]interface{}, signingInput string, sig []byte) {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3, "JWT must have 3 dot-separated parts")

	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(hb, &header))

	cb, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(cb, &claims))

	sig, err = base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)

	return header, claims, parts[0] + "." + parts[1], sig
}

func dict(t *testing.T, kv map[string]starlarklib.Value) *starlarklib.Dict {
	t.Helper()
	d := starlarklib.NewDict(len(kv))
	for k, v := range kv {
		require.NoError(t, d.SetKey(starlarklib.String(k), v))
	}
	return d
}

func genRSAKeyPEM(t *testing.T) (string, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	return string(pemBytes), &priv.PublicKey
}

// TestJWTEncode_AutoFillsTimeClaims covers the clock behavior: iat/exp are
// filled from lifetime when absent, and an explicit iat/exp is left untouched.
func TestJWTEncode_AutoFillsTimeClaims(t *testing.T) {
	keyPEM, _ := genRSAKeyPEM(t)

	before := time.Now().Unix()
	token, err := callJWTEncode(t, dict(t, map[string]starlarklib.Value{"sub": starlarklib.String("x")}), keyPEM,
		starlarklib.Tuple{starlarklib.String("lifetime"), starlarklib.MakeInt(3600)})
	require.NoError(t, err)
	after := time.Now().Unix()

	_, claims, _, _ := decodeJWT(t, token)
	iat := int64(claims["iat"].(float64))
	exp := int64(claims["exp"].(float64))
	assert.GreaterOrEqual(t, iat, before)
	assert.LessOrEqual(t, iat, after)
	assert.Equal(t, iat+3600, exp, "exp should be iat + lifetime")

	// Explicit exp/iat in the payload must win over auto-fill.
	token2, err := callJWTEncode(t, dict(t, map[string]starlarklib.Value{
		"sub": starlarklib.String("x"),
		"iat": starlarklib.MakeInt(1000),
		"exp": starlarklib.MakeInt(2000),
	}), keyPEM)
	require.NoError(t, err)
	_, claims2, _, _ := decodeJWT(t, token2)
	assert.Equal(t, int64(1000), int64(claims2["iat"].(float64)))
	assert.Equal(t, int64(2000), int64(claims2["exp"].(float64)))
}

// TestJWTEncode_NoneTimeClaim pins that an explicit None iat/exp is treated as
// unset and auto-filled, not serialized as JSON null.
func TestJWTEncode_NoneTimeClaim(t *testing.T) {
	keyPEM, _ := genRSAKeyPEM(t)
	token, err := callJWTEncode(t, dict(t, map[string]starlarklib.Value{
		"sub": starlarklib.String("x"),
		"exp": starlarklib.None,
		"iat": starlarklib.None,
	}), keyPEM, starlarklib.Tuple{starlarklib.String("lifetime"), starlarklib.MakeInt(3600)})
	require.NoError(t, err)

	_, claims, _, _ := decodeJWT(t, token)
	require.NotNil(t, claims["exp"], "exp should be auto-filled, not null")
	require.NotNil(t, claims["iat"], "iat should be auto-filled, not null")
	assert.Equal(t, int64(claims["iat"].(float64))+3600, int64(claims["exp"].(float64)))
}

// TestJWTEncode_PayloadNotDict covers the builtin's own argument validation.
func TestJWTEncode_PayloadNotDict(t *testing.T) {
	_, err := jwtEncode(nil, nil,
		starlarklib.Tuple{starlarklib.String("nope"), starlarklib.String("key")}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "payload must be a dict")
}

// TestJWTEncode_InScript runs the builtin through a real script, exercising the
// registration and the vars-supplied key path, and verifies the token signs.
func TestJWTEncode_InScript(t *testing.T) {
	keyPEM, pub := genRSAKeyPEM(t)
	source := `
def probe(target):
    q = vars.get("account") + "." + vars.get("user")
    claims = {"iss": q + "." + vars.get("fp"), "sub": q}
    tok = jwt.encode(claims, vars.get("key"), lifetime=3600)
    state.set("token", tok)
`
	opts := newOpts(t, "example.com", source)
	opts.ProbeConf.(*configpb.ProbeConf).Vars = map[string]string{
		"account": "ACCT",
		"user":    "USER",
		"fp":      "SHA256:abc",
		"key":     keyPEM,
	}
	p := &Probe{}
	require.NoError(t, p.Init("script-jwt", opts))

	runReq := &sched.RunProbeForTargetRequest{Target: endpoint.Endpoint{Name: "example.com"}}
	runProbeWith(t, p, runReq)
	require.NoError(t, runReq.LastRun.Error)

	bucket := runReq.TargetState.(*stateBucket)
	token := bucket.values["token"].(string)
	header, claims, signingInput, sig := decodeJWT(t, token)
	assert.Equal(t, "RS256", header["alg"])
	assert.Equal(t, "ACCT.USER.SHA256:abc", claims["iss"])
	h := sha256.Sum256([]byte(signingInput))
	assert.NoError(t, rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig))
}
