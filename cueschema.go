package cloudprober

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/protobuf/textproto"
	cueyaml "cuelang.org/go/pkg/encoding/yaml"
)

//go:embed cue.mod/module.cue */proto/*.cue */*/proto/*.cue
var vfs embed.FS

const entryPoint = "config/proto/config_proto_gen.cue"

var enumFieldNames = []string{
	"format",
	"ip_type",
	"ip_version",
	"method",
	"method_type",
	"metrics_kind",
	"mode",
	"protocol",
	"query_type",
	"type",
}

func cueSchemaOverlay(overlay map[string]load.Source) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current working directory: %v", err)
	}

	err = fs.WalkDir(vfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		f, err := vfs.Open(path)
		if err != nil {
			return err
		}
		defer f.Close() // nolint: errcheck

		b, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		overlay[filepath.Join(cwd, path)] = load.FromBytes(b)
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func loadCueSchema() (cue.Value, error) {
	overlay := make(map[string]load.Source)
	if err := cueSchemaOverlay(overlay); err != nil {
		return cue.Value{}, fmt.Errorf("error creating CUE schema overlay: %v", err)
	}

	bis := load.Instances([]string{entryPoint}, &load.Config{
		Overlay: overlay,
		Dir:     ".",
	})

	r := cuecontext.New()
	s := r.BuildInstance(bis[0])
	if s.Err() != nil {
		return cue.Value{}, fmt.Errorf("error building instance from CUE schema: %v", s.Err())
	}

	proberConfigDef := s.LookupPath(cue.MakePath(cue.Def("#ProberConfig")))
	if proberConfigDef.Err() != nil {
		return cue.Value{}, fmt.Errorf("error looking up #ProberConfig definition: %v", proberConfigDef.Err())
	}

	return proberConfigDef, nil
}

func fixEnum(in []byte) []byte {
	var sb strings.Builder

	re := regexp.MustCompile(fmt.Sprintf("^([ ]*(?:%s): )\"(.*)\"$", strings.Join(enumFieldNames, "|")))
	scanner := bufio.NewScanner(bytes.NewReader(in))

	for scanner.Scan() {
		b := scanner.Bytes()
		matches := re.FindSubmatch(b)
		if len(matches) >= 3 {
			sb.Write(matches[1]) // [.. type: ]
			sb.Write(matches[2]) // [PING|HTTP|...]
		} else {
			sb.Write(b)
		}
		sb.WriteByte('\n')
	}

	return []byte(sb.String())
}

func YAMLToTextproto(yamlStr string) ([]byte, error) {
	// We got a YAML file. Try to parse it using CUE.
	proberConfigDef, err := loadCueSchema()
	if err != nil {
		return nil, fmt.Errorf("error loading CUE schema: %v", err)
	}

	expr, err := cueyaml.Unmarshal([]byte(yamlStr))
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling YAML: %v", err)
	}

	v := proberConfigDef.Context().BuildExpr(expr)
	if v.Err() != nil {
		return nil, fmt.Errorf("error building cue value: %v", v.Err())
	}

	b, err := textproto.NewEncoder().Encode(v)
	if err != nil {
		return nil, fmt.Errorf("error encoding cue value (%v) into textproto: %v", v, err)
	}

	return fixEnum(b), nil
}
