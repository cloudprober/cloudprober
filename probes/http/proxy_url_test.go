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

package http

import (
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/probes/http/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestProxyURLParseErrorHidesPassword(t *testing.T) {
	p := &Probe{
		// Invalid port, so url.Parse rejects it.
		c:    &configpb.ProbeConf{ProxyUrl: proto.String("http://user:hunter2@proxy.example.com:notaport/")},
		opts: &options.Options{Timeout: time.Second},
	}

	_, err := p.getTransport()
	if err == nil {
		t.Fatal("expected a parse error for the malformed proxy_url")
	}

	assert.NotContains(t, err.Error(), "hunter2", "proxy password leaked into the error")
	// The reason and the rest of the URL survive, so the message stays useful.
	assert.Contains(t, err.Error(), "invalid port")
	assert.Contains(t, err.Error(), "proxy.example.com")
}
