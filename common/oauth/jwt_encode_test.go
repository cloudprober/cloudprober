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
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func genRSAKeyPEM(t *testing.T) (string, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), &priv.PublicKey
}

func decode(t *testing.T, token string) (header, claims map[string]any, signingInput string, sig []byte) {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
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

func TestEncodeJWT_RS256(t *testing.T) {
	keyPEM, pub := genRSAKeyPEM(t)
	token, err := encodeJWT(
		map[string]any{"iss": "acct.user", "sub": "acct.user"},
		map[string]any{"kid": "k1"},
		keyPEM, "RS256")
	require.NoError(t, err)

	header, claims, signingInput, sig := decode(t, token)
	assert.Equal(t, "RS256", header["alg"])
	assert.Equal(t, "JWT", header["typ"])
	assert.Equal(t, "k1", header["kid"])
	assert.Equal(t, "acct.user", claims["iss"])

	h := sha256.Sum256([]byte(signingInput))
	assert.NoError(t, rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig))
}

func TestEncodeJWT_PKCS1Key(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	keyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv),
	}))
	token, err := encodeJWT(map[string]any{"sub": "x"}, nil, keyPEM, "RS256")
	require.NoError(t, err)
	_, _, signingInput, sig := decode(t, token)
	h := sha256.Sum256([]byte(signingInput))
	assert.NoError(t, rsa.VerifyPKCS1v15(&priv.PublicKey, crypto.SHA256, h[:], sig))
}

func TestEncodeJWT_HS256(t *testing.T) {
	secret := "s3cret"
	token, err := encodeJWT(map[string]any{"sub": "alice"}, nil, secret, "HS256")
	require.NoError(t, err)
	header, _, signingInput, sig := decode(t, token)
	assert.Equal(t, "HS256", header["alg"])
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	assert.True(t, hmac.Equal(sig, mac.Sum(nil)))
}

func TestEncodeJWT_HeaderAlg(t *testing.T) {
	keyPEM, _ := genRSAKeyPEM(t)

	// Matching alg in header is allowed.
	_, err := encodeJWT(map[string]any{"sub": "x"}, map[string]any{"alg": "RS256"}, keyPEM, "RS256")
	require.NoError(t, err)

	// Conflicting alg is rejected.
	_, err = encodeJWT(map[string]any{"sub": "x"}, map[string]any{"alg": "none"}, keyPEM, "RS256")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicts with algorithm")
}

func TestEncodeJWT_Errors(t *testing.T) {
	_, err := encodeJWT(map[string]any{"sub": "x"}, nil, "secret", "XX999")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported algorithm")

	_, err = encodeJWT(map[string]any{"sub": "x"}, nil, "not-a-pem", "RS256")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no PEM block")
}
