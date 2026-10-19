// Copyright 2026 The Cloudprober Authors.
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

package sql

import (
	"crypto/tls"
	"net"
	"testing"

	configpb "github.com/cloudprober/cloudprober/probes/sql/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/microsoft/go-mssqldb/msdsn"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

// msProbe is testProbe with the flavor pinned to MSSQL (testProbe only
// defaults Flavor to POSTGRES when it's unset).
func msProbe(t *testing.T, conf *configpb.ProbeConf) *Probe {
	t.Helper()
	conf.Flavor = configpb.ProbeConf_MSSQL.Enum()
	return testProbe(t, conf, nil)
}

func TestMsConnConfig(t *testing.T) {
	tests := []struct {
		name     string
		conf     *configpb.ProbeConf
		target   endpoint.Endpoint
		wantHost string
		wantPort uint64
		wantUser string
		wantPass string
		wantDB   string
	}{
		{
			name: "host from target, rest from connection string",
			conf: &configpb.ProbeConf{
				ConnectionString: proto.String("user=cpuser;password=cppass;database=cpdb;port=1434"),
			},
			target:   endpoint.Endpoint{Name: "dbhost"},
			wantHost: "dbhost",
			wantPort: 1434,
			wantUser: "cpuser",
			wantPass: "cppass",
			wantDB:   "cpdb",
		},
		{
			// A semicolon in the host must not become an extra ADO
			// parameter (e.g. password=injected).
			name: "host with a semicolon is quoted",
			conf: &configpb.ProbeConf{
				ConnectionString: proto.String("user=u;password=realpass;database=db;port=1433"),
			},
			target:   endpoint.Endpoint{Name: "db.example.com;password=injected"},
			wantHost: "db.example.com;password=injected",
			wantPort: 1433,
			wantUser: "u",
			wantPass: "realpass",
			wantDB:   "db",
		},
		{
			name: "fields override connection string",
			conf: &configpb.ProbeConf{
				ConnectionString: proto.String("user=cpuser;password=cppass;database=cpdb;port=1433"),
				User:             proto.String("u2"),
				Password:         proto.String("p2"),
				Database:         proto.String("d2"),
			},
			target:   endpoint.Endpoint{Name: "dbhost"},
			wantHost: "dbhost",
			wantPort: 1433,
			wantUser: "u2",
			wantPass: "p2",
			wantDB:   "d2",
		},
		{
			name: "target port overrides connection string port",
			conf: &configpb.ProbeConf{
				ConnectionString: proto.String("port=1234;user=u;database=db"),
			},
			target:   endpoint.Endpoint{Name: "tgt.host", Port: 9432},
			wantHost: "tgt.host",
			wantPort: 9432,
			wantUser: "u",
			wantDB:   "db",
		},
		{
			name: "substitution in connection string",
			conf: &configpb.ProbeConf{
				ConnectionString: proto.String("port=1234;database=@target.label.db@;user=u"),
			},
			target:   endpoint.Endpoint{Name: "sub.host", Labels: map[string]string{"db": "mydb"}},
			wantHost: "sub.host",
			wantPort: 1234,
			wantUser: "u",
			wantDB:   "mydb",
		},
		{
			name: "dummy target: connection string's host and port used as-is",
			conf: &configpb.ProbeConf{
				ConnectionString: proto.String("sqlserver://cpuser:cppass@dbhost:1433?database=cpdb"),
			},
			target:   endpoint.Endpoint{}, // dummy_targets always resolves to this
			wantHost: "dbhost",
			wantPort: 1433,
			wantUser: "cpuser",
			wantPass: "cppass",
			wantDB:   "cpdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := msProbe(t, tt.conf)
			cfg, err := p.msConnConfig(tt.target)
			if err != nil {
				t.Fatalf("msConnConfig: %v", err)
			}
			assert.Equal(t, tt.wantHost, cfg.Host)
			assert.Equal(t, tt.wantPort, cfg.Port)
			assert.Equal(t, tt.wantUser, cfg.User)
			assert.Equal(t, tt.wantPass, cfg.Password)
			assert.Equal(t, tt.wantDB, cfg.Database)
		})
	}
}

