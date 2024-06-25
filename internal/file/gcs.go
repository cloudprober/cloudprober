// Copyright 2020-2023 The Cloudprober Authors.
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

package file

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"golang.org/x/oauth2/google"
)

const gcsHTTPBaseURL = "https://storage.googleapis.com"

func gcsRequest(ctx context.Context, method, objectPath string) (*http.Response, error) {
	hc, err := google.DefaultClient(ctx)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(gcsHTTPBaseURL)
	if err != nil {
	  // This should never happen as base URL is defined by us.
	  panic("invalid GCS base URL: " + err.Error())
	}  
	u.Path = path.Join(u.Path, objectPath)


	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return nil, err
	}

	res, err := hc.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got error while retrieving GCS object, http status: %s, status code: %d", res.Status, res.StatusCode)
	}

	return res, nil
}

func readFileFromGCS(ctx context.Context, objectPath string) ([]byte, error) {
	res, err := gcsRequest(ctx, "GET", objectPath)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func gcsModTime(ctx context.Context, objectPath string) (time.Time, error) {
	res, err := gcsRequest(ctx, "HEAD", objectPath)
	if err != nil {
		return zeroTime, err
	}

	defer res.Body.Close()
	return httpLastModified(res)
}
