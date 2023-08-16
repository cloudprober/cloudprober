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

// Package logger provides a logger that logs to Google Cloud Logging. It's a thin wrapper around
// golang/cloud/logging package.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"flag"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/logging"
	md "github.com/cloudprober/cloudprober/common/metadata"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

var (
	logFmt = flag.String("logfmt", "text", "Log format. Valid values: text, json")
	_      = flag.Bool("logtostderr", true, "(deprecated) this option doesn't do anything anymore. All logs to stderr by default.")

	debugLog     = flag.Bool("debug_log", false, "Whether to output debug logs or not")
	debugLogList = flag.String("debug_logname_regex", "", "Enable debug logs for only for log names that match this regex (e.g. --debug_logname_regex=.*probe1.*")

	// Enable/Disable cloud logging
	disableCloudLogging = flag.Bool("disable_cloud_logging", false, "Disable cloud logging.")

	// LogPrefixEnvVar environment variable is used to determine the stackdriver
	// log name prefix. Default prefix is "cloudprober".
	LogPrefixEnvVar = "CLOUDPROBER_LOG_PREFIX"
)

// EnvVars defines environment variables that can be used to modify the logging
// behavior.
var EnvVars = struct {
	DisableCloudLogging, DebugLog string
}{
	"CLOUDPROBER_DISABLE_CLOUD_LOGGING",
	"CLOUDPROBER_DEBUG_LOG",
}

const (
	// Prefix for the cloudprober stackdriver log names.
	cloudproberPrefix = "cloudprober"
)

const (
	// Regular Expression for all characters that are illegal for log names
	//	Ref: https://cloud.google.com/logging/docs/api/ref_v2beta1/rest/v2beta1/LogEntry
	disapprovedRegExp = "[^A-Za-z0-9_/.-]"

	// MaxLogEntrySize Max size of each log entry (100 KB)
	// This limit helps avoid creating very large log lines in case someone
	// accidentally creates a large EventMetric, which in turn is possible due to
	// unbounded nature of "map" metric where keys are created on demand.
	//
	// TODO(manugarg): We can possibly get rid of this check now as the code that
	// could cause a large map metric has been fixed now. Earlier, cloudprober's
	// HTTP server used to respond to all URLs and used to record access to those
	// URLs as a "map" metric. Now, it responds only to pre-configured URLs.
	MaxLogEntrySize = 102400
)

var defaultWritter = io.Writer(os.Stderr)

func slogHandler() slog.Handler {
	if *logFmt != "json" && *logFmt != "text" {
		slog.Default().Error("invalid log format: " + *logFmt)
		os.Exit(1)
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
	}

	attrs := []slog.Attr{slog.String("system", "cloudprober")}
	if *logFmt == "json" {
		return slog.NewJSONHandler(defaultWritter, opts).WithAttrs(attrs)
	}
	return slog.NewTextHandler(defaultWritter, opts).WithAttrs(attrs)
}

func (l *Logger) enableDebugLog(debugLog bool, debugLogRe string) bool {
	if !debugLog && debugLogRe == "" {
		return false
	}

	if debugLog && debugLogRe == "" {
		// Enable for all logs, regardless of log names.
		return true
	}

	r, err := regexp.Compile(debugLogRe)
	if err != nil {
		panic(fmt.Sprintf("error while parsing log name regex (%s): %v", debugLogRe, err))
	}

	for _, attr := range l.attrs {
		if r.MatchString(attr.Key + "=" + attr.Value.String()) {
			return true
		}
	}

	return false
}

