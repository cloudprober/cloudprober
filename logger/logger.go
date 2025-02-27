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

// Package logger provides a logger that logs to Google Cloud Logging. It's a thin wrapper around
// golang/cloud/logging package.
package logger

import (
	"bytes"
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

	minLogLevel  = flag.String("min_log_level", "INFO", "Minimum log level to log. Valid values: DEBUG, INFO, WARNING, ERROR, CRITICAL")
	debugLog     = flag.Bool("debug_log", false, "Whether to output debug logs or not. Deprecated: use --min_log_level=DEBUG instead.")
	debugLogList = flag.String("debug_logname_regex", "", "Enable debug logs for only for log names that match this regex (e.g. --debug_logname_regex=.*probe1.*")

	// Enable/Disable cloud logging
	disableCloudLogging = flag.Bool("disable_cloud_logging", false, "Disable cloud logging.")

	// Override the GCP cloud logging endpoint.
	gcpLoggingEndpoint = flag.String("gcp_logging_endpoint", "", "GCP logging endpoint")

	// LogPrefixEnvVar environment variable is used to determine the stackdriver
	// log name prefix. Default prefix is "cloudprober".
	LogPrefixEnvVar = "CLOUDPROBER_LOG_PREFIX"
)

// EnvVars defines environment variables that can be used to modify the logging
// behavior.
var EnvVars = struct {
	DisableCloudLogging, DebugLog, GCPLoggingEndpoint string
}{
	"CLOUDPROBER_DISABLE_CLOUD_LOGGING",
	"CLOUDPROBER_DEBUG_LOG",
	"CLOUDPROBER_GCP_LOGGING_ENDPOINT",
}

func isEnvSet(key string) bool {
	v, ok := os.LookupEnv(key)
	if ok && strings.ToUpper(v) != "NO" && strings.ToUpper(v) != "FALSE" && v != "" {
		return true
	}
	return false
}

const (
	// Default "system" label and stackdriver log name prefix.
	defaultSystemName = "cloudprober"

	criticalLevel = slog.Level(12)
)

const (
	// Regular Expression for all characters that are illegal for log names
	//	Ref: https://cloud.google.com/logging/docs/api/ref_v2beta1/rest/v2beta1/LogEntry
	disapprovedRegExp = "[^A-Za-z0-9_/.-]"

	// MaxLogEntrySize Max size of each log entry (100 KB). We truncate logs if
	// they are bigger than this size.
	MaxLogEntrySize = 102400
)

// basePath is the root location of the cloudprober source code at the build
// time, e.g. /Users/manugarg/code/cloudprober. We trim this path from the
// logged source file names. We set this in the init() function.
var basePath string

// We trim this path from the logged source function name.
const basePackage = "github.com/cloudprober/cloudprober/"

var defaultWritter = io.Writer(os.Stderr)

func replaceAttrs(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.SourceKey {
		source := a.Value.Any().(*slog.Source)
		source.File = strings.TrimPrefix(source.File, basePath)
		source.Function = strings.TrimPrefix(source.Function, basePackage)
	}
	return a
}

func parseMinLogLevel() slog.Level {
	switch strings.ToUpper(*minLogLevel) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	case "CRITICAL":
		return criticalLevel
	}
	panic("invalid log level: " + *minLogLevel)
}

func slogHandler(w io.Writer) slog.Handler {
	if w == nil {
		w = defaultWritter
	}
	opts := &slog.HandlerOptions{
		AddSource:   true,
		ReplaceAttr: replaceAttrs,
	}

	switch *logFmt {
	case "json":
		return slog.NewJSONHandler(w, opts)
	case "text":
		return slog.NewTextHandler(w, opts)
	}
	panic("invalid log format: " + *logFmt)
}

