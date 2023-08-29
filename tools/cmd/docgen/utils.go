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

package main

import (
	"strings"

	"github.com/cloudprober/cloudprober/logger"
)

func kindToURL(kind string) string {
	if !strings.HasPrefix(kind, "cloudprober.") {
		return ""
	}
	parts := strings.SplitN(kind, ".", 3)
	if len(parts) > 2 {
		return *homeURL + parts[1] + ".html#" + kind
	}
	return ""
}

func arrangeIntoPackages(paths []string, l *logger.Logger) map[string][]string {
	packages := make(map[string][]string)
	for _, path := range paths {
		parts := strings.SplitN(path, ".", 3)
		if len(parts) < 3 {
			l.Warningf("Skipping %s, not enough parts in package", path)
			continue
		}
		if parts[0] != "cloudprober" {
			l.Warningf("Skipping %s, not a cloudprober package", path)
			continue
		}
		packages[parts[1]] = append(packages[parts[1]], path)
	}
	return packages
}
