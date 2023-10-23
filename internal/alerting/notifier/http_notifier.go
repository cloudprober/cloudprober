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

func (n *Notifier) httpNotify(ctx context.Context, fields map[string]string) error {
	newCfg := &httpreqpb.HTTPRequest{
		Method: n.httpNotifier.GetMethod(),
	}

	url, _ := strtemplate.SubstituteLabels(n.httpNotifier.GetUrl(), fields)
	newCfg.Url = url

	for k, v := range n.httpNotifier.GetHeader() {
		newCfg.Header[k], _ = strtemplate.SubstituteLabels(v, fields)
	}
	for _, d := range n.httpNotifier.GetData() {
		newData, _ := strtemplate.SubstituteLabels(d, fields)
		newCfg.Data = append(newCfg.Data, newData)
	}

	n.l.Infof("Sending HTTP notification to URL: %s", newCfg.Url)

	req, err := httpreq.FromConfig(newCfg)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending HTTP request: %v", err)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("status code: %d, error reading HTTP response body: %v", resp.StatusCode, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return fmt.Errorf("status code: %d, response: %s", resp.StatusCode, string(b))
	}

	n.l.Infof("HTTP notification sent successfully. Response code: %d, body: %s", resp.StatusCode, string(b))

	return nil
}
