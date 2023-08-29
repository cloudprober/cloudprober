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

func buildFileDescRegistry(l *logger.Logger) {
	p := protoparse.Parser{
		ImportPaths:      []string{"."},
		InferImportPaths: true,
		Accessor: func(name string) (io.ReadCloser, error) {
			name = strings.TrimPrefix(name, "github.com/cloudprober/cloudprober/")
			return os.Open(name)
		},
		IncludeSourceCodeInfo: true,
	}

	var protos []string
	filepath.WalkDir(".", func(s string, d fs.DirEntry, e error) error {
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