// Logger implements a logger that logs messages to Google Cloud Logging. It
// provides a suite of methods where each method corresponds to a specific
// logging.Level, e.g. Error(paylod interface{}). Each method takes a payload
// that has to either be a JSON-encodable object, a string or a []byte slice
// (all other types of payload will result in error).
//
// It falls back to logging through the traditional logger if:
//
//   - Not running on GCE,
//   - Logging client is uninitialized (e.g. for testing),
//   - Logging to cloud fails for some reason.
//
// Logger{} is a valid object that will log through the traditional logger.
//
// labels is a map which is present in every log entry and all custom
// metadata about the log entries can be inserted into this map.
// For example probe id can be inserted into this map.
type Logger struct {
	slogger             *slog.Logger
	gcpLogc             *logging.Client
	gcpLogger           *logging.Logger
	debugLog            bool
	disableCloudLogging bool
	labels              map[string]string
	attrs               []slog.Attr
	// TODO(manugarg): Logger should eventually embed the probe id and each probe
	// should get a different Logger object (embedding that probe's probe id) but
	// sharing the same logging client. We could then make probe id one of the
	// metadata on all logging messages.
}

// Option can be used for adding additional metadata information in logger.
type Option func(*Logger)

// NewWithAttrs returns a new cloudprober Logger.
// This logger logs to stderr by default. If running on GCE, it also logs to
// Google Cloud Logging.
func NewWithAttrs(attrs ...slog.Attr) *Logger {
	return newLogger(WithAttr(attrs...))
}

// New returns a new cloudprober Logger.
// This logger logs to stderr by default. If running on GCE, it also logs to
// Google Cloud Logging.
// Deprecated: Use NewWithAttrs instead.
func New(ctx context.Context, logName string, opts ...Option) (*Logger, error) {
	return newLogger(append(opts, WithAttr(slog.String("name", logName)))...), nil
}

func newLogger(opts ...Option) *Logger {
	l := &Logger{
		labels:              make(map[string]string),
		disableCloudLogging: *disableCloudLogging,
	}
	for _, opt := range opts {
		opt(l)
	}

	// Initialize the traditional logger.
	l.slogger = slog.New(slogHandler().WithAttrs(l.attrs))
	for k, v := range l.labels {
		l.slogger = l.slogger.With(k, v)
	}

	l.debugLog = l.enableDebugLog(*debugLog, *debugLogList)

	if metadata.OnGCE() && !l.disableCloudLogging {
		l.EnableStackdriverLogging()
	}
	return l
}

// WithAttr option can be used to add a set of labels to all logs, e.g.
// logger.New(ctx, logName, logger.WithAttr(myLabels))
func WithAttr(attrs ...slog.Attr) Option {
	return func(l *Logger) {
		l.attrs = append(l.attrs, attrs...)
	}
}

// WithLabels option can be used to add a set of labels to all logs, e.g.
// logger.New(ctx, logName, logger.WithLabels(myLabels))
func WithLabels(labels map[string]string) Option {
	return func(l *Logger) {
		if l.labels == nil {
			l.labels = make(map[string]string)
		}
		for k, v := range labels {
			l.labels[k] = v
		}
	}
}

func verifySDLogName(logName string) (string, error) {
	// Check for illegal characters in the log name
	if match, err := regexp.Match(disapprovedRegExp, []byte(logName)); err != nil || match {
		if err != nil {
			return "", fmt.Errorf("unable to parse logName (%s): %v", logName, err)
		}
		return "", fmt.Errorf("logName (%s) contains an invalid character, valid characters are [A-Za-z0-9_/.-]", logName)
	}

	// Any forward slashes need to be URL encoded, so we query escape to replace them
	return url.QueryEscape(logName), nil
}

func (l *Logger) sdLogName() (string, error) {
	prefix := cloudproberPrefix
	envLogPrefix := os.Getenv(LogPrefixEnvVar)
	if os.Getenv(LogPrefixEnvVar) != "" {
		prefix = envLogPrefix
	}

	for _, attr := range l.attrs {
		if attr.Key == "name" {
			return verifySDLogName(attr.Value.String())
		}

		if slices.Contains([]string{"component", "probe", "surfacer"}, attr.Key) {
			return verifySDLogName(prefix + "." + attr.Value.String())
		}
	}

	return verifySDLogName(prefix)
}

