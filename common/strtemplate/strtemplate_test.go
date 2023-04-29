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
package strtemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubstituteLabels(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		labels    map[string]string
		want      string
		wantFound bool
	}{
		{
			name: "multiple clean substitutions",
			in:   "--target=@target@ --alert=@alert@",
			labels: map[string]string{
				"target": "cloudprober.org",
				"alert":  "WebsiteDown",
			},
			want:      "--target=cloudprober.org --alert=WebsiteDown",
			wantFound: true,
		},
		{
			name: "not all found",
			in:   "--servername=@target@ --port=@port@",
			labels: map[string]string{
				"target": "www.google.com",
			},
			want:      "--servername=www.google.com --port=@port@",
			wantFound: false,
		},
		{
			// This is an invalid substitution as the first @@ gets replaced
			// by a literal @, and rest of the string has unbalanced @ (3).
			in: "--servername=@@target@ --port=@port@",
			labels: map[string]string{
				"target": "www.google.com",
				"port":   "443",
			},
			want:      "--servername=@target@ --port=@port@",
			wantFound: false,
		},
		{
			in: "--email=user@@@domain@",
			labels: map[string]string{
				"domain": "cloudprober.org",
			},
			want:      "--email=user@cloudprober.org",
			wantFound: true,
		},
		{
			name:      "No replacement",
			in:        "foo bar baz",
			want:      "foo bar baz",
			wantFound: true,
		},
		{
			name: "Replacement beginning",
			in:   "@foo@ bar baz",
			labels: map[string]string{
				"foo": "h e llo",
			},
			want:      "h e llo bar baz",
			wantFound: true,
		},
		{
			name: "Replacement special character",
			in:   "beginning @😿@ end",
			labels: map[string]string{
				"😿": "😺",
			},
			want:      "beginning 😺 end",
			wantFound: true,
		},
		{
			name: "Replacement end",
			in:   "bar baz @foo@",
			labels: map[string]string{
				"foo": "XöX",
				"bar": "nope",
			},
			want:      "bar baz XöX",
			wantFound: true,
		},
		{
			name: "Not found",
			in:   "A b C @d@ e",
			labels: map[string]string{
				"def": "nope",
			},
			want:      "A b C @d@ e",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, foundAll := SubstituteLabels(tt.in, tt.labels)
			assert.Equal(t, tt.want, got, "Substitution mismatch")
			assert.Equal(t, tt.wantFound, foundAll, "Found mismatch")
		})
	}
}
