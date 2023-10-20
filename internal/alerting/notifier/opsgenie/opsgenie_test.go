// Copyright 2023 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opsgenie

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/cloudprober/cloudprober/internal/alerting/alertinfo"
	configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	os.Setenv(DefaultApiKeyEnvVar, "default-genie-key")
	defer os.Unsetenv(DefaultApiKeyEnvVar)
	os.Setenv("TEST_GENIE_KEY", "test-genie-key")
	defer os.Unsetenv("TEST_GENIE_KEY")

	tests := []struct {
		name    string
		cfg     *configpb.Opsgenie
		want    *Client
		wantErr bool
	}{
		{
			name: "default-key-url",
			cfg:  &configpb.Opsgenie{},
			want: &Client{
				apiURL:     DefaultOpsgenieAPIURL,
				httpClient: http.DefaultClient,
				ogKey:      "default-genie-key",
			},
		},
		{
			name: "config",
			cfg: &configpb.Opsgenie{
				ApiUrl: "https://test-api.opsgenie.com/v2/alerts",
				ApiKey: "config-genie-key",
			},
			want: &Client{
				apiURL:     "https://test-api.opsgenie.com/v2/alerts",
				httpClient: http.DefaultClient,
				ogKey:      "config-genie-key",
			},
		},
		{
			name: "custom-key-env-var",
			cfg: &configpb.Opsgenie{
				ApiKeyEnvVar: "TEST_GENIE_KEY",
			},
			want: &Client{
				apiURL:     DefaultOpsgenieAPIURL,
				httpClient: http.DefaultClient,
				ogKey:      "test-genie-key",
			},
		},
	}
	for _, tt := range tests {
		tt.want.cfg = tt.cfg
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.cfg, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

type testTransport struct {
	closeRequest bool
	wantMessage  string
	wantKey      string
	wantAlias    string
	wantCloseURL string
}

func (tt *testTransport) verifyCreateRequest(req *http.Request) error {
	reqBody, _ := io.ReadAll(req.Body)
	msg := alertMessage{}

	if err := json.Unmarshal(reqBody, &msg); err != nil {
		return fmt.Errorf("error unmarshaling: %v", err)
	}

	if msg.Message != tt.wantMessage {
		return fmt.Errorf("message mismatch, want: %s, got: %s", tt.wantMessage, msg.Message)
	}

	if msg.Alias != tt.wantAlias {
		return fmt.Errorf("alias mismatch, want: %s, got: %s", tt.wantAlias, msg.Alias)
	}

	authHeader := req.Header.Get("Authorization")
	if authHeader != "GenieKey "+tt.wantKey {
		return fmt.Errorf("auth header mismatch, want: %s, got: %s", "GenieKey "+tt.wantKey, authHeader)
	}

	return nil
}

func (tt *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !tt.closeRequest {
		if err := tt.verifyCreateRequest(req); err != nil {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(bytes.NewBufferString(err.Error())),
			}, nil
		}
	}

	if tt.closeRequest {
		if req.URL.String() != tt.wantCloseURL {
			s := fmt.Sprintf("url mismatch, want: %s, got: %s", tt.wantCloseURL, req.URL.String())
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(bytes.NewBufferString(s)),
			}, nil
		}
	}

	return &http.Response{
		StatusCode: http.StatusAccepted,
		Body:       io.NopCloser(bytes.NewBufferString(tt.wantMessage)),
	}, nil
}

func TestClientNotify(t *testing.T) {
	ogKey := "default-genie-key"

	tests := []struct {
		name          string
		alertInfo     *alertinfo.AlertInfo
		alertFields   map[string]string
		wantAlias     string
		wantMessage   string
		errorContains string
	}{
		{
			name: "no-error",
			alertInfo: &alertinfo.AlertInfo{
				DeduplicationID: "test-condition",
			},
			alertFields: map[string]string{
				"summary": "test-alert-message",
			},
			wantAlias:   "test-condition",
			wantMessage: "test-alert-message",
		},
		{
			name: "alias-mismatch",
			alertInfo: &alertinfo.AlertInfo{
				DeduplicationID: "test-condition-2",
			},
			alertFields:   map[string]string{},
			wantAlias:     "test-condition-3",
			errorContains: "alias mismatch",
		},
		{
			name:      "msg-mismatch",
			alertInfo: &alertinfo.AlertInfo{},
			alertFields: map[string]string{
				"summary": "alert-message2",
			},
			wantMessage:   "alert-message",
			errorContains: "message mismatch",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				httpClient: &http.Client{Transport: &testTransport{
					wantMessage: tt.wantMessage,
					wantKey:     ogKey,
					wantAlias:   tt.wantAlias,
				}},
				ogKey: ogKey,
			}

			err := c.Notify(context.Background(), tt.alertInfo, tt.alertFields)
			if tt.errorContains != "" {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestClientNotifyResolve(t *testing.T) {
	ogKey := "default-genie-key"

	tests := []struct {
		name          string
		cfg           *configpb.Opsgenie
		alertInfo     *alertinfo.AlertInfo
		wantURL       string
		errorContains string
	}{
		{
			name: "default",
			alertInfo: &alertinfo.AlertInfo{
				DeduplicationID: "test-condition",
			},
			wantURL: "https://api.opsgenie.com/v2/alerts/test-condition/close?identifierType=alias",
		},
		{
			name: "default-mismatch",
			alertInfo: &alertinfo.AlertInfo{
				DeduplicationID: "test-condition2",
			},
			wantURL:       "https://api.opsgenie.com/v2/alerts/test-condition/close?identifierType=alias",
			errorContains: "url mismatch",
		},
		{
			name: "disabled",
			cfg: &configpb.Opsgenie{
				DisableSendResolved: true,
			},
			alertInfo: &alertinfo.AlertInfo{
				// there is mismatch but resolve is disabled
				DeduplicationID: "test-condition2",
			},
			wantURL: "https://api.opsgenie.com/v2/alerts/test-condition/close?identifierType=alias",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				cfg: tt.cfg,
				httpClient: &http.Client{Transport: &testTransport{
					closeRequest: true,
					wantCloseURL: tt.wantURL,
				}},
				apiURL: DefaultOpsgenieAPIURL,
				ogKey:  ogKey,
			}

			err := c.NotifyResolve(context.Background(), tt.alertInfo, map[string]string{})
			if tt.errorContains != "" {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}
			assert.NoError(t, err)
		})
	}
}
