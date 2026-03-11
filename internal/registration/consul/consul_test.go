// Copyright 2025 The Cloudprober Authors.
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

package consul

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
)

// mockConsul records Consul API calls for assertions in tests.
type mockConsul struct {
	mu              sync.Mutex
	server          *httptest.Server
	registrations   []serviceRegistration
	deregistrations []string
	lastToken       string
}

func newMockConsul(t *testing.T) *mockConsul {
	t.Helper()
	mc := &mockConsul{}
	mc.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mc.mu.Lock()
		defer mc.mu.Unlock()

		mc.lastToken = r.Header.Get("X-Consul-Token")

		switch {
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/agent/service/register"):
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var reg serviceRegistration
			if err := json.Unmarshal(body, &reg); err != nil {
				t.Logf("failed to unmarshal registration: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			mc.registrations = append(mc.registrations, reg)
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/agent/service/deregister/"):
			parts := strings.Split(r.URL.Path, "/")
			mc.deregistrations = append(mc.deregistrations, parts[len(parts)-1])
			w.WriteHeader(http.StatusOK)

		default:
			t.Logf("unhandled request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return mc
}

func (mc *mockConsul) addr() string {
	// Strip the "http://" prefix to get host:port.
	return strings.TrimPrefix(mc.server.URL, "http://")
}

func newTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
	return logger.New()
}

func TestRegister(t *testing.T) {
	mc := newMockConsul(t)
	defer mc.server.Close()

	reg := &registration{
		consulAddr:  mc.addr(),
		serviceID:   "cloudprober-host-9313",
		serviceName: "cloudprober",
		servicePort: 9313,
		serviceAddr: "host.example.com",
		l:           newTestLogger(t),
	}

	if err := reg.register(); err != nil {
		t.Fatalf("register() error: %v", err)
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	if len(mc.registrations) != 1 {
		t.Fatalf("expected 1 registration, got %d", len(mc.registrations))
	}

	got := mc.registrations[0]
	if got.ID != reg.serviceID {
		t.Errorf("ID: want %q, got %q", reg.serviceID, got.ID)
	}
	if got.Name != reg.serviceName {
		t.Errorf("Name: want %q, got %q", reg.serviceName, got.Name)
	}
	if got.Port != reg.servicePort {
		t.Errorf("Port: want %d, got %d", reg.servicePort, got.Port)
	}
	if got.Address != reg.serviceAddr {
		t.Errorf("Address: want %q, got %q", reg.serviceAddr, got.Address)
	}
	wantHealthURL := "http://host.example.com:9313/health"
	if got.Check == nil || got.Check.HTTP != wantHealthURL {
		t.Errorf("health check HTTP: want %q, got %v", wantHealthURL, got.Check)
	}
}

func TestRegisterWithTags(t *testing.T) {
	mc := newMockConsul(t)
	defer mc.server.Close()

	// Save and restore flag values to avoid polluting other tests.
	orig := *serviceTags
	defer func() { *serviceTags = orig }()
	*serviceTags = "cloudprober, monitoring, prod"

	reg := &registration{
		consulAddr:  mc.addr(),
		serviceID:   "test-id",
		serviceName: "cloudprober",
		servicePort: 9313,
		serviceAddr: "host.example.com",
		l:           newTestLogger(t),
	}

	if err := reg.register(); err != nil {
		t.Fatalf("register() error: %v", err)
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	gotTags := mc.registrations[0].Tags
	wantTags := []string{"cloudprober", "monitoring", "prod"}
	if len(gotTags) != len(wantTags) {
		t.Fatalf("tags: want %v, got %v", wantTags, gotTags)
	}
	for i, tag := range wantTags {
		if gotTags[i] != tag {
			t.Errorf("tag[%d]: want %q, got %q", i, tag, gotTags[i])
		}
	}
}

func TestDeregister(t *testing.T) {
	mc := newMockConsul(t)
	defer mc.server.Close()

	reg := &registration{
		consulAddr:  mc.addr(),
		serviceID:   "cloudprober-host-9313",
		serviceName: "cloudprober",
		servicePort: 9313,
		serviceAddr: "host.example.com",
		l:           newTestLogger(t),
	}

	reg.deregister()

	mc.mu.Lock()
	defer mc.mu.Unlock()

	if len(mc.deregistrations) != 1 {
		t.Fatalf("expected 1 deregistration, got %d", len(mc.deregistrations))
	}
	if mc.deregistrations[0] != reg.serviceID {
		t.Errorf("deregistered ID: want %q, got %q", reg.serviceID, mc.deregistrations[0])
	}
}

func TestACLToken(t *testing.T) {
	mc := newMockConsul(t)
	defer mc.server.Close()

	reg := &registration{
		consulAddr:  mc.addr(),
		serviceID:   "test-id",
		serviceName: "cloudprober",
		servicePort: 9313,
		serviceAddr: "host.example.com",
		token:       "secret-acl-token",
		l:           newTestLogger(t),
	}

	if err := reg.register(); err != nil {
		t.Fatalf("register() error: %v", err)
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.lastToken != "secret-acl-token" {
		t.Errorf("X-Consul-Token: want %q, got %q", "secret-acl-token", mc.lastToken)
	}
}

func TestStartDisabled(t *testing.T) {
	// --consul_registration is false by default; Start should be a no-op.
	orig := *enabled
	defer func() { *enabled = orig }()
	*enabled = false

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := Start(ctx, newTestLogger(t)); err != nil {
		t.Fatalf("Start() with registration disabled returned error: %v", err)
	}
}

func TestStartRegistersAndDeregisters(t *testing.T) {
	mc := newMockConsul(t)
	defer mc.server.Close()

	// Temporarily override all relevant flags.
	origEnabled := *enabled
	origAddr := *consulAddr
	origSvcName := *serviceName
	origSvcID := *serviceID
	origSvcPort := *servicePort
	origSvcAddr := *serviceAddr
	origDeregOnShutdown := *deregisterOnShutdown
	defer func() {
		*enabled = origEnabled
		*consulAddr = origAddr
		*serviceName = origSvcName
		*serviceID = origSvcID
		*servicePort = origSvcPort
		*serviceAddr = origSvcAddr
		*deregisterOnShutdown = origDeregOnShutdown
	}()

	*enabled = true
	*consulAddr = mc.addr()
	*serviceName = "cloudprober"
	*serviceID = "test-cloudprober-id"
	*servicePort = 9313
	*serviceAddr = "test.example.com"
	*deregisterOnShutdown = true

	ctx, cancel := context.WithCancel(context.Background())

	if err := Start(ctx, newTestLogger(t)); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	mc.mu.Lock()
	if len(mc.registrations) != 1 {
		mc.mu.Unlock()
		t.Fatalf("expected 1 registration after Start, got %d", len(mc.registrations))
	}
	mc.mu.Unlock()

	// Cancel context to trigger deregistration and give the goroutine time to run.
	cancel()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mc.mu.Lock()
		n := len(mc.deregistrations)
		mc.mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()
	if len(mc.deregistrations) != 1 || mc.deregistrations[0] != "test-cloudprober-id" {
		t.Errorf("deregistrations: want [test-cloudprober-id], got %v", mc.deregistrations)
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		raw  string
		want []string
	}{
		{"cloudprober", []string{"cloudprober"}},
		{"a, b, c", []string{"a", "b", "c"}},
		{" a , b ", []string{"a", "b"}},
		{"", nil},
		{"  ,  ", nil},
	}
	for _, tc := range tests {
		got := parseTags(tc.raw)
		if len(got) != len(tc.want) {
			t.Errorf("parseTags(%q) = %v, want %v", tc.raw, got, tc.want)
			continue
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Errorf("parseTags(%q)[%d] = %q, want %q", tc.raw, i, got[i], tc.want[i])
			}
		}
	}
}
