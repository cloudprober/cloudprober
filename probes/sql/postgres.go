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
		cfg.Host = target.Name
		if target.Port != 0 {
			cfg.Port = uint16(target.Port)
		}
		// ParseConfig may have generated fallbacks (e.g. for sslmode=prefer,
		// a plaintext retry) pointing at the pre-override host; redirect them
		// to the target.
		for _, fb := range cfg.Fallbacks {
			fb.Host, fb.Port = cfg.Host, cfg.Port
		}
	}

	// Explicit tls_config replaces connection string's TLS parameters,
	// including any non-TLS fallback that sslmode=prefer would allow.
	if p.tlsConfig != nil {
		cfg.TLSConfig = p.tlsConfig
		cfg.Fallbacks = nil
	}

	return cfg, nil
}

func (p *Probe) pgOpenDB(target endpoint.Endpoint) (*gosql.DB, error) {
	cfg, err := p.pgConnConfig(target)
	if err != nil {
		return nil, err
	}
	return gosql.OpenDB(stdlib.GetConnector(*cfg)), nil
}
