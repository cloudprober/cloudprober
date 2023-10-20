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
	"io"
	"testing"

	"github.com/cloudprober/cloudprober/internal/alerting/alertinfo"
	configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
	"github.com/stretchr/testify/assert"
)

func TestAlertRequest(t *testing.T) {
	alertInfo := &alertinfo.AlertInfo{
		DeduplicationID: "test-condition",
	}

	testKey := "test-genie-key"
	wantReqHeaders := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "GenieKey " + testKey,
	}

	tests := []struct {
		name        string
		cfg         *configpb.Opsgenie
		apiURL      string
		alertInfo   *alertinfo.AlertInfo
		alertFields map[string]string

		wantAlertMsg *alertMessage
		wantReqBody  string
		wantReqURL   string
	}{
		{
			name: "defaultSeverity",
			alertFields: map[string]string{
				"name":    "test-alert",
				"summary": `Cloudprober alert "test-alert" for "test-target"`,
			},
			wantAlertMsg: &alertMessage{
				Alias:    "test-condition",
				Entity:   "Cloudprober",
				Message:  `Cloudprober alert "test-alert" for "test-target"`,
				Priority: P2,
				Details:  map[string]string{"name": "test-alert"},
			},
			wantReqBody: `{"alias":"test-condition","entity":"Cloudprober","message":"Cloudprober alert \"test-alert\" for \"test-target\"","priority":"P2","details":{"name":"test-alert"}}`,
		},
		{
			name: "criticalSeverity",
			alertFields: map[string]string{
				"severity": "CRITICAL",
				"summary":  `Cloudprober alert "test-alert" for "test-target"`,
			},
			wantAlertMsg: &alertMessage{
				Alias:    "test-condition",
				Entity:   "Cloudprober",
				Message:  `Cloudprober alert "test-alert" for "test-target"`,
				Priority: P1,
				Details:  map[string]string{},
			},
			wantReqBody: `{"alias":"test-condition","entity":"Cloudprober","message":"Cloudprober alert \"test-alert\" for \"test-target\"","priority":"P1"}`,
		},
		{
			name: "withResponders",
			cfg: &configpb.Opsgenie{
				Responders: []*configpb.Opsgenie_Responder{
					{
						Type: configpb.Opsgenie_Responder_TEAM,
						Ref:  &configpb.Opsgenie_Responder_Name{Name: "test-team"},
					},
					{
						Type: configpb.Opsgenie_Responder_SCHEDULE,
						Ref:  &configpb.Opsgenie_Responder_Id{Id: "test-schedule-id"},
					},
				},
			},
			apiURL: "https://test-api.opsgenie.com/v2/alerts",
			alertFields: map[string]string{
				"summary": `Cloudprober alert "test-alert" for "test-target"`,
			},
			wantAlertMsg: &alertMessage{
				Alias:    "test-condition",
				Entity:   "Cloudprober",
				Message:  `Cloudprober alert "test-alert" for "test-target"`,
				Priority: P2,
				Details:  map[string]string{},
				Responders: []responder{
					{
						Type: "team",
						Name: "test-team",
					},
					{
						Type: "schedule",
						ID:   "test-schedule-id",
					},
				},
			},
			wantReqURL:  "https://test-api.opsgenie.com/v2/alerts",
			wantReqBody: `{"alias":"test-condition","entity":"Cloudprober","message":"Cloudprober alert \"test-alert\" for \"test-target\"","priority":"P2","responders":[{"type":"team","name":"test-team"},{"type":"schedule","id":"test-schedule-id"}]}`,
		},
	}
	for _, tt := range tests {
		if tt.apiURL == "" {
			tt.apiURL = "https://api.opsgenie.com/v2/alerts"
			tt.wantReqURL = "https://api.opsgenie.com/v2/alerts"
		}
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				cfg:    tt.cfg,
				apiURL: tt.apiURL,
				ogKey:  testKey,
			}
			msg := c.createAlertMessage(alertInfo, tt.alertFields)
			assert.Equal(t, tt.wantAlertMsg, msg, "request mismatch")

			req, err := c.alertRequest(msg)
			assert.NoError(t, err, "error creating request")

			assert.Equal(t, tt.wantReqURL, req.URL.String(), "request url mismatch")
			headers := make(map[string]string)
			for k, v := range req.Header {
				headers[k] = v[0]
			}
			assert.Equal(t, wantReqHeaders, headers, "request headers mismatch")

			reqBody, err := io.ReadAll(req.Body)
			assert.NoError(t, err, "error reading request body")
			assert.Equal(t, tt.wantReqBody, string(reqBody), "request body mismatch")
		})
	}
}

func TestClientCloseRequest(t *testing.T) {
	c := &Client{
		apiURL: "https://api.opsgenie.com/v2/alerts",
		ogKey:  "test-genie-key",
	}
	alias := "test-condition"
	wantReqURL := "https://api.opsgenie.com/v2/alerts/test-condition/close?identifierType=alias"
	wantReqBody := "{}"

	req, err := c.closeRequest(alias)
	assert.NoError(t, err, "error creating request")
	assert.Equal(t, wantReqURL, req.URL.String(), "request url mismatch")
	assert.Equal(t, "POST", req.Method, "request method mismatch")
	assert.Equal(t, "GenieKey "+c.ogKey, req.Header.Get("Authorization"), "request header mismatch")

	gotBody, _ := io.ReadAll(req.Body)
	assert.Equal(t, wantReqBody, string(gotBody), "request body mismatch")

}
