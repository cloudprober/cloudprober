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
	"errors"
	"os"
	"path/filepath"
	"testing"

	oauthpb "github.com/cloudprober/cloudprober/common/oauth/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	configpb "github.com/cloudprober/cloudprober/probes/starlark/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	starlarklib "go.starlark.net/starlark"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"
)

// stubTokenSource returns a fixed token (or error), so the builtin tests cover
// name resolution and formatting without real token infrastructure. The
// TokenSourceFromConfig / refresh / caching path is tested in common/oauth.
type stubTokenSource struct {
	tok *oauth2.Token
	err error
}

func (s stubTokenSource) Token() (*oauth2.Token, error) { return s.tok, s.err }

func accessTS(token string) oauth2.TokenSource {
	return stubTokenSource{tok: &oauth2.Token{AccessToken: token}}
}

// oauthThread builds a thread carrying the given identities plus a logger, the
// two thread-locals the oauth builtins read.
func oauthThread(ids map[string]*oauthIdentity) *starlarklib.Thread {
	th := &starlarklib.Thread{}
	th.SetLocal(threadOAuthKey, ids)
	th.SetLocal(threadLoggerKey, newLoggerHolder(&logger.Logger{}))
	return th
}

func callOAuth(t *testing.T, fn func(*starlarklib.Thread, *starlarklib.Builtin, starlarklib.Tuple, []starlarklib.Tuple) (starlarklib.Value, error), name string, th *starlarklib.Thread, args ...starlarklib.Value) (string, error) {
	t.Helper()
	b := starlarklib.NewBuiltin(name, fn)
	v, err := fn(th, b, starlarklib.Tuple(args), nil)
	if err != nil {
		return "", err
	}
	s, ok := starlarklib.AsString(v)
	require.True(t, ok, "%s returned non-string %s", name, v.Type())
	return s, nil
}

// TestOAuthTokenAndHeader covers the core distinction: token() is raw, header()
// is formatted via token_type_format.
func TestOAuthTokenAndHeader(t *testing.T) {
	ids := map[string]*oauthIdentity{
		"api": {ts: accessTS("abc123"), format: "Bearer %s"},
	}
	th := oauthThread(ids)

	tok, err := callOAuth(t, oauthToken, "oauth.token", th)
	require.NoError(t, err)
	assert.Equal(t, "abc123", tok)

	hdr, err := callOAuth(t, oauthHeader, "oauth.header", th)
	require.NoError(t, err)
	assert.Equal(t, "Bearer abc123", hdr)
}

// TestOAuthHeaderCustomFormat pins that token_type_format flows through header().
func TestOAuthHeaderCustomFormat(t *testing.T) {
	ids := map[string]*oauthIdentity{
		"api": {ts: accessTS("xyz"), format: "token %s"},
	}
	hdr, err := callOAuth(t, oauthHeader, "oauth.header", oauthThread(ids))
	require.NoError(t, err)
	assert.Equal(t, "token xyz", hdr)
}

// TestOAuthNameResolution covers omitted name (allowed only with one config),
// explicit selection, unknown names, and the no-configs case.
func TestOAuthNameResolution(t *testing.T) {
	two := map[string]*oauthIdentity{
		"a": {ts: accessTS("tok-a"), format: "Bearer %s"},
		"b": {ts: accessTS("tok-b"), format: "Bearer %s"},
	}

	// Explicit name picks the right identity.
	tok, err := callOAuth(t, oauthToken, "oauth.token", oauthThread(two), starlarklib.String("b"))
	require.NoError(t, err)
	assert.Equal(t, "tok-b", tok)

	// Omitting the name with multiple configs is an error listing the names.
	_, err = callOAuth(t, oauthToken, "oauth.token", oauthThread(two))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "2 oauth_configs")
	assert.Contains(t, err.Error(), "a, b")

	// Unknown name errors and lists what's configured.
	_, err = callOAuth(t, oauthToken, "oauth.token", oauthThread(two), starlarklib.String("c"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `named "c"`)

	// No configs at all.
	_, err = callOAuth(t, oauthToken, "oauth.token", oauthThread(nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no oauth_configs configured")
}

// TestOAuthTokenFetchError surfaces a token-source failure as a Starlark error
// (no <token-missing> fallback).
func TestOAuthTokenFetchError(t *testing.T) {
	ids := map[string]*oauthIdentity{
		"api": {ts: stubTokenSource{err: errors.New("boom")}, format: "Bearer %s"},
	}
	_, err := callOAuth(t, oauthToken, "oauth.token", oauthThread(ids))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

// TestOAuthInScript runs the builtin end-to-end through Init (building the token
// source from a file config) and a real script.
func TestOAuthInScript(t *testing.T) {
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "token")
	require.NoError(t, os.WriteFile(tokenFile, []byte("filetoken"), 0600))

	source := `
def probe(target):
    state.set("raw", oauth.token())
    state.set("hdr", oauth.header())
`
	opts := newOpts(t, "example.com", source)
	opts.ProbeConf.(*configpb.ProbeConf).OauthConfigs = map[string]*oauthpb.Config{
		"api": {
			Source:          &oauthpb.Config_File{File: tokenFile},
			TokenTypeFormat: proto.String("Bearer %s"),
		},
	}
	p := &Probe{}
	require.NoError(t, p.Init("script-oauth", opts))

	runReq := &sched.RunProbeForTargetRequest{Target: endpoint.Endpoint{Name: "example.com"}}
	runProbeWith(t, p, runReq)
	require.NoError(t, runReq.LastRun.Error)

	bucket := runReq.TargetState.(*stateBucket)
	assert.Equal(t, "filetoken", bucket.values["raw"].(string))
	assert.Equal(t, "Bearer filetoken", bucket.values["hdr"].(string))
}

// TestOAuthInitBadConfig fails probe Init when a token source cannot be built.
func TestOAuthInitBadConfig(t *testing.T) {
	opts := newOpts(t, "example.com", "def probe(target): pass\n")
	opts.ProbeConf.(*configpb.ProbeConf).OauthConfigs = map[string]*oauthpb.Config{
		"api": {
			Source:          &oauthpb.Config_File{File: "/does/not/matter"},
			TokenTypeFormat: proto.String("no placeholder"),
		},
	}
	err := (&Probe{}).Init("script-oauth-bad", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `oauth_configs["api"]`)
}
