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
	"flag"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	homeURL      = flag.String("home_url", "", "Home URL for the documentation.")
	outFmt       = flag.String("format", "yaml", "textpb or yaml")
	outDir       = flag.String("out_dir", "docs", "Output directory for the documentation.")
	protoRootDir = flag.String("proto_root_dir", ".", "Root directory for the proto files.")
)

type Token struct {
	Prefix  string
	Suffix  template.HTML
	Comment string
	Kind    string
	Text    string
	URL     string
	Default string

	PrefixHTML template.HTML
	TextHTML   template.HTML
}

type formatter struct {
	yaml   bool
	depth  int
	prefix string
}

func (f formatter) WithDepth(depth int) formatter {
	f2 := f
	f2.depth = depth
	return f2
}

func (f formatter) WithPrefix(prefix string) formatter {
	f2 := f
	f2.prefix = prefix
	return f2
}

func finalToToken(fld protoreflect.FieldDescriptor, f formatter, nocomment bool) *Token {
	var comment string
	if !nocomment {
		comment = formatComment(fld, f)
	}

	kind := fld.Kind().String()
	if fld.Kind() == protoreflect.MessageKind {
		kind = string(fld.Message().FullName())
	}

	tok := &Token{
		Prefix:  f.prefix,
		Comment: comment,
		Kind:    kind,
		Text:    string(fld.Name()),
	}

	if fld.HasDefault() {
		tok.Default = fld.Default().String()
	}

	if f.yaml {
		tok.Text = fld.JSONName()
	}

	return tok
}

func fieldToTokens(fld protoreflect.FieldDescriptor, f formatter, done *map[string]bool) *Token {
	ed := fld.Enum()
	if ed != nil {
		name := string(fld.Name())
		if f.yaml {
			name = fld.JSONName()
		}
		if (*done)[string(ed.Name())] {
			return nil
		}
		(*done)[string(ed.Name())] = true
		return formatEnum(ed, name, f)
	}

	oo := fld.ContainingOneof()
	if oo != nil {
		if (*done)[string(oo.Name())] {
			return nil
		}
		(*done)[string(oo.Name())] = true
		return formatOneOf(oo, f)
	}

	return finalToToken(fld, f, false)
}

func dumpExtendedMsg(fld protoreflect.FieldDescriptor, f formatter) ([]*Token, []protoreflect.FullName) {
	var nextMessageName []protoreflect.FullName
	var lines []*Token

	nextMessageName = append(nextMessageName, fld.Message().FullName())
	tok := finalToToken(fld, f, false)
	tok.Suffix = " {"
	if f.yaml {
		tok.Suffix = ":"
	}
	lines = append(lines, tok)

	newPrefix := f.prefix + "  "
	if fld.Cardinality() == protoreflect.Repeated && f.yaml {
		newPrefix = f.prefix + "    "
	}
	toks, next := dumpMessage(fld.Message(), f.WithDepth(f.depth-1).WithPrefix(newPrefix))
	if f.yaml && fld.Cardinality() == protoreflect.Repeated {
		toks[0].Prefix = f.prefix + "  - "
	}
	lines = append(lines, toks...)

	// If it's not a yaml, add a "}" at the end and limit the line break before
	// that to just one (default is 2).
	if !f.yaml {
		lines[len(lines)-1].Suffix = "<br>"
		lines = append(lines, &Token{Prefix: f.prefix, Text: "}"})
	}
	nextMessageName = append(nextMessageName, next...)

	return lines, nextMessageName
}

func dumpMessage(md protoreflect.MessageDescriptor, f formatter) ([]*Token, []protoreflect.FullName) {
	var nextMessageName []protoreflect.FullName

	var lines []*Token

	// We use this to catch duplication for oneof and enum fields.
	done := map[string]bool{}

	for i := 0; i < md.Fields().Len(); i++ {
		fld := md.Fields().Get(i)

		if fld.Kind() == protoreflect.MessageKind && f.depth > 1 {
			toks, next := dumpExtendedMsg(fld, f)
			lines = append(lines, toks...)
			nextMessageName = append(nextMessageName, next...)
		} else {
			if tok := fieldToTokens(md.Fields().Get(i), f, &done); tok != nil {
				lines = append(lines, tok)
			}
			if fld.Kind() == protoreflect.MessageKind {
				nextMessageName = append(nextMessageName, fld.Message().FullName())
			}
		}
	}

	return lines, nextMessageName
}

func processTokensForHTML(toks []*Token) []*Token {
	for _, tok := range toks {
		tok.PrefixHTML = template.HTML(strings.ReplaceAll(tok.Prefix, " ", "&nbsp;"))

		tok.URL = kindToURL(tok.Kind)

		if tok.Suffix == "" {
			tok.Suffix = template.HTML("<br><br>")
			if tok.Default != "" {
				tok.Suffix = template.HTML(" | default: " + tok.Default + "<br><br>")
			}
		} else {
			if !strings.HasSuffix(string(tok.Suffix), "<br>") {
				tok.Suffix = template.HTML(tok.Suffix + "<br>")
			}
		}

		if tok.TextHTML == "" {
			tok.TextHTML = template.HTML(template.HTMLEscapeString(tok.Text))
		}
	}
	return toks
}

func main() {
	flag.Parse()

	l := &logger.Logger{}

	buildFileDescRegistry(*protoRootDir, l)

	// Top level message
	m, err := files.FindDescriptorByName("cloudprober.ProberConfig")
	if err != nil {
		panic(err)
	}

	f := formatter{
		yaml: *outFmt == "yaml",
	}

	toks, nextMessageNames := dumpMessage(m.(protoreflect.MessageDescriptor), f.WithDepth(2))
	outF, err := os.OpenFile(filepath.Join(*outDir, "index.html"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		l.Criticalf("Error opening output file: %v", err)
	}
	template.Must(template.New("index").Parse(indexTmpl)).Execute(outF, processTokensForHTML(toks))

	msgToDoc := map[string][]*Token{}
	for len(nextMessageNames) > 0 {
		var nextLoop []protoreflect.FullName
		for _, msgName := range nextMessageNames {
			m, err := files.FindDescriptorByName(protoreflect.FullName(msgName))
			if err != nil {
				panic(err)
			}
			toks, next := dumpMessage(m.(protoreflect.MessageDescriptor), f.WithDepth(1).WithPrefix("  "))
			msgToDoc[string(msgName)] = toks
			nextLoop = append(nextLoop, next...)
		}
		nextMessageNames = nextLoop
	}

	var msgs []string
	for key := range msgToDoc {
		msgs = append(msgs, key)
	}

	packages := arrangeIntoPackages(msgs, l)
	type msgTokens struct {
		Name   string
		Tokens []*Token
	}

	for pkg, msgs := range packages {
		sort.Strings(msgs)
		toks := []*msgTokens{}
		for _, msg := range msgs {
			toks = append(toks, &msgTokens{Name: msg, Tokens: processTokensForHTML(msgToDoc[msg])})
		}
		outF, err := os.OpenFile(filepath.Join(*outDir, pkg+".html"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			l.Criticalf("Error opening output file: %v", err)
		}
		template.Must(template.New("package").Parse(packageTmpl)).Execute(outF, toks)
	}

}
