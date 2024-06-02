// Copyright 2018-2024 The Cloudprober Authors.
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

// Package web provides web interface for cloudprober.
package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudprober/cloudprober/config/runconfig"
	"github.com/cloudprober/cloudprober/internal/servers"
	"github.com/cloudprober/cloudprober/probes"
	"github.com/cloudprober/cloudprober/surfacers"
	"github.com/stretchr/testify/assert"
)

func TestInitWithDataFuncs(t *testing.T) {
	getInfo := func() (map[string]*probes.ProbeInfo, []*surfacers.SurfacerInfo, []*servers.ServerInfo) {

		return nil, nil, nil
	}

	tests := []struct {
		name              string
		dataFuncs         DataFuncs
		wantResp          map[string]string
		wantConfigRunning string
		wantErr           bool
	}{
		{
			name: "base",
			dataFuncs: DataFuncs{
				GetRawConfig:    func() string { return "raw-config" },
				GetParsedConfig: func() string { return "parsed-config" },
				GetInfo:         getInfo,
			},
			wantResp: map[string]string{
				"/config":        "raw-config",
				"/config-parsed": "parsed-config",
			},
		},
		{
			name: "with-secret",
			dataFuncs: DataFuncs{
				GetRawConfig:    func() string { return "raw-config" },
				GetParsedConfig: func() string { return "parsed-config **$password**" },
				GetInfo:         getInfo,
			},
			wantResp: map[string]string{
				"/config":        "raw-config",
				"/config-parsed": "parsed-config **$password**",
			},
			wantConfigRunning: secretConfigRunningMsg,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldSrvMux := runconfig.DefaultHTTPServeMux()
			defer runconfig.SetDefaultHTTPServeMux(oldSrvMux)
			srvMux := http.NewServeMux()
			runconfig.SetDefaultHTTPServeMux(srvMux)

			httpSrv := httptest.NewServer(srvMux)
			defer httpSrv.Close()

			if err := InitWithDataFuncs(tt.dataFuncs); (err != nil) != tt.wantErr {
				t.Errorf("InitWithDataFuncs() error = %v, wantErr %v", err, tt.wantErr)
			}

			client := httpSrv.Client()
			if tt.wantConfigRunning == "" {
				tt.wantConfigRunning = runningConfig(tt.dataFuncs)
			}
			tt.wantResp["/config-running"] = tt.wantConfigRunning
			for path, wantResp := range tt.wantResp {
				resp, err := client.Get(httpSrv.URL + path)
				if err != nil {
					t.Errorf("Error getting %s: %v", path, err)
					continue
				}
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Got status code: %d, want: %d", resp.StatusCode, http.StatusOK)
				}
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Errorf("Error reading response body: %v", err)
				}
				assert.Equal(t, wantResp, string(body), "response body")
			}

			httpSrv.Close()
		})
	}
}

func TestInit(t *testing.T) {
	assert.Nil(t, Init(), "Init() should return nil")
}
