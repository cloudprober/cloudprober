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

package oauth

import (
	"time"

	configpb "github.com/cloudprober/cloudprober/common/oauth/proto"
	"github.com/cloudprober/cloudprober/logger"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"
)

func newBearerTokenSource(btc *configpb.BearerToken, refreshExpiryBuffer time.Duration, l *logger.Logger) (oauth2.TokenSource, error) {
	c := &configpb.Config{
		RefreshIntervalSec: proto.Float32(btc.GetRefreshIntervalSec()),
	}
	switch btc.GetSource().(type) {
	case *configpb.BearerToken_File:
		c.Source = &configpb.Config_File{
			File: btc.GetFile(),
		}
	case *configpb.BearerToken_Cmd:
		c.Source = &configpb.Config_Cmd{
			Cmd: btc.GetCmd(),
		}
	case *configpb.BearerToken_GceServiceAccount:
		c.Source = &configpb.Config_GceServiceAccount{
			GceServiceAccount: btc.GetGceServiceAccount(),
		}
	case *configpb.BearerToken_K8SLocalToken:
		c.Source = &configpb.Config_K8SLocalToken{
			K8SLocalToken: btc.GetK8SLocalToken(),
		}
	}

	return newTokenSource(c, refreshExpiryBuffer, l)
}
