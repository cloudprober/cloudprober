// Copyright 2022 The Cloudprober Authors.
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

// Package resources provides webpages related resources.
package resources

var Style = `
<style type="text/css">
body {
  font-family: "Roboto","Helvetica","Arial",sans-serif;
	font-size: 14px;
  line-height: 1.3;
}

.debugging {
  font-size: 12px;
}

table.status-list {
  border-collapse: collapse;
  border-spacing: 0;
	margin-top: 10px;
	margin-bottom: 20px;
	font-family: monospace;
	font-size: 14px;
}
table.status-list td,th {
  border: 1px solid gray;
  padding: 0.25em 0.5em;
	max-width: 200px;
}
pre {
    white-space: pre-wrap;
		word-wrap: break-word;
}
</style>
`
