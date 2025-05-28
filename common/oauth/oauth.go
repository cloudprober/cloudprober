// Copyright 2019-2025 The Cloudprober Authors.
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

/*
Package oauth implements OAuth related utilities for Cloudprober.
*/
package oauth

import (
	"context"
	"fmt"
	"time"

	configpb "github.com/cloudprober/cloudprober/common/oauth/proto"
	"github.com/cloudprober/cloudprober/internal/file"
	"github.com/cloudprober/cloudprober/logger"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// jsonToken represents OAuth2 token. We use this struct to parse responses
// token URL, command output, or file.
type jsonToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// TokenSourceFromConfig builds a oauth2.TokenSource from the provided config.
func TokenSourceFromConfig(c *configpb.Config, l *logger.Logger) (oauth2.TokenSource, error) {
	// Set default
	refreshExpiryBuffer := time.Minute
	if c.RefreshExpiryBufferSec != nil {
		refreshExpiryBuffer = time.Duration(c.GetRefreshExpiryBufferSec()) * time.Second
	}

	switch c.Source.(type) {
	case *configpb.Config_BearerToken:
		l.Warningf("oauth.TokenSourceFromConfig: BearerToken is deprecated. Move bearer token source config to directly under oauth_config.")
		return newBearerTokenSource(c.GetBearerToken(), refreshExpiryBuffer, l)

	case *configpb.Config_GoogleCredentials:
		f := c.GetGoogleCredentials().GetJsonFile()

		// If JSON file is not provided, try default credentials.
		if f == "" {
			creds, err := google.FindDefaultCredentials(context.Background(), c.GetGoogleCredentials().GetScope()...)
			if err != nil {
				return nil, err
			}
			return creds.TokenSource, nil
		}

		jsonKey, err := file.ReadFile(context.Background(), f)
		if err != nil {
			return nil, fmt.Errorf("error reading Google Credentials file (%s): %v", f, err)
		}

		aud := c.GetGoogleCredentials().GetAudience()
		if aud != "" || c.GetGoogleCredentials().GetJwtAsAccessToken() {
			if !c.GetGoogleCredentials().GetJwtAsAccessToken() {
				return nil, fmt.Errorf("oauth: audience (%s) should only be set if jwt_as_access_token is set to true", aud)
			}
			return google.JWTAccessTokenSourceFromJSON(jsonKey, aud)
		}

		creds, err := google.CredentialsFromJSON(context.Background(), jsonKey, c.GetGoogleCredentials().GetScope()...)
		if err != nil {
			return nil, fmt.Errorf("error parsing Google Credentials file (%s): %v", f, err)
		}
		return creds.TokenSource, nil
	}

	return newTokenSource(c, refreshExpiryBuffer, l)
}
