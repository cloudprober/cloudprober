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

// Package jwt mints and signs compact-serialization JSON Web Tokens. It is the
// shared signing core used by the Starlark probe's jwt.encode builtin and the
// oauth module's self-signed JWT token source.
package jwt

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
	"fmt"
)

// Encode builds and signs a compact-serialization JWT and returns the token
// string. alg selects the signature:
//
//   - "RS256": RSA PKCS1v15 with SHA-256; key is a PEM-encoded RSA private key
//     (PKCS#1 or PKCS#8).
//   - "HS256": HMAC-SHA256; key is the shared secret.
//
// header holds extra JOSE header fields, merged over {"alg":alg,"typ":"JWT"};
// an "alg" present in header must equal alg, so the header can never disagree
// with how the token is actually signed. Callers own the claim set, including
// any time claims (iat/exp).
func Encode(claims, header map[string]any, key, alg string) (string, error) {
	h := map[string]any{"alg": alg, "typ": "JWT"}
	if a, ok := header["alg"]; ok && a != alg {
		return "", fmt.Errorf("header[\"alg\"]=%v conflicts with algorithm=%q", a, alg)
	}
	for k, v := range header {
		h[k] = v
	}

	headerJSON, err := json.Marshal(h)
	if err != nil {
		return "", fmt.Errorf("marshaling header: %v", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshaling claims: %v", err)
	}

	signingInput := b64(headerJSON) + "." + b64(claimsJSON)
	sig, err := sign(alg, []byte(signingInput), key)
	if err != nil {
		return "", err
	}
	return signingInput + "." + b64(sig), nil
}

func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func sign(alg string, signingInput []byte, key string) ([]byte, error) {
	switch alg {
	case "RS256":
		priv, err := parseRSAKey(key)
		if err != nil {
			return nil, err
		}
		h := sha256.Sum256(signingInput)
		return rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	case "HS256":
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write(signingInput)
		return mac.Sum(nil), nil
	default:
		return nil, fmt.Errorf("unsupported algorithm %q (supported: RS256, HS256)", alg)
	}
}

func parseRSAKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in key")
	}
	if block.Type == "ENCRYPTED PRIVATE KEY" {
		return nil, fmt.Errorf("encrypted private keys are not supported; provide an unencrypted PEM")
	}
	// Snowflake/GCP keys are usually PKCS#8 (openssl genpkey); older tooling
	// emits PKCS#1. Try both.
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing RSA private key (tried PKCS1 and PKCS8): %v", err)
	}
	rsaKey, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is %T, want an RSA private key for RS256", k)
	}
	return rsaKey, nil
}