// EnableStackdriverLogging enables logging to stackdriver.
func (l *Logger) EnableStackdriverLogging() {
	logName, err := l.sdLogName()
	if err != nil {
		l.Warningf("Error getting log name for google cloud logging: %v, will skip", err)
		return
	}

	projectID, err := metadata.ProjectID()
	if err != nil {
		l.Warningf("Error getting project id for google cloud logging: %v, will skip", err)
		return
	}

	l.gcpLogc, err = logging.NewClient(context.Background(), projectID, option.WithTokenSource(google.ComputeTokenSource("")))
	if err != nil {
		l.Warningf("Error creating client for google cloud logging: %v, will skip", err)
		return
	}

	// Add instance_name to common labels if available.
	if !md.IsKubernetes() && !md.IsCloudRunJob() && !md.IsCloudRunService() {
		instanceName, err := metadata.InstanceName()
		if err != nil {
			l.Infof("Error getting instance name on GCE: %v", err)
		} else {
			l.labels["instance_name"] = instanceName
		}
	}

	loggerOpts := []logging.LoggerOption{
		// Encourage batching of write requests.
		// Flush logs to remote logging after 1000 entries (default is 10).
		logging.EntryCountThreshold(1000),
		// Maximum amount of time that an item should remain buffered in memory
		// before being flushed to the logging service. Default is 1 second.
		// We want flushing to be mostly driven by the buffer size (configured
		// above), rather than time.
		logging.DelayThreshold(10 * time.Second),
		// Common labels that will be present in all log entries.
		logging.CommonLabels(l.labels),
	}

	l.gcpLogger = l.gcpLogc.Logger(logName, loggerOpts...)
}

// logWithAttrs logs the message to stderr with the given attributes. If
// running on GCE, logs are also sent to GCE or cloud logging.
func (l *Logger) log(severity logging.Severity, depth int, payload ...string) {
	depth++

	if len(payload) == 1 {
		l.logAttrs(severity, depth, payload[0])
		return
	}

	var b strings.Builder
	for _, s := range payload {
		b.WriteString(s)
	}

	l.logAttrs(severity, depth, b.String())
}

// logAttrs logs the message to stderr with the given attributes. If
// running on GCE, logs are also sent to GCE or cloud logging.
func (l *Logger) logAttrs(severity logging.Severity, depth int, msg string, attrs ...slog.Attr) {
	depth++

	if len(msg) > MaxLogEntrySize {
		truncateMsg := "... (truncated)"
		truncateMsgLen := len(truncateMsg)
		msg = msg[:MaxLogEntrySize-truncateMsgLen] + truncateMsg
	}

	l.genericLog(severity, depth, msg, attrs...)

	if l != nil && l.gcpLogger != nil {
		l.gcpLogger.Log(logging.Entry{
			Severity: severity,
			Payload:  msg,
		})
	}

	if severity == logging.Critical {
		l.Close()
		os.Exit(1)
	}
}

func (l *Logger) genericLog(severity logging.Severity, depth int, s string, attrs ...slog.Attr) {
	depth++
	var pcs [1]uintptr
	runtime.Callers(depth, pcs[:])

	var r slog.Record

	switch severity {
	case logging.Debug:
		r = slog.NewRecord(time.Now(), slog.LevelDebug, s, pcs[0])
	case logging.Info:
		r = slog.NewRecord(time.Now(), slog.LevelInfo, s, pcs[0])
	case logging.Warning:
		r = slog.NewRecord(time.Now(), slog.LevelWarn, s, pcs[0])
	case logging.Error:
		r = slog.NewRecord(time.Now(), slog.LevelError, s, pcs[0])
	case logging.Critical:
		r = slog.NewRecord(time.Now(), slog.Level(12), s, pcs[0])
	}

	r.AddAttrs(attrs...)

	if l != nil && l.slogger != nil {
		_ = l.slogger.Handler().Handle(context.Background(), r)
	} else {
		_ = slogHandler().Handle(context.Background(), r)
	}
}

