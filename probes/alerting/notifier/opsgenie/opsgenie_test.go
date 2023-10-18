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
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/cloudprober/cloudprober/probes/alerting/alertinfo"
	configpb "github.com/cloudprober/cloudprober/probes/alerting/proto"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	os.Setenv(DefaultGenieKeyEnvVar, "default-genie-key")
	defer os.Unsetenv(DefaultGenieKeyEnvVar)
	os.Setenv("TEST_GENIE_KEY", "test-genie-key")
	defer os.Unsetenv("TEST_GENIE_KEY")

	tests := []struct {
		name    string
		cfg     *configpb.OpsGenie
		want    *Client
		wantErr bool
	}{
		{
			name: "default-key-url",
			cfg:  &configpb.OpsGenie{},
			want: &Client{
				apiURL:     DefaultOpsgenieAPIURL,
				httpClient: http.DefaultClient,
				ogKey:      "default-genie-key",
			},
		},
		{
			name: "config",
			cfg: &configpb.OpsGenie{
				ApiUrl:   "https://test-api.opsgenie.com/v2/alerts",
				GenieKey: "config-genie-key",
			},
			want: &Client{
				apiURL:     "https://test-api.opsgenie.com/v2/alerts",
				httpClient: http.DefaultClient,
				ogKey:      "config-genie-key",
			},
		},
		{
			name: "custom-key-env-var",
			cfg: &configpb.OpsGenie{
				GenieKeyEnvVar: "TEST_GENIE_KEY",
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
	wantMessage string
	wantKey     string
	wantAlias   string
}

func (tt *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqBody, _ := io.ReadAll(req.Body)
	msg := alertMessage{}

	err := json.Unmarshal(reqBody, &msg)
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewBufferString("error unmarshaling")),
		}, err
	}

	if msg.Message != tt.wantMessage {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewBufferString("message mismatch")),
		}, nil
	}

	if msg.Alias != tt.wantAlias {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewBufferString("alias mismatch")),
		}, nil
	}

	authHeader := req.Header.Get("Authorization")
	if authHeader != "GenieKey "+tt.wantKey {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewBufferString("auth header mismatch")),
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusAccepted,
		Body:       io.NopCloser(bytes.NewBufferString(msg.Message)),
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
				ConditionID: "test-condition",
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
				ConditionID: "test-condition-2",
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