func enableDebugLog(debugLog bool, debugLogRe string, attrs ...slog.Attr) bool {
	if !debugLog && !isEnvSet(EnvVars.DebugLog) && debugLogRe == "" {
		return false
	}

	if (debugLog || isEnvSet(EnvVars.DebugLog)) && debugLogRe == "" {
		// Enable for all logs, regardless of log names.
		return true
	}

	r, err := regexp.Compile(debugLogRe)
	if err != nil {
		panic(fmt.Sprintf("error while parsing log name regex (%s): %v", debugLogRe, err))
	}

	for _, attr := range attrs {
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
	shandler            slog.Handler
	gcpLogc             *logging.Client
	gcpLogger           *logging.Logger
	minLogLevel         slog.Level
	disableCloudLogging bool
	gcpLoggingEndpoint  string
	attrs               []slog.Attr
	systemAttr          string
	writer              io.Writer
}

// Option can be used for adding additional metadata information in logger.
type Option func(*Logger)

// NewWithAttrs is a shortcut to create a new logger with a set of attributes.
func NewWithAttrs(attrs ...slog.Attr) *Logger {
	return New(WithAttr(attrs...))
}

// New returns a new cloudprober Logger.
func New(opts ...Option) *Logger {
	return newLogger(opts...)
}

// NewLegacy returns a new cloudprober Logger.
// Deprecated: Use New or NewWithAttrs instead.
func NewLegacy(ctx context.Context, logName string, opts ...Option) (*Logger, error) {
	return newLogger(append(opts, WithAttr(slog.String("name", logName)))...), nil
}

func newLogger(opts ...Option) *Logger {
	l := &Logger{
		disableCloudLogging: *disableCloudLogging,
		gcpLoggingEndpoint:  *gcpLoggingEndpoint,
		systemAttr:          defaultSystemName,
		minLogLevel:         parseMinLogLevel(),
	}

	if l.gcpLoggingEndpoint == "" && isEnvSet(EnvVars.GCPLoggingEndpoint) {
		l.gcpLoggingEndpoint = os.Getenv(EnvVars.GCPLoggingEndpoint)
	}

	for _, opt := range opts {
		opt(l)
	}

	l.attrs = append([]slog.Attr{slog.String("system", l.systemAttr)}, l.attrs...)

	// Initialize the traditional logger.
	l.shandler = slogHandler(l.writer).WithAttrs(l.attrs)

	if enableDebugLog(*debugLog, *debugLogList, l.attrs...) {
		l.minLogLevel = slog.LevelDebug
	}

	if metadata.OnGCE() && !l.disableCloudLogging && !isEnvSet(EnvVars.DisableCloudLogging) {
		l.EnableStackdriverLogging()
	}
	return l
}

// WithAttr option can be used to add a set of labels to all logs, e.g.
// logger.New(ctx, logName, logger.WithAttr(myLabels))
func WithAttr(attrs ...slog.Attr) Option {
	return func(l *Logger) {
		for _, attr := range attrs {
			if attr.Key == "system" {
				l.systemAttr = attr.Value.String()
				continue
			}
			l.attrs = append(l.attrs, attr)
		}
	}
}

func WithWriter(w io.Writer) Option {
	return func(l *Logger) {
		l.writer = w
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
	prefix := l.systemAttr
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
	disableCloudLoggingMsg := "Set flag --disable_cloud_logging to explicitly disable cloud logging."
	logName, err := l.sdLogName()
	if err != nil {
		l.Warningf("Error getting log name for google cloud logging: %v, will skip. %s", err, disableCloudLoggingMsg)
		return
	}

	projectID, err := metadata.ProjectID()
	if err != nil {
		l.Warningf("Error getting project id for google cloud logging: %v, will skip. %s", err, disableCloudLoggingMsg)
		return
	}

	// Create Client options for logging client
	o := []option.ClientOption{option.WithTokenSource(google.ComputeTokenSource(""))}
	if l.gcpLoggingEndpoint != "" {
		l.Infof("Setting logging endpoint to %s", l.gcpLoggingEndpoint)
		o = append(o, option.WithEndpoint(l.gcpLoggingEndpoint))
	}

	l.gcpLogc, err = logging.NewClient(context.Background(), projectID, o...)
	if err != nil {
		l.Warningf("Error creating client for google cloud logging: %v, will skip. %s", err, disableCloudLoggingMsg)
		return
	}

	commonLabels := make(map[string]string)
	// Add instance_name to common labels if available.
	if !md.IsKubernetes() && !md.IsCloudRunJob() && !md.IsCloudRunService() {
		instanceName, err := metadata.InstanceName()
		if err != nil {
			l.Infof("Error getting instance name on GCE: %v", err)
		} else {
			commonLabels["instance_name"] = instanceName
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
		logging.CommonLabels(commonLabels),
	}

	l.gcpLogger = l.gcpLogc.Logger(logName, loggerOpts...)
}

func (l *Logger) gcpLogEntry(r *slog.Record) logging.Entry {
	// Let's print the log message.
	var buf bytes.Buffer
	slogHandler(&buf).Handle(context.Background(), *r)

	return logging.Entry{
		Severity: map[slog.Level]logging.Severity{
			slog.LevelDebug: logging.Debug,
			slog.LevelInfo:  logging.Info,
			slog.LevelWarn:  logging.Warning,
			slog.LevelError: logging.Error,
			criticalLevel:   logging.Critical,
		}[r.Level],
		Payload: buf.String(),
	}
}

func (l *Logger) skipLog(level slog.Level) bool {
	if l != nil {
		return l.minLogLevel > level
	}
	return parseMinLogLevel() > level
}

// logAttrs logs the message to stderr with the given attributes. If
// running on GCE, logs are also sent to GCE or cloud logging.
func (l *Logger) logAttrs(level slog.Level, depth int, msg string, attrs ...slog.Attr) {
	// Debug logs' skip behavior is handled outside of this function so don't
	// decide on them here.
	if level != slog.LevelDebug && l.skipLog(level) {
		return
	}

	depth++

	if len(msg) > MaxLogEntrySize {
		truncateMsg := "... (truncated)"
		truncateMsgLen := len(truncateMsg)
		msg = msg[:MaxLogEntrySize-truncateMsgLen] + truncateMsg
	}

	var pcs [1]uintptr
	runtime.Callers(depth, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.AddAttrs(attrs...)

	if l != nil && l.shandler != nil {
		l.shandler.Handle(context.Background(), r)
	} else {
		slogHandler(nil).Handle(context.Background(), r)
	}

	if l != nil && l.gcpLogger != nil {
		l.gcpLogger.Log(l.gcpLogEntry(&r))
	}

	if level == criticalLevel {
		if l != nil && l.gcpLogc != nil {
			l.gcpLogc.Close()
		}
		os.Exit(1)
	}
}

func (l *Logger) logDebug() bool {
	if l != nil {
		return l.minLogLevel == slog.LevelDebug
	}
	return enableDebugLog(*debugLog, *debugLogList) || parseMinLogLevel() == slog.LevelDebug
}

// Debug logs messages with logging level set to "Debug".
func (l *Logger) Debug(payload ...string) {
	if l.logDebug() {
		l.logAttrs(slog.LevelDebug, 2, strings.Join(payload, ""))
	}
}

// DebugAttrs logs messages with logging level set to "Debug".
func (l *Logger) DebugAttrs(msg string, attrs ...slog.Attr) {
	if l.logDebug() {
		l.logAttrs(slog.LevelDebug, 2, msg, attrs...)
	}
}

// Info logs messages with logging level set to "Info".
func (l *Logger) Info(payload ...string) {
	l.logAttrs(slog.LevelInfo, 2, strings.Join(payload, ""))
}

// InfoAttrs logs messages with logging level set to "Info".
func (l *Logger) InfoAttrs(msg string, attrs ...slog.Attr) {
	l.logAttrs(slog.LevelInfo, 2, msg, attrs...)
}

// Warning logs messages with logging level set to "Warning".
func (l *Logger) Warning(payload ...string) {
	l.logAttrs(slog.LevelWarn, 2, strings.Join(payload, ""))
}

// WarningAttrs logs messages with logging level set to "Warning".
func (l *Logger) WarningAttrs(msg string, attrs ...slog.Attr) {
	l.logAttrs(slog.LevelWarn, 2, msg, attrs...)
}

// Error logs messages with logging level set to "Error".
func (l *Logger) Error(payload ...string) {
	l.logAttrs(slog.LevelError, 2, strings.Join(payload, ""))
}

// ErrorAttrs logs messages with logging level set to "Warning".
func (l *Logger) ErrorAttrs(msg string, attrs ...slog.Attr) {
	l.logAttrs(slog.LevelError, 2, msg, attrs...)
}

// Critical logs messages with logging level set to "Critical" and
// exits the process with error status. The buffer is flushed before exiting.
func (l *Logger) Critical(payload ...string) {
	l.logAttrs(criticalLevel, 2, strings.Join(payload, ""))
}

// CriticalAttrs logs messages with logging level set to "Critical" and
// exits the process with error status. The buffer is flushed before exiting.
func (l *Logger) CriticalAttrs(msg string, attrs ...slog.Attr) {
	l.logAttrs(criticalLevel, 2, msg, attrs...)
}

// Debugf logs formatted text messages with logging level "Debug".
func (l *Logger) Debugf(format string, args ...interface{}) {
	if l.logDebug() {
		l.logAttrs(slog.LevelDebug, 2, fmt.Sprintf(format, args...))
	}
}

// Infof logs formatted text messages with logging level "Info".
func (l *Logger) Infof(format string, args ...interface{}) {
	l.logAttrs(slog.LevelInfo, 2, fmt.Sprintf(format, args...))
}

// Warningf logs formatted text messages with logging level "Warning".
func (l *Logger) Warningf(format string, args ...interface{}) {
	l.logAttrs(slog.LevelWarn, 2, fmt.Sprintf(format, args...))
}

// Errorf logs formatted text messages with logging level "Error".
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logAttrs(slog.LevelError, 2, fmt.Sprintf(format, args...))
}

// Criticalf logs formatted text messages with logging level "Critical" and
// exits the process with error status. The buffer is flushed before exiting.
func (l *Logger) Criticalf(format string, args ...interface{}) {
	l.logAttrs(criticalLevel, 2, fmt.Sprintf(format, args...))
}

// init initializes basePath. We generally avoid init() but initializing
// basePath here, instead of newLogger, makes sense as we support 'nil' logger
// as well and newLogger will not be called in that case.
func init() {
	var pcs [1]uintptr
	runtime.Callers(1, pcs[:])
	frame, _ := runtime.CallersFrames(pcs[:]).Next()
	basePath = strings.TrimSuffix(frame.File, "logger/logger.go")
}
