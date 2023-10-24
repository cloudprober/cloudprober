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

package notifier

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/cloudprober/cloudprober/common/strtemplate"
	"github.com/cloudprober/cloudprober/internal/httpreq"
	httpreqpb "github.com/cloudprober/cloudprober/internal/httpreq/proto"
)

var doHTTPRequest = func(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}

func (n *Notifier) httpNotify(ctx context.Context, fields map[string]string) error {
	req := &httpreqpb.HTTPRequest{
		Method: n.httpNotifier.GetMethod(),
		Header: make(map[string]string),
	}

	url, _ := strtemplate.SubstituteLabels(n.httpNotifier.GetUrl(), fields)
	req.Url = url

	for k, v := range n.httpNotifier.GetHeader() {
		req.Header[k], _ = strtemplate.SubstituteLabels(v, fields)
	}
	for _, d := range n.httpNotifier.GetData() {
		d2, _ := strtemplate.SubstituteLabels(d, fields)
		req.Data = append(req.Data, d2)
	}

	n.l.Infof("Sending HTTP notification to URL: %s", req.Url)

	r, err := httpreq.FromConfig(req)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}

	resp, err := doHTTPRequest(r)
	if err != nil {
		return fmt.Errorf("error sending HTTP request: %v", err)
	}

	var respBody string
	if resp.Body != nil {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("status code: %d, error reading HTTP response body: %v", resp.StatusCode, err)
		}
		respBody = string(b)
		resp.Body.Close()
	}

	if resp.StatusCode > 299 {
		return fmt.Errorf("status code: %d, response: %s", resp.StatusCode, respBody)
	}

	n.l.Infof("HTTP notification sent successfully. Response code: %d, body: %s", resp.StatusCode, respBody)

	return nil
}
