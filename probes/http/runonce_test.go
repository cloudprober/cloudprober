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

package http

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/internal/validators"
	configpb "github.com/cloudprober/cloudprober/probes/http/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
)

func TestRunOnce(t *testing.T) {
	tests := []struct {
		name            string
		targets         []endpoint.Endpoint
		setupServer     func(*testing.T) *httptest.Server
		expectedSuccess []bool
	}{
		{
			name:    "single_target_success",
			targets: []endpoint.Endpoint{{Name: "test-target-1"}},
			setupServer: func(t *testing.T) *httptest.Server {
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
				return httptest.NewServer(handler)
			},
			expectedSuccess: []bool{true},
		},
		{
			name: "multiple_targets_mixed_status",
			targets: []endpoint.Endpoint{
				{Name: "test-bad-target"}, // Invalid scheme
				{Name: "test-target-1"},   // Success
				{Name: "test-target-2"},   // Bad status code
			},
			setupServer: func(t *testing.T) *httptest.Server {
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.Host, "test-target-1") {
						w.WriteHeader(http.StatusOK)
					} else {
						w.WriteHeader(http.StatusInternalServerError)
					}
				})
				return httptest.NewServer(handler)
			},
			expectedSuccess: []bool{false, true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test server
			ts := tt.setupServer(t)
			serverAddr := ts.Listener.Addr().(*net.TCPAddr)
			defer ts.Close()

			// Probe options with test configuration
			opts := options.DefaultOptions()
			opts.Validators = []*validators.Validator{
				{
					Name: "statuscode-validator",
					Validate: func(input *validators.Input) (bool, error) {
						if input.Response.(*http.Response).StatusCode != http.StatusOK {
							return false, nil
						}
						return true, nil
					},
				},
			}
			opts.ProbeConf = &configpb.ProbeConf{}
			for i := range tt.targets {
				tt.targets[i].IP, tt.targets[i].Port = serverAddr.IP, serverAddr.Port
				if tt.targets[i].Name == "test-bad-target" {
					tt.targets[i].Labels = map[string]string{"__cp_scheme__": "ftp"}
				}
			}
			opts.Targets = targets.StaticEndpoints(tt.targets)

			p := &Probe{}
			assert.NoError(t, p.Init("test-probe", opts))

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			results := p.RunOnce(ctx)

			assert.Equal(t, len(tt.targets), len(results), "Number of results should match number of targets")
			for i, target := range tt.targets {
				assert.NotNil(t, results[i], "Result should not be nil")
				assert.Equal(t, target.Name, results[i].Target.Name, "Target name should match")

				if tt.expectedSuccess[i] {
					assert.NotNil(t, results[i].Metrics, "Metrics for target %s should not be nil", target.Name)
				}
				assert.Equal(t, tt.expectedSuccess[i], results[i].Success, "Target %s success", target.Name)
			}
		})
	}
}
