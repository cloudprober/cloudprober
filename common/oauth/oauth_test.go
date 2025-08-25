// Copyright 2019 The Cloudprober Authors.
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
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	configpb "github.com/cloudprober/cloudprober/common/oauth/proto"
	"google.golang.org/protobuf/proto"
)

func createTempFile(t *testing.T, b []byte) string {
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
		return ""
	}

	defer tmpfile.Close()
	if _, err := tmpfile.Write(b); err != nil {
		t.Fatal(err)
	}

	return tmpfile.Name()
}

// This is of course a test key with access to nothing.
var testPrivateKey = `-----BEGIN PRIVATE KEY-----
MIICeAIBADANBgkqhkiG9w0BAQEFAASCAmIwggJeAgEAAoGBAN8WnorOX4fqt/1I
prU0eaCw9KiLuv5lSyv2Niq7lkqM2vPlCHv5DLOJJwfhhidzmpOtmR+pt3p/2ecM
ocH2Gnqq3OxfVObA7pEJPrHfSL7v6tEmleDEytfziog2vYwwGcPrIWZ6tPrY5GiI
+1w2+cw34GOwk6EqqTOOLYsuFH5VAgMBAAECgYEAyh+CUZ0drNWbEd7rPC5pLJBn
evXu3GMGMrSG6zy+tJjeIDAY+cnyGhBfzqIknEX/fWHB5JAubsy7rr0hKc1CusPK
4TQbVJLTj0rAlRs65X5QERTxmN93wyru4Ps7AJub/KPI3t98Pj7g6ozgSFUUoCYu
Bm1tFQgopCmH+vJGo1kCQQD7iX3g6nKvqwU70k2bvjAmC+/ef32GA/U7NPk8mXbz
gDfJGFupN5mdt9J6rnI38zt9CM9q5PMLcyuacuLWznljAkEA4wvpFCxH1z/l7zUw
iXRKcIy745zRE62DzFJMqvwDjIM7ECdmaEd0V8eDWG9dgBPqqU+1VYY/CeWoQ9Vn
OJQS5wJAapdtHG69guu6IAuSj7kctkLAt0zXaH8s4JYmOCPgYHepLDKCRUcmdct8
Cjj6dfNA9k9Rdj7nL6byh1TAA78jeQJBANGXxiNkOTGAgC+RZ2wMWUeK80vMEMnc
jOWKN9JD8La+0kA4TvYGuGTr/dkefS7ls+N2fIwl8H2fbvSnxLDbKJMCQQDa7PzK
Y5Zr3MnCwUj2H5mmpTVpj59zj+uf3SJWCx3Fr7TqIHe1O77xkHRo2ZDIUKskwoL/
MZFXww9EJVpAgX2a
-----END PRIVATE KEY-----`

func testJSONKey() string {
	keyTmpl := `{
  "type": "service_account",
  "project_id": "cloud-nat-prober",
  "private_key_id": "testprivateid",
  "private_key": "%s",
  "client_email": "test-consumer@test-project.iam.gserviceaccount.com",
  "client_id": "testclientid",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test-consumer%%40test-project.iam.gserviceaccount.com"
}
`
	return fmt.Sprintf(keyTmpl, strings.Replace(testPrivateKey, "\n", "\\n", -1))
}

func TestGoogleCredentials(t *testing.T) {
	jsonF := createTempFile(t, []byte(testJSONKey()))

	googleC := &configpb.GoogleCredentials{
		JsonFile: proto.String(jsonF),
	}

	c := &configpb.Config{
		Source: &configpb.Config_GoogleCredentials{
			GoogleCredentials: googleC,
		},
	}

	_, err := TokenSourceFromConfig(c, nil)
	if err != nil {
		t.Errorf("Config: %v, Unexpected error: %v", c, err)
	}

	// Set audience, it should fail as jwt_as_access_token is not set.
	googleC.Audience = proto.String("test-audience")

	_, err = TokenSourceFromConfig(c, nil)
	if err == nil {
		t.Errorf("Config: %v, Expected error, but got none.", c)
	}

	// Set jwt_as_access_token, no errors now.
	googleC.JwtAsAccessToken = proto.Bool(true)

	_, err = TokenSourceFromConfig(c, nil)
	if err != nil {
		t.Errorf("Config: %v, Unexpected error: %v", c, err)
	}
}
