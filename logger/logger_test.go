// Copyright 2017-2025 The Cloudprober Authors.
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
	"runtime"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/logging"
	"github.com/stretchr/testify/assert"
)

func TestGCPLogEntry(t *testing.T) {
	l := New(WithAttr(slog.String("dst", "gcp")))
	msg := "test message"
	tests := []struct {
		name  string
		level slog.Level
		want  logging.Entry
	}{
		{
			name:  "info",
			level: slog.LevelInfo,
			want: logging.Entry{
				Severity: logging.Info,
				Payload:  "level=INFO source=logger/logger_test.go:62 msg=\"test message\" system=cloudprober dst=gcp\n",
			},
		},
		{
			name:  "warning",
			level: slog.LevelWarn,
			want: logging.Entry{
				Severity: logging.Warning,
				Payload:  "level=WARN source=logger/logger_test.go:62 msg=\"test message\" system=cloudprober dst=gcp\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pcs [1]uintptr
			runtime.Callers(1, pcs[:])
			r := slog.NewRecord(time.Time{}, tt.level, msg, pcs[0])
			r.AddAttrs(l.attrs...)
			assert.Equal(t, tt.want, l.gcpLogEntry(&r))
		})
	}
}

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

			got := isEnvSet(varName)
			if got != row.expected {
				t.Errorf("Variable set: got=%v, expected=%v", got, row.expected)
			}
		})
	}
}

