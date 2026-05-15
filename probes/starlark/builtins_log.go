// Copyright 2026 The Cloudprober Authors.
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

package starlark

import (
	starlarklib "go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// log.{info,warn,error,debug}(msg) routes through the per-target logger
// stashed on the thread by runProbe (or the probe-level logger during
// load-time evaluation in newRuntime). Single-string signature; scripts
// build composite messages with Starlark's % operator before calling.

func logModule() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "log",
		Members: starlarklib.StringDict{
			"info":     starlarklib.NewBuiltin("log.info", logAt("info")),
			"warn":     starlarklib.NewBuiltin("log.warn", logAt("warn")),
			"error":    starlarklib.NewBuiltin("log.error", logAt("error")),
			"debug":    starlarklib.NewBuiltin("log.debug", logAt("debug")),
			"set_attr": starlarklib.NewBuiltin("log.set_attr", logSetAttr),
		},
	}
}

// logSetAttr installs an attribute (key=value) on the per-run logger so all
// subsequent log calls from the script — and the probe-failure log line
// runProbe writes if probe() returns an error — carry it. Sticky for the
// duration of one probe() invocation; cleared on the next.
func logSetAttr(thread *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	var key, value string
	if err := starlarklib.UnpackArgs("log.set_attr", args, kwargs,
		"key", &key,
		"value", &value,
	); err != nil {
		return nil, err
	}
	loggerHolderFromThread(thread).SetAttr(key, value)
	return starlarklib.None, nil
}

func logAt(level string) func(*starlarklib.Thread, *starlarklib.Builtin, starlarklib.Tuple, []starlarklib.Tuple) (starlarklib.Value, error) {
	return func(thread *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
		var msg string
		if err := starlarklib.UnpackArgs("log."+level, args, kwargs, "msg", &msg); err != nil {
			return nil, err
		}
		l := loggerFromThread(thread)
		switch level {
		case "info":
			l.Info(msg)
		case "warn":
			l.Warning(msg)
		case "error":
			l.Error(msg)
		case "debug":
			l.Debug(msg)
		}
		return starlarklib.None, nil
	}
}
