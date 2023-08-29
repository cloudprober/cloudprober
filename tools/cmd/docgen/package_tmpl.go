package main

var packageTmpl = `
<html>
<head>
    <title>Cloudprober Configuration</title>
    <style>
        body {
            font-family: monospace;
        }
        .comment {
            color: #888;
            font-family: monospace;
            white-space: pre;
        }
    </style>
</head>
<body>
{{range .}}
<h2 id="{{ .Name }}">{{ .Name }}</h2>
    {{range .Tokens}}
        {{- if .Comment}}
            <div class="comment">{{.Comment}}</div>
        {{- end -}}
        {{- if .URL}}
            {{- .PrefixHTML}}{{- .TextHTML}}: <<a href="{{.URL}}">{{- .Kind}}</a>>{{- .Suffix }}
        {{- else if .Kind}}
            {{- .PrefixHTML}}{{- .TextHTML}}: <{{- .Kind}}>{{- .Suffix }}
        {{- else}}
            {{- .PrefixHTML}}{{- .TextHTML}}{{- .Suffix }}
        {{- end}}
    {{end}}
{{end}}
</body>
</html>`
