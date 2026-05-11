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

package logger

import (
	"flag"
	"os"
	"strings"
)

// StripDeprecatedFlags removes -logtostderr from os.Args so callers can still
// pass it for backwards compatibility without cloudprober declaring the flag
// (which would conflict with packages like glog that also declare it). If
// another package has already declared -logtostderr, this is a no-op so that
// package parses the value normally. Call before flag.Parse.
func StripDeprecatedFlags() {
	if flag.Lookup("logtostderr") != nil {
		return
	}
	os.Args = stripLogtostderr(os.Args)
}

func stripLogtostderr(args []string) []string {
	out := make([]string, 0, len(args))
	for i, arg := range args {
		if arg == "--" {
			return append(out, args[i:]...)
		}
		if arg == "-logtostderr" || arg == "--logtostderr" ||
			strings.HasPrefix(arg, "-logtostderr=") || strings.HasPrefix(arg, "--logtostderr=") {
			continue
		}
		out = append(out, arg)
	}
	return out
}
