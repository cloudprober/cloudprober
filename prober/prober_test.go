// Copyright 2023 The Cloudprober Authors.
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

package prober

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/config/proto"
	"github.com/cloudprober/cloudprober/probes"
	probespb "github.com/cloudprober/cloudprober/probes/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func TestRandomDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		ceiling  time.Duration
	}{
		{
			duration: 0,
			ceiling:  10 * time.Second,
		},
		{
			duration: 5 * time.Second,
			ceiling:  10 * time.Second,
		},
		{
			duration: 30 * time.Second,
			ceiling:  10 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt), func(t *testing.T) {
			got := randomDuration(tt.duration, tt.ceiling)
			assert.LessOrEqual(t, got, tt.duration)
			assert.LessOrEqual(t, got, tt.ceiling)
		})
	}
}

func TestInterProbeWait(t *testing.T) {
	tests := []struct {
		interval  time.Duration
		numProbes int
		want      time.Duration
	}{
		{
			interval:  2 * time.Second,
			numProbes: 16,
			want:      125 * time.Millisecond,
		},
		{
			interval:  30 * time.Second,
			numProbes: 12,
			want:      2 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s:%d", tt.interval, tt.numProbes), func(t *testing.T) {
			assert.Equal(t, tt.want, interProbeWait(tt.interval, tt.numProbes))
		})
	}
}

func TestSaveProbesConfigUnprotected(t *testing.T) {
	probes := map[string]*probes.ProbeInfo{
		"test-probe-1": {
			ProbeDef: &probespb.ProbeDef{
				Name: proto.String("test-probe-1"),
				Type: probespb.ProbeDef_DNS.Enum(),
			},
		},
		"test-probe-2": {
			ProbeDef: &probespb.ProbeDef{
				Name: proto.String("test-probe-2"),
				Type: probespb.ProbeDef_PING.Enum(),
			},
		},
	}

	tempDir := os.TempDir()

	tests := []struct {
		name       string
		filePath   string
		wantConfig string
		wantErr    bool
	}{
		{
			name:    "empty_file",
			wantErr: true,
		},
		{
			name:     "dir_path",
			filePath: tempDir,
			wantErr:  true,
		},
		{
			name:     "normal",
			filePath: filepath.Join(tempDir, "cp_probes.cfg"),
			wantConfig: `probe: {
  name: "test-probe-1"
  type: DNS
}
probe: {
  name: "test-probe-2"
  type: PING
}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &Prober{
				Probes: probes,
			}
			if err := pr.saveProbesConfigUnprotected(tt.filePath); (err != nil) != tt.wantErr {
				t.Errorf("Prober.saveProbesConfigUnprotected() error = %v, wantErr %v", err, tt.wantErr)
			}

			got, _ := os.ReadFile(tt.filePath)

			var g configpb.ProberConfig
			var w configpb.ProberConfig
			assert.NoError(t, prototext.Unmarshal(got, &g))
			assert.NoError(t, prototext.Unmarshal([]byte(tt.wantConfig), &w))
			assert.Equal(t, w.String(), g.String())
		})
	}
}
