// Copyright 2017-2023 The Cloudprober Authors.
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

package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvVarSet(t *testing.T) {
	varName := "TEST_VAR"

	testRows := []struct {
		v        string
		expected bool
	}{
		{"1", true},
		{"yes", true},
		{"not_set", false},
		{"no", false},
		{"false", false},
	}

	for _, row := range testRows {
		t.Run(fmt.Sprintf("Val: %s, should be set: %v", row.v, row.expected), func(t *testing.T) {
			os.Unsetenv(varName)
			if row.v != "not_set" {
				os.Setenv(varName, row.v)
			}

			got := envVarSet(varName)
			if got != row.expected {
				t.Errorf("Variable set: got=%v, expected=%v", got, row.expected)
			}
		})
	}
}

func TestWithAttr(t *testing.T) {
	tests := []struct {
		name      string
		l         *Logger
		wantAttrs []slog.Attr
	}{
		{
			name:      "new-withAttrs",
			l:         New(WithAttr(slog.String("probe", "testprobe"))),
			wantAttrs: []slog.Attr{slog.String("system", "cloudprober"), slog.String("probe", "testprobe")},
		},
		{
			name:      "new-withAttrs-different-system",
			l:         New(WithAttr(slog.String("probe", "testprobe"), slog.String("system", "testsystem"))),
			wantAttrs: []slog.Attr{slog.String("system", "testsystem"), slog.String("probe", "testprobe")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantAttrs, tt.l.attrs)
		})
	}

}

func testVerifyJSONLog(t *testing.T, b []byte, wantLabels map[string]string) {
	t.Helper()

	// Verify it's a proper JSON.
	gotMap := make(map[string]interface{})
	err := json.Unmarshal(b, &gotMap)
	if err != nil {
		t.Errorf("Error unmarshalling JSON: %v", err)
		return
	}

	for k, v := range wantLabels {
		assert.Equal(t, v, gotMap[k], "label %s in %s", k+"="+v, string(b))
	}

	// Verify json source
	gotSource := gotMap["source"].(map[string]interface{})
	assert.Equal(t, "logger.testLog", gotSource["function"].(string), "json source - function")
	assert.Equal(t, "logger/logger_test.go", gotSource["file"], "json source - file")
	assert.GreaterOrEqual(t, gotSource["line"].(float64), float64(100), "json source - line")

}

func testVerifyTextLog(t *testing.T, line string, wantLabels map[string]string) {
	t.Helper()

	for k, v := range wantLabels {
		assert.Contains(t, line, k+"="+v, "label in %s", line)
	}
	sourceRegex := regexp.MustCompile("source=logger/logger_test.go:[0-9]+")
	assert.Regexp(t, sourceRegex, line, "source in log")
}

func testLog(t *testing.T, funcName string, msg string, logAttr slog.Attr, strAttrs [][2]string) {
	t.Helper()

	var attrs []slog.Attr
	for _, a := range strAttrs {
		attrs = append(attrs, slog.String(a[0], a[1]))
	}

	l := NewWithAttrs(logAttr)

	switch funcName {
	case "Debug":
		l.Debug(msg)
	case "DebugAttrs":
		l.DebugAttrs(msg, attrs...)
	case "InfoAttrs":
		l.InfoAttrs(msg, attrs...)
	case "Warning":
		l.Warning(msg)
	case "WarningAttrs":
		l.WarningAttrs(msg, attrs...)
	case "Error":
		l.Error(msg)
	case "ErrorAttrs":
		l.ErrorAttrs(msg, attrs...)
	default:
		l.Info(msg)
	}
}

