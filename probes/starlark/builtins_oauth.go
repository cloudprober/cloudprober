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
	"sort"
	"strings"

	"github.com/cloudprober/cloudprober/common/oauth"
	oauthpb "github.com/cloudprober/cloudprober/common/oauth/proto"
	"github.com/cloudprober/cloudprober/logger"
	starlarklib "go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"golang.org/x/oauth2"
)

// oauth.token(name=None) returns the raw access token; oauth.header(name=None)
// returns the formatted Authorization value (token_type_format, default
// "Bearer %s"). Tokens come from the probe's oauth_configs; unlike the HTTP
// probe nothing is auto-injected, so the script chooses where the token goes:
//
//	def probe(target):
//	    r = http.get(url, headers={"Authorization": oauth.header("api")})
//	    assert.http_status(r, 200)
//
// name is optional when the probe has exactly one oauth_config; with several
// it must be passed. Token-fetch errors surface as Starlark errors (no
// <token-missing> fallback) so the script controls how to react.

// oauthIdentity is one configured token source plus its token_type_format.
type oauthIdentity struct {
	ts     oauth2.TokenSource
	format string
}

// oauthIdentitiesFromThread returns the per-runtime oauth identities stashed on
// the thread. Unlike httpClientFromThread this does not panic on a miss: a
// probe with no oauth_configs stores a nil map, indistinguishable from unset,
// and the builtins report the "none configured" case themselves.
func oauthIdentitiesFromThread(t *starlarklib.Thread) map[string]*oauthIdentity {
	ids, _ := t.Local(threadOAuthKey).(map[string]*oauthIdentity)
	return ids
}

func oauthModule() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "oauth",
		Members: starlarklib.StringDict{
			"token":  starlarklib.NewBuiltin("oauth.token", oauthToken),
			"header": starlarklib.NewBuiltin("oauth.header", oauthHeader),
		},
	}
}

func oauthToken(t *starlarklib.Thread, b *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	id, err := resolveOAuthIdentity(t, b.Name(), args, kwargs)
	if err != nil {
		return nil, err
	}
	tok, err := oauth.GetToken(id.ts, loggerFromThread(t))
	if err != nil {
		return nil, fmt.Errorf("%s: %v", b.Name(), err)
	}
	return starlarklib.String(tok), nil
}

func oauthHeader(t *starlarklib.Thread, b *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	id, err := resolveOAuthIdentity(t, b.Name(), args, kwargs)
	if err != nil {
		return nil, err
	}
	tok, err := oauth.GetToken(id.ts, loggerFromThread(t))
	if err != nil {
		return nil, fmt.Errorf("%s: %v", b.Name(), err)
	}
	return starlarklib.String(fmt.Sprintf(id.format, tok)), nil
}

// resolveOAuthIdentity unpacks the optional name argument and picks the
// matching identity from the thread. name may be omitted only when exactly one
// config exists.
func resolveOAuthIdentity(t *starlarklib.Thread, fname string, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (*oauthIdentity, error) {
	var name string
	if err := starlarklib.UnpackArgs(fname, args, kwargs, "name?", &name); err != nil {
		return nil, err
	}

	ids := oauthIdentitiesFromThread(t)
	if len(ids) == 0 {
		return nil, fmt.Errorf("%s: probe has no oauth_configs configured", fname)
	}

	if name == "" {
		if len(ids) != 1 {
			return nil, fmt.Errorf("%s: probe has %d oauth_configs (%s); pass a name to select one", fname, len(ids), strings.Join(sortedOAuthNames(ids), ", "))
		}
		for _, id := range ids {
			return id, nil
		}
	}

	id, ok := ids[name]
	if !ok {
		return nil, fmt.Errorf("%s: no oauth config named %q (configured: %s)", fname, name, strings.Join(sortedOAuthNames(ids), ", "))
	}
	return id, nil
}

func sortedOAuthNames(ids map[string]*oauthIdentity) []string {
	names := make([]string, 0, len(ids))
	for name := range ids {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// newOAuthIdentities builds a token source per configured oauth config. It
// returns nil when none are configured. Any build error (bad source, invalid
// token_type_format) fails probe Init.
func newOAuthIdentities(cfgs map[string]*oauthpb.Config, l *logger.Logger) (map[string]*oauthIdentity, error) {
	if len(cfgs) == 0 {
		return nil, nil
	}
	ids := make(map[string]*oauthIdentity, len(cfgs))
	for name, c := range cfgs {
		ts, err := oauth.TokenSourceFromConfig(c, l)
		if err != nil {
			return nil, fmt.Errorf("oauth_configs[%q]: %v", name, err)
		}
		ids[name] = &oauthIdentity{ts: ts, format: c.GetTokenTypeFormat()}
	}
	return ids, nil
}
