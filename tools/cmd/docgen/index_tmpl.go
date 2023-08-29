package main

var indexTmpl = `
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
</body>
</html>`
