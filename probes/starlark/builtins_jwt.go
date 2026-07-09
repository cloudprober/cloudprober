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
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"time"

	starlarklib "go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// jwt.encode(payload, key, algorithm="RS256", headers=None, lifetime=60) mints
// and signs a compact-serialization JWT. It exists for APIs that take a
// self-signed JWT directly as the bearer credential — Snowflake's SQL REST API
// key-pair auth is the driving case, GCP service-account JWTs are the other.
// It is deliberately not part of the oauth module: there is no token exchange
// here, the script signs its own short-lived token and sends it.
//
// The private key (for RS*) comes in as a PEM string, typically via the vars
// module, e.g. jwt.encode(claims, vars.get("sf_private_key")). Scripts have no
// clock, so iat/exp are filled in from lifetime unless the payload sets them
// (see jwtEncode). Cache the result across runs with the state builtin rather
// than re-signing every run — Snowflake tokens are valid up to an hour.
//
//	def probe(target):
//	    q = vars.get("sf_account") + "." + vars.get("sf_user")   # UPPERCASE
//	    claims = {"iss": q + "." + vars.get("sf_pubkey_fp"), "sub": q}
//	    tok = jwt.encode(claims, vars.get("sf_private_key"), lifetime=3600)
//	    r = http.post(url, headers={"Authorization": "Bearer " + tok})

func jwtModule() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "jwt",
		Members: starlarklib.StringDict{
			"encode": starlarklib.NewBuiltin("jwt.encode", jwtEncode),
		},
	}
}

func jwtEncode(_ *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	var payloadV starlarklib.Value
	var key string
	alg := "RS256"
	var headersV starlarklib.Value = starlarklib.None
	lifetime := 60
	if err := starlarklib.UnpackArgs("jwt.encode", args, kwargs,
		"payload", &payloadV,
		"key", &key,
		"algorithm?", &alg,
		"headers?", &headersV,
		"lifetime?", &lifetime,
	); err != nil {
		return nil, err
	}

	claims, err := jwtDict(payloadV, "payload")
	if err != nil {
		return nil, err
	}
	// Scripts have no clock, so fill the time claims here. Anything the payload
	// already sets wins — an explicit iat/exp is never overwritten.
	now := time.Now().Unix()
	if _, ok := claims["iat"]; !ok {
		claims["iat"] = now
	}
	if lifetime > 0 {
		if _, ok := claims["exp"]; !ok {
			claims["exp"] = now + int64(lifetime)
		}
	}

	header := map[string]interface{}{"alg": alg, "typ": "JWT"}
	if headersV != starlarklib.None {
		extra, err := jwtDict(headersV, "headers")
		if err != nil {
			return nil, err
		}
		for k, v := range extra {
			header[k] = v
		}
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return nil, fmt.Errorf("jwt.encode: marshaling header: %v", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("jwt.encode: marshaling payload: %v", err)
	}

	signingInput := jwtBase64(headerJSON) + "." + jwtBase64(claimsJSON)
	sig, err := jwtSign(alg, []byte(signingInput), key)
	if err != nil {
		return nil, fmt.Errorf("jwt.encode: %v", err)
	}
	return starlarklib.String(signingInput + "." + jwtBase64(sig)), nil
}

// jwtDict converts a Starlark dict argument to a Go map via the shared
// starlarkToGo, rejecting anything that isn't a dict.
func jwtDict(v starlarklib.Value, name string) (map[string]interface{}, error) {
	g, err := starlarkToGo(v)
	if err != nil {
		return nil, fmt.Errorf("jwt.encode: %s: %v", name, err)
	}
	m, ok := g.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("jwt.encode: %s must be a dict, got %s", name, v.Type())
	}
	return m, nil
}

func jwtBase64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// jwtSign signs signingInput for the given JOSE alg. RS256 (RSA-PKCS1v15 +
// SHA-256) is the Snowflake/GCP case; HS256 (HMAC-SHA256) is the shared-secret
// companion. Add more alg entries here as concrete APIs need them.
func jwtSign(alg string, signingInput []byte, key string) ([]byte, error) {
	switch alg {
	case "RS256":
		priv, err := jwtParseRSAKey(key)
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

func jwtParseRSAKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in key")
	}
	if block.Type == "ENCRYPTED PRIVATE KEY" {
		return nil, fmt.Errorf("encrypted private keys are not supported; provide an unencrypted PEM")
	}
	// Snowflake keys are usually PKCS#8 (openssl genpkey); older tooling emits
	// PKCS#1. Try both.
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
