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

package tlsconfig

// openssl ecparam -name prime256v1 -genkey -noout -out private-key.pem
// openssl req -new -x509 -key private-key.pem -out cert.pem -days 360
// US, CA, CN: cert1.cloudprober.org
const cert1PEM = `
-----BEGIN CERTIFICATE-----
MIIB4DCCAYYCCQC545iBOVHSTDAKBggqhkjOPQQDAjB4MQswCQYDVQQGEwJVUzEL
MAkGA1UECAwCQ0ExGTAXBgNVBAoMEENsb3VkcHJvYmVyLCBJbmMxHjAcBgNVBAMM
FWNlcnQxLmNsb3VkcHJvYmVyLm9yZzEhMB8GCSqGSIb3DQEJARYSbWFudWdhcmdA
Z21haWwuY29tMB4XDTIzMDgwMTA3MDY1MloXDTI0MDcyNjA3MDY1MloweDELMAkG
A1UEBhMCVVMxCzAJBgNVBAgMAkNBMRkwFwYDVQQKDBBDbG91ZHByb2JlciwgSW5j
MR4wHAYDVQQDDBVjZXJ0MS5jbG91ZHByb2Jlci5vcmcxITAfBgkqhkiG9w0BCQEW
Em1hbnVnYXJnQGdtYWlsLmNvbTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABIVh
Cmj1dRyMV2yKigJO1FJvpR95N03ngXEZAdPXRt6tR1G7kTEOoE+mRbO5CGpCfofO
9gOMZrsUuwfSGJLzqJIwCgYIKoZIzj0EAwIDSAAwRQIgIdCFppeeoheycfEIa//a
1FJkYpRYmbRoRBchtcOIYy4CIQDZtItSV4+MIxzdsg2W4QTP8coAit6t/05j/sIr
fgQNtA==
-----END CERTIFICATE-----
`

const cert1Key = `
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEILmuVz5jU6rlOpkysldvpgPeqTDWk7PUFXricdXEHOvJoAoGCCqGSM49
AwEHoUQDQgAEhWEKaPV1HIxXbIqKAk7UUm+lH3k3TeeBcRkB09dG3q1HUbuRMQ6g
T6ZFs7kIakJ+h872A4xmuxS7B9IYkvOokg==
-----END EC PRIVATE KEY-----
`

// openssl ecparam -name prime256v1 -genkey -noout -out private-key.pem
// openssl req -new -x509 -key private-key.pem -out cert.pem -days 360
// US, CA, CN: cert2.cloudprober.org
const cert2PEM = `
-----BEGIN CERTIFICATE-----
MIIB4TCCAYYCCQCkcSPHZI1mwzAKBggqhkjOPQQDAjB4MQswCQYDVQQGEwJVUzEL
MAkGA1UECAwCQ0ExGTAXBgNVBAoMEENsb3VkcHJvYmVyLCBJbmMxHjAcBgNVBAMM
FWNlcnQyLmNsb3VkcHJvYmVyLm9yZzEhMB8GCSqGSIb3DQEJARYSbWFudWdhcmdA
Z21haWwuY29tMB4XDTIzMDgwMTA3MDM1MVoXDTI0MDcyNjA3MDM1MVoweDELMAkG
A1UEBhMCVVMxCzAJBgNVBAgMAkNBMRkwFwYDVQQKDBBDbG91ZHByb2JlciwgSW5j
MR4wHAYDVQQDDBVjZXJ0Mi5jbG91ZHByb2Jlci5vcmcxITAfBgkqhkiG9w0BCQEW
Em1hbnVnYXJnQGdtYWlsLmNvbTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABAsM
kgTI0ZmdU8MIcZpaz4nwDB8yKlLwpraLKeGi9YsYTYquOo6oXnSfIdAYxJ6ebpYb
LuFU3s7oB76gHQ6gJ2EwCgYIKoZIzj0EAwIDSQAwRgIhAJW80Y2b4DtUyCL8lwR4
aTffkfb8Uo4TuE6xTYkiiSYpAiEA6Cz4k2wW4/zSaK5eFpMMFOdeCBNk4DbWEvXc
3w1Upw8=
-----END CERTIFICATE-----
`

const cert2Key = `
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIOGMrTkEh7cxGpLQnq/ZuNGiCDCjnY8JUfb83ezMgzWaoAoGCCqGSM49
AwEHoUQDQgAECwySBMjRmZ1TwwhxmlrPifAMHzIqUvCmtosp4aL1ixhNiq46jqhe
dJ8h0BjEnp5ulhsu4VTezugHvqAdDqAnYQ==
-----END EC PRIVATE KEY-----
`
