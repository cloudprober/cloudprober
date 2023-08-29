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
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/reflect/protoregistry"
)

var files = &protoregistry.Files{}

func buildFileDescRegistry(protoRoot string, l *logger.Logger) {
	p := protoparse.Parser{
		ImportPaths: []string{protoRoot, "."},
		Accessor: func(name string) (io.ReadCloser, error) {
			// For imports
			if strings.HasPrefix(name, "github.com/cloudprober/cloudprober/") {
				name = strings.TrimPrefix(name, "github.com/cloudprober/cloudprober/")
				return os.Open(filepath.Join(protoRoot, name))
			}
			// Top files
			return os.Open(name)
		},
		IncludeSourceCodeInfo: true,
	}

	var protos []string
	filepath.WalkDir(protoRoot, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(s) == ".proto" {
			protos = append(protos, s)

			// Use jhump's parser to parse protos into file descriptors
			// and register them with protoregistry.
			fds, err := p.ParseFiles(s)
			if err != nil {
				l.Errorf("Error parsing proto %s: %v", s, err)
				return err
			}
			fd := fds[0]
			files.RegisterFile(fd.UnwrapFile())
		}
		return nil
	})
}