func TestLog(t *testing.T) {
	tests := []struct {
		msg          string
		funcName     string
		logFmtFlag   string
		debugLogFlag bool
		debugReFlag  string
		attrs        [][2]string
		wantLabels   map[string]string
	}{
		{
			msg: "test message - text",
			wantLabels: map[string]string{
				"level": "INFO",
			},
		},
		{
			msg:        "test message - json",
			logFmtFlag: "json",
			wantLabels: map[string]string{"level": "INFO"},
		},
		{
			msg:        "test message - text - warning",
			funcName:   "Warning",
			wantLabels: map[string]string{"level": "WARN"},
		},
		{
			msg:        "test message - text - error",
			funcName:   "Error",
			wantLabels: map[string]string{"level": "ERROR"},
		},
		{
			msg:        "test message - text - info - attrs",
			funcName:   "InfoAttrs",
			attrs:      [][2]string{{"attr1", "v1"}, {"attr2", "v2"}},
			wantLabels: map[string]string{"level": "INFO", "attr1": "v1", "attr2": "v2"},
		},
		{
			msg:        "test message - text - warning - attrs",
			funcName:   "WarningAttrs",
			attrs:      [][2]string{{"attr1", "v1"}, {"attr2", "v2"}},
			wantLabels: map[string]string{"level": "WARN", "attr1": "v1", "attr2": "v2"},
		},
		{
			msg:        "test message - text - error - attrs",
			funcName:   "ErrorAttrs",
			attrs:      [][2]string{{"attr1", "v1"}, {"attr2", "v2"}},
			wantLabels: map[string]string{"level": "ERROR", "attr1": "v1", "attr2": "v2"},
		},
		{
			msg:      "test message - text - debug - nolog",
			funcName: "Debug",
		},
		{
			msg:         "test message - text - debug - log",
			debugReFlag: ".*testc.*",
			funcName:    "Debug",
			wantLabels:  map[string]string{"level": "DEBUG"},
		},
		{
			msg:         "test message - text - debug - log",
			debugReFlag: ".*probe1.*",
			funcName:    "Debug",
		},
		{
			msg:          "test message - text - debug",
			funcName:     "Debug",
			debugLogFlag: true,
			wantLabels:   map[string]string{"level": "DEBUG"},
		},
		{
			msg:          "test message - text - debug - attrs",
			funcName:     "DebugAttrs",
			debugLogFlag: true,
			attrs:        [][2]string{{"attr1", "v1"}, {"attr2", "v2"}},
			wantLabels:   map[string]string{"level": "DEBUG", "attr1": "v1", "attr2": "v2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			var b bytes.Buffer
			defaultWritter = &b
			defer func() {
				defaultWritter = os.Stderr
			}()

			if tt.logFmtFlag == "" {
				tt.logFmtFlag = "text"
			}
			*logFmt = tt.logFmtFlag
			*debugLog = tt.debugLogFlag
			*debugLogList = tt.debugReFlag

			testLog(t, tt.funcName, tt.msg, slog.String("component", "testc"), tt.attrs)

			if len(tt.wantLabels) == 0 {
				assert.Equal(t, "", b.String())
				return
			}
			tt.wantLabels["component"] = "testc"
			tt.wantLabels["system"] = "cloudprober"

			if tt.logFmtFlag == "json" {
				tt.wantLabels["msg"] = tt.msg
				testVerifyJSONLog(t, b.Bytes(), tt.wantLabels)
			} else {
				tt.wantLabels["msg"] = "\"" + tt.msg + "\""
				testVerifyTextLog(t, b.String(), tt.wantLabels)
			}
		})
	}
}

func TestSDLogName(t *testing.T) {
	tests := []struct {
		name    string
		attrs   []slog.Attr
		want    string
		wantErr string
	}{
		{
			name: "no attrs",
			want: "cloudprober",
		},
		{
			name:  "probe",
			attrs: []slog.Attr{slog.String("probe", "testprobe")},
			want:  "cloudprober.testprobe",
		},
		{
			name:  "component",
			attrs: []slog.Attr{slog.String("component", "rds-server")},
			want:  "cloudprober.rds-server",
		},
		{
			name:  "surfacer_cloudwatch",
			attrs: []slog.Attr{slog.String("surfacer", "cloudwatch")},
			want:  "cloudprober.cloudwatch",
		},
		{
			name:  "different_system",
			attrs: []slog.Attr{slog.String("system", "protodoc"), slog.String("surfacer", "cloudwatch")},
			want:  "protodoc.cloudwatch",
		},
		{
			name:    "invalid char",
			attrs:   []slog.Attr{slog.String("surfacer", "cloudwatch*")},
			wantErr: "invalid character",
		},
		{
			name:  "url escape",
			attrs: []slog.Attr{slog.String("surfacer", "cloudwatch/v1")},
			want:  "cloudprober.cloudwatch%2Fv1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewWithAttrs(tt.attrs...)
			got, err := l.sdLogName()
			if err != nil {
				if tt.wantErr == "" {
					t.Errorf("Logger.sdLogName() unexpected error: %v", err)
					return
				}
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			if tt.wantErr != "" {
				t.Errorf("Logger.sdLogName() expected error: %v", tt.wantErr)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