func TestMsConnConfigTLSServerName(t *testing.T) {
	// Explicit tls_config without ServerName: default to the target's name.
	p := msProbe(t, &configpb.ProbeConf{
		ConnectionString: proto.String("user=u"),
	})
	p.tlsConfig = &tls.Config{}
	cfg, err := p.msConnConfig(endpoint.Endpoint{Name: "tgt.host"})
	if err != nil {
		t.Fatalf("msConnConfig: %v", err)
	}
	assert.Equal(t, "tgt.host", cfg.TLSConfig.ServerName)
	assert.Equal(t, msdsn.Encryption(msdsn.EncryptionRequired), cfg.Encryption)
	assert.Empty(t, p.tlsConfig.ServerName, "shared tls config must not be mutated")

	// Without a target (dummy_targets), default to the connection string's
	// host.
	p2 := msProbe(t, &configpb.ProbeConf{
		ConnectionString: proto.String("server=orig.host;user=u"),
	})
	p2.tlsConfig = &tls.Config{}
	cfg, err = p2.msConnConfig(endpoint.Endpoint{})
	if err != nil {
		t.Fatalf("msConnConfig: %v", err)
	}
	assert.Equal(t, "orig.host", cfg.TLSConfig.ServerName)
}

func TestMsConnConfigTargetResolve(t *testing.T) {
	p := msProbe(t, &configpb.ProbeConf{User: proto.String("u")})

	// Targets that come with an IP (e.g. k8s, rds) are dialed by IP; their
	// names are often not resolvable over DNS. Certificate verification must
	// still use the target's name, not the IP msdsn would otherwise default
	// the hostname-in-certificate to.
	cfg, err := p.msConnConfig(endpoint.Endpoint{Name: "pod-1", Port: 1433, IP: net.ParseIP("192.168.9.9")})
	if err != nil {
		t.Fatalf("msConnConfig: %v", err)
	}
	assert.Equal(t, "192.168.9.9", cfg.Host)
	assert.Equal(t, "pod-1", cfg.TLSConfig.ServerName)

	// Out-of-range target port is an error, not a silent wrap.
	_, err = p.msConnConfig(endpoint.Endpoint{Name: "tgt.host", Port: 99999})
	assert.ErrorContains(t, err, "invalid target port")
}

func TestMsConnConfigTargetHostConflict(t *testing.T) {
	tests := []struct {
		name    string
		connStr string
		target  endpoint.Endpoint
		wantErr bool
	}{
		{
			name:    "key/value server with real target: error",
			connStr: "server=orig.host;user=u",
			target:  endpoint.Endpoint{Name: "tgt.host"},
			wantErr: true,
		},
		{
			name:    "URL form with real target: error, even without an explicit port",
			connStr: "sqlserver://u@orig.host/db",
			target:  endpoint.Endpoint{Name: "tgt.host"},
			wantErr: true,
		},
		{
			name:    "URL form case-insensitive scheme with real target: error",
			connStr: "SqlServer://u@orig.host/db",
			target:  endpoint.Endpoint{Name: "tgt.host"},
			wantErr: true,
		},
		{
			// A semicolon inside a quoted value is not a new parameter; the
			// check must respect ADO quoting, not just scan for "server=".
			name:    "server= inside a quoted value with real target: no error",
			connStr: `user=u;password="p;server=x";database=db`,
			target:  endpoint.Endpoint{Name: "tgt.host"},
		},
		{
			name:    "key/value server with dummy target: no error",
			connStr: "server=orig.host;user=u",
			target:  endpoint.Endpoint{},
		},
		{
			name:    "URL form with dummy target: no error",
			connStr: "sqlserver://u@orig.host/db",
			target:  endpoint.Endpoint{},
		},
		{
			name:    "no server, real target: no error",
			connStr: "user=u",
			target:  endpoint.Endpoint{Name: "tgt.host"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := msProbe(t, &configpb.ProbeConf{ConnectionString: proto.String(tt.connStr)})
			_, err := p.msConnConfig(tt.target)
			if tt.wantErr {
				assert.ErrorContains(t, err, "connection_string")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
