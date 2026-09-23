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
	"reflect"
	"testing"
)

func TestStripDeprecatedFlags_OtherPackageDeclared(t *testing.T) {
	origArgs := os.Args
	origCmdLine := flag.CommandLine
	t.Cleanup(func() {
		os.Args = origArgs
		flag.CommandLine = origCmdLine
	})

	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	flag.Bool("logtostderr", false, "declared by another package")

	os.Args = []string{"cmd", "-logtostderr=true", "-other"}
	StripDeprecatedFlags()

	want := []string{"cmd", "-logtostderr=true", "-other"}
	if !reflect.DeepEqual(os.Args, want) {
		t.Errorf("os.Args = %v, want %v (unchanged)", os.Args, want)
	}
}

func TestStripLogtostderr(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "absent",
			in:   []string{"cmd", "-other", "-foo=1"},
			want: []string{"cmd", "-other", "-foo=1"},
		},
		{
			name: "single dash bare",
			in:   []string{"cmd", "-logtostderr", "-other"},
			want: []string{"cmd", "-other"},
		},
		{
			name: "double dash bare",
			in:   []string{"cmd", "--logtostderr", "-other"},
			want: []string{"cmd", "-other"},
		},
		{
			name: "with value",
			in:   []string{"cmd", "-logtostderr=true", "-other"},
			want: []string{"cmd", "-other"},
		},
		{
			name: "double dash with value",
			in:   []string{"cmd", "--logtostderr=false", "-other"},
			want: []string{"cmd", "-other"},
		},
		{
			name: "does not match prefix",
			in:   []string{"cmd", "-logtostderrx", "-other"},
			want: []string{"cmd", "-logtostderrx", "-other"},
		},
		{
			name: "preserves args after --",
			in:   []string{"cmd", "-other", "--", "-logtostderr"},
			want: []string{"cmd", "-other", "--", "-logtostderr"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripLogtostderr(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stripLogtostderr(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
