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
	"fmt"
	"time"

	"github.com/cloudprober/cloudprober/common/jwt"
	starlarklib "go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// jwt.encode(payload, key, algorithm="RS256", headers=None, lifetime=60) mints
// and signs a compact-serialization JWT (via common/jwt). It exists for APIs
// that take a self-signed JWT directly as the bearer credential — Google
// service-account JWTs are the driving case, Snowflake's SQL REST API key-pair
// auth is the other.
//
// The private key (for RS*) comes in as a PEM string, typically via the vars
// module, e.g. jwt.encode(claims, vars.get("sa_private_key")). Scripts have no
// clock, so iat/exp are filled in from lifetime unless the payload sets them
// (see jwtEncode).
//
//	def probe(target):
//	    email = vars.get("sa_email")
//	    claims = {"iss": email, "sub": email, "aud": "https://storage.googleapis.com/"}
//	    tok = jwt.encode(claims, vars.get("sa_private_key"),
//	                     headers={"kid": vars.get("sa_private_key_id")}, lifetime=3600)
//	    r = http.get(url, headers={"Authorization": "Bearer " + tok})

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
	// Scripts have no clock, so fill the time claims here. An explicit
	// (non-None) iat/exp in the payload wins; a None value counts as unset, so
	// `exp = None` still gets a default rather than serializing as null.
	now := time.Now().Unix()
	if v, ok := claims["iat"]; !ok || v == nil {
		claims["iat"] = now
	}
	if lifetime > 0 {
		if v, ok := claims["exp"]; !ok || v == nil {
			claims["exp"] = now + int64(lifetime)
		}
	}

	var header map[string]any
	if headersV != starlarklib.None {
		header, err = jwtDict(headersV, "headers")
		if err != nil {
			return nil, err
		}
	}

	tok, err := jwt.Encode(claims, header, key, alg)
	if err != nil {
		return nil, fmt.Errorf("jwt.encode: %v", err)
	}
	return starlarklib.String(tok), nil
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
