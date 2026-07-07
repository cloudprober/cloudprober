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
	gosql "database/sql"
	"fmt"
	"strings"

	"github.com/cloudprober/cloudprober/common/strtemplate"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// pgConnConfig builds the connection config for the POSTGRES flavor:
// libpq defaults and environment variables (PGHOST, PGUSER, PGPASSWORD, ...),
// overridden by connection_string, overridden by user/password/database
// fields, overridden by target's host and port.
func (p *Probe) pgConnConfig(target endpoint.Endpoint) (*pgx.ConnConfig, error) {
	connStr := p.c.GetConnectionString()
	if strings.Contains(connStr, "@") {
		connStr, _ = strtemplate.SubstituteLabels(connStr, p.targetLabels(target))
	}

	cfg, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parsing connection_string: %v", err)
	}

	if v := p.c.GetUser(); v != "" {
		cfg.User = v
	}
	if v := p.c.GetPassword(); v != "" {
		cfg.Password = v
	}
	if v := p.c.GetDatabase(); v != "" {
		cfg.Database = v
	}

	if target.Name != "" {
		if target.Port < 0 || target.Port > 65535 {
			return nil, fmt.Errorf("invalid target port: %d", target.Port)
		}

		host := target.Name
		// Like the TCP probe, dial the target's IP if the targets provider
		// supplies one (e.g. k8s, rds); such target names are often not
		// resolvable over DNS.
		if target.IP != nil {
			ip, err := target.Resolve(p.opts.IPVersion, p.opts.Targets)
			if err != nil {
				return nil, fmt.Errorf("resolving target %s: %v", target.Name, err)
			}
			host = ip.String()
		}

		oldHost := cfg.Host
		cfg.Host = host
		if target.Port != 0 {
			cfg.Port = uint16(target.Port)
		}
		// ParseConfig may have generated fallbacks (e.g. for sslmode=prefer,
		// a plaintext retry) pointing at the pre-override host; redirect them
		// to the target. TLS configs built from the connection string (e.g.
		// sslmode=verify-full) carry the old host as ServerName; certificate
		// verification must run against the target's name instead, even when
		// we dial its IP.
		retargetTLSServerName(cfg.TLSConfig, oldHost, target.Name)
		for _, fb := range cfg.Fallbacks {
			fb.Host, fb.Port = cfg.Host, cfg.Port
			retargetTLSServerName(fb.TLSConfig, oldHost, target.Name)
		}
	}

	// Explicit tls_config replaces connection string's TLS parameters,
	// including any non-TLS fallback that sslmode=prefer would allow.
	if p.tlsConfig != nil {
		// Clone: the shared config would otherwise be mutated per target, and
		// pgconn uses the config as-is — an empty ServerName fails the TLS
		// handshake, so default it to the host we verify against.
		tlsCfg := p.tlsConfig.Clone()
		if tlsCfg.ServerName == "" {
			if target.Name != "" {
				tlsCfg.ServerName = target.Name
			} else {
				tlsCfg.ServerName = cfg.Host
			}
		}
		cfg.TLSConfig = tlsCfg
		cfg.Fallbacks = nil
	}

	return cfg, nil
}

// retargetTLSServerName updates a TLS config that pgx.ParseConfig built for
// the connection string's host when the connection is redirected to a target.
func retargetTLSServerName(tlsCfg *tls.Config, oldHost, newName string) {
	if tlsCfg != nil && tlsCfg.ServerName == oldHost {
		tlsCfg.ServerName = newName
	}
}

func (p *Probe) pgOpenDB(target endpoint.Endpoint) (*gosql.DB, error) {
	cfg, err := p.pgConnConfig(target)
	if err != nil {
		return nil, err
	}
	return gosql.OpenDB(stdlib.GetConnector(*cfg)), nil
}