func TestWithAttributes(t *testing.T) {
	// Create a base logger with some initial attributes
	baseLogger := New(WithAttr(slog.String("base1", "value1"), slog.String("base2", "value2")))

	// Add new attributes using WithAttributes
	newAttrs := []slog.Attr{
		slog.String("new1", "newvalue1"),
		slog.String("new2", "newvalue2"),
	}
	newLogger := baseLogger.WithAttributes(newAttrs...)

	assert.Equal(t, newLogger.attrs, append(baseLogger.attrs, newAttrs...))

	// Verify other fields are properly copied
	if newLogger.minLogLevel != baseLogger.minLogLevel {
		t.Error("minLogLevel was not properly copied")
	}
	if newLogger.disableCloudLogging != baseLogger.disableCloudLogging {
		t.Error("disableCloudLogging was not properly copied")
	}
	if newLogger.gcpLoggingEndpoint != baseLogger.gcpLoggingEndpoint {
		t.Error("gcpLoggingEndpoint was not properly copied")
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
		t.Errorf("Error unmarshalling JSON (%s): %v, ", string(b), err)
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

func testVerifyTextLog(t *testing.T, lineBytes []byte, wantLabels map[string]string) {
	t.Helper()
	line := string(lineBytes)

	for k, v := range wantLabels {
		assert.Contains(t, line, k+"="+v, "label in %s", line)
	}
	sourceRegex := regexp.MustCompile("source=logger/logger_test.go:[0-9]+")
	assert.Regexp(t, sourceRegex, line, "source in log")
}

func testLog(t *testing.T, funcName string, msg string, logAttr slog.Attr, strAttrs [][2]string, nilLogger bool) []byte {
	t.Helper()

	var buf bytes.Buffer

	var attrs []slog.Attr
	for _, a := range strAttrs {
		attrs = append(attrs, slog.String(a[0], a[1]))
	}

	var l *Logger
	if !nilLogger {
		l = New(WithAttr(logAttr), WithWriter(&buf))
	} else {
		defaultWritter = &buf
		defer func() {
			defaultWritter = os.Stderr
		}()
	}

	switch funcName {
	case "Debug":
		l.Debug(msg)
	case "Debugf":
		l.Debugf("%s", msg)
	case "DebugAttrs":
		l.DebugAttrs(msg, attrs...)
	case "Infof":
		l.Infof("%s", msg)
	case "InfoAttrs":
		l.InfoAttrs(msg, attrs...)
	case "Warning":
		l.Warning(msg)
	case "Warningf":
		l.Warningf("%s", msg)
	case "WarningAttrs":
		l.WarningAttrs(msg, attrs...)
	case "Error":
		l.Error(msg)
	case "Errorf":
		l.Errorf("%s", msg)
	case "ErrorAttrs":
		l.ErrorAttrs(msg, attrs...)
	default:
		l.Info(msg)
	}

	return buf.Bytes()
}

func TestLog(t *testing.T) {
	largeLogLine := strings.Repeat("cloudprober", 10000)

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
			msg: "test-message_text",
			wantLabels: map[string]string{
				"level": "INFO",
			},
		},
		{
			msg:      "test-message_text_infof",
			funcName: "Infof",
			wantLabels: map[string]string{
				"level": "INFO",
			},
		},
		{
			msg: "test message_text_with_space",
			wantLabels: map[string]string{
				"level": "INFO",
			},
		},
		{
			msg: largeLogLine,
			wantLabels: map[string]string{
				"level": "INFO",
			},
		},
		{
			msg:        "test-message_json",
			logFmtFlag: "json",
			wantLabels: map[string]string{"level": "INFO"},
		},
		{
			msg:        "test-message_text_warning",
			funcName:   "Warning",
			wantLabels: map[string]string{"level": "WARN"},
		},
		{
			msg:        "test-message_text_warningf",
			funcName:   "Warningf",
			wantLabels: map[string]string{"level": "WARN"},
		},
		{
			msg:        "test-message_text_error",
			funcName:   "Error",
			wantLabels: map[string]string{"level": "ERROR"},
		},
		{
			msg:        "test-message_text_errorf",
			funcName:   "Errorf",
			wantLabels: map[string]string{"level": "ERROR"},
		},
		{
			msg:        "test-message_text_info_attrs",
			funcName:   "InfoAttrs",
			attrs:      [][2]string{{"attr1", "v1"}, {"attr2", "v2"}},
			wantLabels: map[string]string{"level": "INFO", "attr1": "v1", "attr2": "v2"},
		},
		{
			msg:        "test-message_text_warning_attrs",
			funcName:   "WarningAttrs",
			attrs:      [][2]string{{"attr1", "v1"}, {"attr2", "v2"}},
			wantLabels: map[string]string{"level": "WARN", "attr1": "v1", "attr2": "v2"},
		},
		{
			msg:        "test-message_text_error_attrs",
			funcName:   "ErrorAttrs",
			attrs:      [][2]string{{"attr1", "v1"}, {"attr2", "v2"}},
			wantLabels: map[string]string{"level": "ERROR", "attr1": "v1", "attr2": "v2"},
		},
		{
			msg:      "test-message_text_debug_nolog",
			funcName: "Debug",
		},
		{
			msg:         "test-message_text_debug_log",
			debugReFlag: ".*testc.*",
			funcName:    "Debug",
			wantLabels:  map[string]string{"level": "DEBUG"},
		},
		{
			msg:         "test-message_text_debug_noregexmatch",
			debugReFlag: ".*probe1.*",
			funcName:    "Debug",
		},
		{
			msg:          "test-message_text_debug",
			funcName:     "Debug",
			debugLogFlag: true,
			wantLabels:   map[string]string{"level": "DEBUG"},
		},
		{
			msg:          "test-message_text_debugf",
			funcName:     "Debugf",
			debugLogFlag: true,
			wantLabels:   map[string]string{"level": "DEBUG"},
		},
		{
			msg:          "test-message_text_debug_attrs",
			funcName:     "DebugAttrs",
			debugLogFlag: true,
			attrs:        [][2]string{{"attr1", "v1"}, {"attr2", "v2"}},
			wantLabels:   map[string]string{"level": "DEBUG", "attr1": "v1", "attr2": "v2"},
		},
	}

	for _, tt := range tests {
		for _, nilLogger := range []bool{false, true} {
			name := tt.msg
			if tt.msg == largeLogLine {
				name = "large-log-line"
			}
			if nilLogger {
				name += "-nilLogger"
			}
			t.Run(name, func(t *testing.T) {
				wantLabels := make(map[string]string)
				for k, v := range tt.wantLabels {
					wantLabels[k] = v
				}

				if tt.logFmtFlag == "" {
					tt.logFmtFlag = "text"
				}

				defer func() {
					*logFmt = "text"
					*debugLog = false
					*debugLogList = ""
				}()
				*logFmt = tt.logFmtFlag
				*debugLog = tt.debugLogFlag
				*debugLogList = tt.debugReFlag

				b := testLog(t, tt.funcName, tt.msg, slog.String("component", "testc"), tt.attrs, nilLogger)

				if len(tt.wantLabels) == 0 || (nilLogger && tt.debugReFlag == ".*testc.*") {
					assert.Equal(t, "", string(b))
					return
				}
				if !nilLogger {
					wantLabels["component"] = "testc"
					wantLabels["system"] = "cloudprober"
				}

				wantLabels["msg"] = tt.msg
				if tt.msg == largeLogLine {
					s := "... (truncated)"
					wantLabels["msg"] = tt.msg[:MaxLogEntrySize-len(s)] + s
				}

				if tt.logFmtFlag == "json" {
					testVerifyJSONLog(t, b, wantLabels)
				} else {
					// Logger adds quotes to the message if it contains spaces.
					if strings.Contains(wantLabels["msg"], " ") {
						wantLabels["msg"] = "\"" + wantLabels["msg"] + "\""
					}
					testVerifyTextLog(t, b, wantLabels)
				}
			})
		}
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

func TestSkipLog(t *testing.T) {
	tests := []struct {
		minLogLevelFlag string
		levels          []slog.Level
		want            []bool
	}{
		{
			minLogLevelFlag: "INFO",
			levels:          []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn},
			want:            []bool{true, false, false},
		},
		{
			minLogLevelFlag: "WARNING",
			levels:          []slog.Level{slog.LevelInfo, slog.LevelWarn, slog.LevelError},
			want:            []bool{true, false, false},
		},
		{
			minLogLevelFlag: "ERROR",
			levels:          []slog.Level{slog.LevelWarn, slog.LevelError, criticalLevel},
			want:            []bool{true, false, false},
		},
		{
			minLogLevelFlag: "CRITICAL",
			levels:          []slog.Level{slog.LevelError, criticalLevel},
			want:            []bool{true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.minLogLevelFlag, func(t *testing.T) {
			defer func() {
				*minLogLevel = "INFO"
			}()
			*minLogLevel = tt.minLogLevelFlag

			for i, level := range tt.levels {
				t.Run(level.String(), func(t *testing.T) {
					var l *Logger
					if got := l.skipLog(level); got != tt.want[i] {
						t.Errorf("nil Logger.skipLog() = %v, want %v", got, tt.want)
					}
					l = newLogger()
					if got := l.skipLog(level); got != tt.want[i] {
						t.Errorf("non-nil Logger.skipLog() = %v, want %v", got, tt.want)
					}
				})
			}
		})
	}
}
