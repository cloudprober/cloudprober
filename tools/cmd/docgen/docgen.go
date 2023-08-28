package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

var (
	homeURL = flag.String("home_url", "/", "Home URL for the documentation.")
	outDir  = flag.String("out_dir", "docs", "Output directory for the documentation.")
)

var files = &protoregistry.Files{}

type token struct {
	comment string
	kind    string
	name    string
}

func finalFieldStr(f protoreflect.FieldDescriptor, prefix string, nocomment bool) string {
	var comment string
	if !nocomment {
		ff, err := files.FindDescriptorByName(f.FullName())
		if err != nil {
			panic(err)
		}
		wf, err := desc.WrapDescriptor(ff)
		if err != nil {
			panic(err)
		}
		comment = wf.GetSourceInfo().GetLeadingComments()
		if comment != "" && strings.TrimSpace(comment) != "" {
			var temp []string
			for _, line := range strings.Split(comment, "\n") {
				temp = append(temp, prefix+"#"+line)
			}
			comment = "<div class=comment>" + strings.Join(temp, "\n") + "</div>"
		}
	}
	if f.Kind() == protoreflect.MessageKind {
		return fmt.Sprintf("%s%s%s: <%s>", comment, prefix, f.Name(), f.Message().FullName())
	}
	return fmt.Sprintf("%s%s%s: <%s>", comment, prefix, f.Name(), f.Kind())
}

func oneOfToString(ood protoreflect.OneofDescriptor, prefix string) string {
	oof := ood.Fields()
	oneofFields := []string{}
	for i := 0; i < oof.Len(); i++ {
		oneofFields = append(oneofFields, finalFieldStr(oof.Get(i), "", true))
	}
	return fmt.Sprintf("%s[%s]", prefix, strings.Join(oneofFields, "|"))
}

func enumToString(ed protoreflect.EnumDescriptor, name, prefix string) string {
	enumVals := []string{}
	for i := 0; i < ed.Values().Len(); i++ {
		enumVals = append(enumVals, string(ed.Values().Get(i).Name()))
	}
	return fmt.Sprintf("%s%s: (%s)", prefix, name, strings.Join(enumVals, "|"))
}

func dedupPrint(s string, printedFields *map[string]bool) string {
	if !(*printedFields)[s] {
		(*printedFields)[s] = true
		return s
	}
	return ""
}

func printNonMessageField(f protoreflect.FieldDescriptor, prefix string, printedFields *map[string]bool) string {
	ed := f.Enum()
	if ed != nil {
		return dedupPrint(enumToString(ed, string(f.Name()), prefix), printedFields)
	}

	oo := f.ContainingOneof()
	if oo != nil {
		return dedupPrint(oneOfToString(oo, prefix), printedFields)
	}

	return finalFieldStr(f, prefix, false)
}

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

type formatter struct {
	depth  int
	prefix string
}

func dumpProtoDescriptor(md protoreflect.MessageDescriptor, f formatter) (string, []protoreflect.FullName) {
	var nextMessageName []protoreflect.FullName

	var lines []string

	// We use this to catch duplication for oneof and enum fields.
	printedFields := map[string]bool{}

	for i := 0; i < md.Fields().Len(); i++ {
		fld := md.Fields().Get(i)

		if fld.Kind() == protoreflect.MessageKind && f.depth > 1 {
			header := finalFieldStr(fld, f.prefix, false)
			line, next := dumpProtoDescriptor(fld.Message(), formatter{
				prefix: f.prefix + "  ",
				depth:  1,
			})
			lines = append(lines, header+"\n"+line)
			nextMessageName = append(nextMessageName, next...)
		} else {
			if line := printNonMessageField(md.Fields().Get(i), f.prefix+"", &printedFields); line != "" {
				lines = append(lines, line)
			}
			if fld.Kind() == protoreflect.MessageKind {
				nextMessageName = append(nextMessageName, fld.Message().FullName())
			}
		}
	}

	return strings.Join(lines, "\n\n"), nextMessageName
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

func main() {
	flag.Parse()

	indexTmpl := template.Must(template.New("index").Parse(indexTmpl))

	l := &logger.Logger{}
	buildFileDescRegistry(l)

	// Top level message
	m, err := files.FindDescriptorByName("cloudprober.ProberConfig")
	if err != nil {
		panic(err)
	}

	s, nextMessageNames := dumpProtoDescriptor(m.(protoreflect.MessageDescriptor), formatter{depth: 2})

	outF, err := os.OpenFile(filepath.Join(*outDir, "index.html"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		l.Criticalf("Error opening output file: %v", err)
	}
	indexTmpl.Execute(outF, map[string]interface{}{"Content": template.HTML(s)})
	// s = "```\n" + s + "\n```\n"
	os.WriteFile(filepath.Join(*outDir, "index.md"), []byte(s), 0644)

	// fmt.Println(nextMessageNames)
	msgToDoc := map[string]string{}
	for len(nextMessageNames) > 0 {
		var nextLoop []protoreflect.FullName
		for _, msgName := range nextMessageNames {
			m, err := files.FindDescriptorByName(protoreflect.FullName(msgName))
			if err != nil {
				panic(err)
			}
			s, next := dumpProtoDescriptor(m.(protoreflect.MessageDescriptor), formatter{prefix: "  ", depth: 1})
			msgToDoc[string(msgName)] = "```\n" + s + "\n```"
			nextLoop = append(nextLoop, next...)
		}
		nextMessageNames = nextLoop
	}

	var msgs []string
	for key := range msgToDoc {
		msgs = append(msgs, key)
	}
	packages := arrangeIntoPackages(msgs, l)

	for pkg, msgs := range packages {
		sort.Strings(msgs)
		var lines []string
		for _, msg := range msgs {
			lines = append(lines, "## "+msg+":\n"+msgToDoc[msg])
		}
		os.WriteFile(filepath.Join(*outDir, pkg+".md"), []byte(strings.Join(lines, "\n\n")), 0644)
	}
}
