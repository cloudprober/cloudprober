// Copyright 2025 The Cloudprober Authors.
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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindModuleRoot(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() (string, func())
		want    string
		wantErr bool
	}{
		{
			name: "finds module root with go.mod",
			setup: func() (string, func()) {
				tempDir, err := os.MkdirTemp("", "testmod")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				// Create a go.mod file
				if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module testmod\n"), 0644); err != nil {
					os.RemoveAll(tempDir)
					t.Fatalf("Failed to create go.mod: %v", err)
				}

				// Create a subdirectory to test from
				subDir := filepath.Join(tempDir, "subdir")
				if err := os.Mkdir(subDir, 0755); err != nil {
					os.RemoveAll(tempDir)
					t.Fatalf("Failed to create subdir: %v", err)
				}

				// Change to the subdirectory
				oldWd, err := os.Getwd()
				if err != nil {
					os.RemoveAll(tempDir)
					t.Fatalf("Failed to get current directory: %v", err)
				}
				if err := os.Chdir(subDir); err != nil {
					os.RemoveAll(tempDir)
					t.Fatalf("Failed to change directory: %v", err)
				}

				return tempDir, func() {
					os.Chdir(oldWd)
					os.RemoveAll(tempDir)
				}
			},
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := func() {}
			if tt.setup != nil {
				_, cleanup = tt.setup()
				defer cleanup()
			}

			got, err := findModuleRoot()
			if (err != nil) != tt.wantErr {
				t.Errorf("findModuleRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == "" {
				t.Error("Expected non-empty module root, got empty string")
			}
		})
	}
}

func TestProcessReExports(t *testing.T) {
	tests := []struct {
		name      string
		reExports map[string][]ReExport
		protoroot string
		want      []string
	}{
		{
			name: "sorts and moves protoroot to front",
			reExports: map[string][]ReExport{
				"github.com/cloudprober/cloudprober/internal/validators/proto": {
					{Alias: "Validator"},
					{Alias: "HttpValidator"},
				},
				"github.com/cloudprober/cloudprober/internal/validators/http/proto": {
					{Alias: "ProbeDef"},
				},
			},
			protoroot: "internal/validators",
			want: []string{
				"github.com/cloudprober/cloudprober/internal/validators/proto",
				"github.com/cloudprober/cloudprober/internal/validators/http/proto",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, processReExports(tt.reExports, tt.protoroot))
			for _, v := range tt.reExports {
				// Verify that the aliases are sorted
				for i := 1; i < len(v); i++ {
					if v[i-1].Alias > v[i].Alias {
						t.Errorf("aliases are not sorted: %v", v)
					}
				}
			}
		})
	}
}

func TestFindProtoDirs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "testprotoroot")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a temporary go.mod in the tempDir
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module testmodule\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(oldWd)

	// Create a test directory structure
	testDirs := []string{
		filepath.Join(tempDir, "internal/pkg1/proto"),
		filepath.Join(tempDir, "internal/pkg1/nested/proto"),
		filepath.Join(tempDir, "internal/pkg2"), // No proto dir here
	}

	for _, dir := range testDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create test dir %s: %v", dir, err)
		}
	}

	tests := []struct {
		name      string
		protoroot string
		module    string
		want      []string
		wantErr   bool
	}{
		{
			name:      "finds all proto dirs in pkg1",
			protoroot: "internal/pkg1",
			module:    "testmodule",
			want: []string{
				filepath.ToSlash("testmodule/internal/pkg1/nested/proto"),
				filepath.ToSlash("testmodule/internal/pkg1/proto"),
			},
			wantErr: false,
		},
		{
			name:      "finds all proto dirs in pkg2",
			protoroot: "internal/pkg2",
			module:    "testmodule",
			want:      []string{},
			wantErr:   false,
		},
		{
			name:      "finds all proto dirs in pkg3",
			protoroot: "internal/pkg3",
			module:    "testmodule",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run the test
			got, err := findProtoDirs(tt.protoroot, tt.module)

			// Verify results
			if (err != nil) != tt.wantErr {
				t.Errorf("findProtoDirs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.ElementsMatch(t, got, tt.want)
		})
	}
}

func TestImportAliasFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "standard proto path",
			path: "internal/validators/http/proto",
			want: "httppb",
		},
		{
			name: "short path",
			path: "proto",
			want: "pb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := importAliasFromPath(tt.path); got != tt.want {
				t.Errorf("importAliasFromPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputPackageName(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "testpkg")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create another directory
	subDir := filepath.Join(tempDir, "subpkg")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subpkg: %v", err)
	}

	// Change to the subdirectory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(oldWd)

	tests := []struct {
		name    string
		out     string
		want    string
		wantErr bool
	}{
		{
			name: "simple path",
			out:  "/path/to/package/file.go",
			want: "package",
		},
		{
			name: "current directory",
			out:  "file.go",
			want: "subpkg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := outputPackageName(tt.out)
			assert.NoError(t, err)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestGenerateTemplateData(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "testprotoroot")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test module structure
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module testmodule\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create test proto package structure
	testProtoPkgs := []string{
		filepath.Join(tempDir, "testpkg", "subpkg1", "proto"),
		filepath.Join(tempDir, "testpkg", "subpkg2", "proto"),
	}
	for _, dir := range testProtoPkgs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create test proto dir: %v", err)
		}

		// Create a simple Go file with some types
		testGoFile := filepath.Join(dir, "test.pb.go")
		testGoContent := `// Code generated by protoc-gen-go. DO NOT EDIT.
package proto

type TestMessage struct {
	Field1 string
	Field2 int
}

type AnotherType struct {
	Value string
}
`
		if err := os.WriteFile(testGoFile, []byte(testGoContent), 0644); err != nil {
			t.Fatalf("Failed to create test Go file: %v", err)
		}
	}

	tests := []struct {
		name      string
		protoroot string
		module    string
		want      TemplateData
		wantErr   bool
	}{
		{
			name:      "successful template generation",
			protoroot: "testpkg",
			module:    "testmodule",
			want: TemplateData{
				Imports: []Import{
					{
						Alias: "subpkg1pb",
						Path:  "testmodule/testpkg/subpkg1/proto",
					},
					{
						Alias: "subpkg2pb",
						Path:  "testmodule/testpkg/subpkg2/proto",
					},
				},
				ReExportOrder: []string{
					"testmodule/testpkg/subpkg1/proto",
					"testmodule/testpkg/subpkg2/proto",
				},
				ImportReExports: map[string][]ReExport{
					"testmodule/testpkg/subpkg1/proto": {
						{Alias: "Subpkg1TestMessage", Original: "TestMessage", ImportAlias: "subpkg1pb"},
						{Alias: "Subpkg1AnotherType", Original: "AnotherType", ImportAlias: "subpkg1pb"},
					},
					"testmodule/testpkg/subpkg2/proto": {
						{Alias: "Subpkg2TestMessage", Original: "TestMessage", ImportAlias: "subpkg2pb"},
						{Alias: "Subpkg2AnotherType", Original: "AnotherType", ImportAlias: "subpkg2pb"},
					},
				},
			},
			wantErr: false,
		},
		{
			name:      "non-existent directory",
			protoroot: filepath.Join(tempDir, "nonexistent"),
			module:    "testmodule/testpkg",
			want:      TemplateData{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the current working directory
			oldWd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer os.Chdir(oldWd)

			// Change to the temp directory for the test
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change directory: %v", err)
			}

			// Run the function under test
			got, err := generateTemplateData(tt.protoroot, tt.module)

			// Check for expected errors
			if (err != nil) != tt.wantErr {
				t.Fatalf("generateTemplateData() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If we expected an error, we're done
			if tt.wantErr {
				return
			}

			// Verify the imports
			if len(got.Imports) != len(tt.want.Imports) {
				t.Fatalf("Expected %d imports, got %d", len(tt.want.Imports), len(got.Imports))
			}

			// Verify the import paths and aliases
			assert.Equal(t, tt.want.Imports, got.Imports)

			// Verify the re-export order
			assert.Equal(t, tt.want.ReExportOrder, got.ReExportOrder)
			/*
				// Verify each import's re-exports
				for pkg, reExports := range got.ImportReExports {
					expectedExports, ok := tt.want.ImportReExports[pkg]
					if !ok {
						t.Errorf("Unexpected package in re-exports: %s", pkg)
						continue
					}

					if len(reExports) != len(expectedExports) {
						t.Errorf("Package %s: expected %d re-exports, got %d",
							pkg, len(expectedExports), len(reExports))
						continue
					}

					// Verify each re-export
					exportMap := make(map[string]ReExport)
					for _, exp := range reExports {
						exportMap[exp.Original] = exp
					}

					for _, expected := range expectedExports {
						got, exists := exportMap[expected.Original]
						if !exists {
							t.Errorf("Missing re-export for %s in package %s",
								expected.Original, pkg)
							continue
						}

						if got.Alias != expected.Alias || got.ImportAlias != expected.ImportAlias {
							t.Errorf("Re-export for %s: got %+v, want %+v",
								expected.Original, got, expected)
						}
					}
				}
			*/
		})
	}
}

func TestCleanAlias(t *testing.T) {
	tests := []struct {
		name        string
		importAlias string
		origName    string
		protoroot   string
		want        string
	}{
		{
			name:        "simple case",
			importAlias: "httppb",
			origName:    "Validator",
			protoroot:   "internal/validators",
			want:        "HttpValidator",
		},
		{
			name:        "with underscore in name",
			importAlias: "httppb",
			origName:    "Validator_Header",
			protoroot:   "internal/validators",
			want:        "HttpValidator_Header",
		},
		{
			name:        "matching protoroot",
			importAlias: "validatorspb",
			origName:    "Validator",
			protoroot:   "internal/validators",
			want:        "Validator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanAlias(tt.importAlias, tt.origName, tt.protoroot); got != tt.want {
				t.Errorf("cleanAlias() = %v, want %v", got, tt.want)
			}
		})
	}
}