// Close closes the cloud logging client if it exists. This flushes the buffer
// and should be called before exiting the program to ensure all logs are persisted.
func (l *Logger) Close() error {
	if l != nil && l.gcpLogc != nil {
		return l.gcpLogc.Close()
	}

	return nil
}

// Debug logs messages with logging level set to "Debug".
func (l *Logger) Debug(payload ...string) {
	if l != nil && l.debugLog {
		l.log(logging.Debug, 2, payload...)
	}
}

// Debug logs messages with logging level set to "Debug".
func (l *Logger) DebugAttrs(msg string, attrs ...slog.Attr) {
	if l != nil && l.debugLog {
		l.logAttrs(logging.Debug, 2, msg, attrs...)
	}
}

// Info logs messages with logging level set to "Info".
func (l *Logger) Info(payload ...string) {
	l.log(logging.Info, 2, payload...)
}

// InfoWithAttrs logs messages with logging level set to "Info".
func (l *Logger) InfoAttrs(msg string, attrs ...slog.Attr) {
	l.logAttrs(logging.Info, 2, msg, attrs...)
}

// Warning logs messages with logging level set to "Warning".
func (l *Logger) Warning(payload ...string) {
	l.log(logging.Warning, 2, payload...)
}

// WarningAttrs logs messages with logging level set to "Warning".
func (l *Logger) WarningAttrs(msg string, attrs ...slog.Attr) {
	l.logAttrs(logging.Warning, 2, msg, attrs...)
}

// Error logs messages with logging level set to "Error".
func (l *Logger) Error(payload ...string) {
	l.log(logging.Error, 2, payload...)
}

// ErrorAttrs logs messages with logging level set to "Warning".
func (l *Logger) ErrorAttrs(msg string, attrs ...slog.Attr) {
	l.logAttrs(logging.Error, 2, msg, attrs...)
}

// Critical logs messages with logging level set to "Critical" and
// exits the process with error status. The buffer is flushed before exiting.
func (l *Logger) Critical(payload ...string) {
	l.log(logging.Critical, 2, payload...)
}

// Critical logs messages with logging level set to "Critical" and
// exits the process with error status. The buffer is flushed before exiting.
func (l *Logger) CriticalAttrs(msg string, attrs ...slog.Attr) {
	l.logAttrs(logging.Critical, 2, msg, attrs...)
}

// Debugf logs formatted text messages with logging level "Debug".
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l != nil && l.debugLog {
		l.log(logging.Debug, 2, fmt.Sprintf(format, args...))
	}
}

// Infof logs formatted text messages with logging level "Info".
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(logging.Info, 2, fmt.Sprintf(format, args...))
}

// Warningf logs formatted text messages with logging level "Warning".
func (l *Logger) Warningf(format string, args ...interface{}) {
	l.log(logging.Warning, 2, fmt.Sprintf(format, args...))
}

// Errorf logs formatted text messages with logging level "Error".
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(logging.Error, 2, fmt.Sprintf(format, args...))
}

// Criticalf logs formatted text messages with logging level "Critical" and
// exits the process with error status. The buffer is flushed before exiting.
func (l *Logger) Criticalf(format string, args ...interface{}) {
	l.log(logging.Critical, 2, fmt.Sprintf(format, args...))
}

func envVarSet(key string) bool {
	v, ok := os.LookupEnv(key)
	if ok && strings.ToUpper(v) != "NO" && strings.ToUpper(v) != "FALSE" {
		return true
	}
	return false
}

func init() {
	if envVarSet(EnvVars.DisableCloudLogging) {
		*disableCloudLogging = true
	}

	if envVarSet(EnvVars.DebugLog) {
		*debugLog = true
	}
}
