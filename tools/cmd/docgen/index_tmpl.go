package main

var indexTmpl = `
<html>
<head>
    <title>Cloudprober Configuration</title>
    <style>
        body {
            font-family: monospace;
            white-space: pre;
        }
        .comment {
            color: #888;
            font-family: monospace;
        }
    </style>
</head>
<body>
{{.Content}}
</body>
</html>`
