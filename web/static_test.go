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

package web

import (
	"bytes"
	"os"
	"testing"
)

// The brand assets in static/ are copies of the kit under docs/brand, and they
// are embedded into the binary, so a copy that was not refreshed ships a stale
// logo with nothing to notice it. Keep them byte-identical to their sources;
// see the web/static table in docs/brand/README.md.
func TestBrandAssetsMatchBrandKit(t *testing.T) {
	for name, src := range map[string]string{
		"cloudprober-horizontal.svg": "../docs/brand/svg/cloudprober-horizontal.svg",
		"cloudprober-icon.svg":       "../docs/brand/svg/cloudprober-icon.svg",
		"favicon.ico":                "../docs/brand/favicon/favicon.ico",
	} {
		t.Run(name, func(t *testing.T) {
			// Read through the embed.FS, so we compare what actually ships.
			got, err := content.ReadFile("static/" + name)
			if err != nil {
				t.Fatalf("static/%s is not embedded: %v", name, err)
			}
			want, err := os.ReadFile(src)
			if err != nil {
				t.Fatalf("reading brand kit source: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("web/static/%s (%d bytes) differs from %s (%d bytes); re-copy it, see docs/brand/README.md", name, len(got), src, len(want))
			}
		})
	}
}
