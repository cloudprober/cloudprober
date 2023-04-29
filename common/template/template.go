// Copyright 2017-2022 The Cloudprober Authors.
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

/*
Package template implements string templating functionality for Cloudprober.
*/
package template

import (
	"strings"
)

// SubstituteLabels replaces occurrences of @label@ with the values from
// labels.  It returns the substituted string and a bool indicating if there
// was a @label@ that did not exist in the labels map.
func SubstituteLabels(in string, labels map[string]string) (string, bool) {
	if len(labels) == 0 {
		return in, !strings.Contains(in, "@")
	}
	delimiter := "@"
	output := ""
	words := strings.Split(in, delimiter)
	count := len(words)
	foundAll := true
	for j, kwd := range words {
		// Even number of words => just copy out.
		if j%2 == 0 {
			output += kwd
			continue
		}
		// Special case: If there are an odd number of '@' (unbalanced), the last
		// odd index doesn't actually have a closing '@', so we just append it as it
		// is.
		if j == count-1 {
			output += delimiter
			output += kwd
			continue
		}

		// Special case: "@@" => "@"
		if kwd == "" {
			output += delimiter
			continue
		}

		// Finally, the labels.
		replace, ok := labels[kwd]
		if ok {
			output += replace
			continue
		}

		// Nothing - put the token back in.
		foundAll = false
		output += delimiter
		output += kwd
		output += delimiter
	}
	return output, foundAll
}
