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
	"regexp"
	"strconv"
	"strings"

	"github.com/cloudprober/cloudprober/common/strtemplate"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// hostKeywordRe matches an explicit host=/hostaddr= keyword in a key/value
// connection string.
var hostKeywordRe = regexp.MustCompile(`(?:^|\s)(host|hostaddr)\s*=\s*\S`)

// connStringConflictsWithTarget reports whether connStr can't be safely
// combined with a real target's host: URL form always carries a host slot
// (filled or not) that we don't attempt to rewrite, and key/value form may
// already specify one explicitly.
func connStringConflictsWithTarget(connStr string) bool {
	s := strings.TrimSpace(connStr)
	if strings.HasPrefix(s, "postgres://") || strings.HasPrefix(s, "postgresql://") {
		return true
	}
	return hostKeywordRe.MatchString(s)
}

// pgConnConfig builds the connection config for the POSTGRES flavor. With a
// real target, host and port come from it, like the tcp, http, and grpc
// probes -- connection_string must be in key/value form and must not specify
// a host itself (parsing/appending "host=..." works reliably for key/value
// form, per libpq's own last-value-wins rule; not for URL form, so URL form
// isn't supported here). With dummy_targets (target.Name == "", the only
// target provider that produces that), there's no target to take a host/port
// from, so connection_string's (or the flavor's environment variables, e.g.
// PGHOST) are used as-is, in any form. Either way, connection_string,
// overridden by user/password/database fields, configures everything else.
func (p *Probe) pgConnConfig(target endpoint.Endpoint) (*pgx.ConnConfig, error) {
	if target.Port < 0 || target.Port > 65535 {
		return nil, fmt.Errorf("invalid target port: %d", target.Port)
	}

	connStr := p.c.GetConnectionString()
	if strings.Contains(connStr, "@") {
		connStr, _ = strtemplate.SubstituteLabels(connStr, p.targetLabels(target))
	}

	if target.Name != "" {
		if connStringConflictsWithTarget(connStr) {
			return nil, fmt.Errorf("connection_string must be in key/value form and must not specify a host when the probe has a target; remove the host or use dummy_targets")
		}

		// Validate connection_string on its own before appending host/port
		// below. A malformed value (e.g. a trailing backslash) would
		// otherwise silently absorb the appended "host=..." during parsing,
		// leaving the probe pointed at the default host instead of the target.
		if _, err := pgx.ParseConfig(connStr); err != nil {
			return nil, fmt.Errorf("parsing connection_string: %v", err)
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

		// Appending, rather than parsing-then-overwriting, lets pgx derive
		// Fallbacks and TLS ServerName natively and correctly from the final
		// host, instead of us patching its output after the fact.
		inject := "host=" + host
		if target.Port != 0 {
			inject += " port=" + strconv.Itoa(target.Port)
		}
		if connStr == "" {
			connStr = inject
		} else {
			connStr = connStr + " " + inject
		}
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
	} else if target.IP != nil {
		// We dialed the target's IP above, so pgx derived TLS ServerName from
		// the IP too; certificate verification must run against the target's
		// name instead.
		setServerName(cfg.TLSConfig, target.Name)
		for _, fb := range cfg.Fallbacks {
			setServerName(fb.TLSConfig, target.Name)
		}
	}

	return cfg, nil
}

func setServerName(tlsCfg *tls.Config, name string) {
	if tlsCfg != nil {
		tlsCfg.ServerName = name
	}
}

func (p *Probe) pgOpenDB(target endpoint.Endpoint) (*gosql.DB, error) {
	cfg, err := p.pgConnConfig(target)
	if err != nil {
		return nil, err
	}
	return gosql.OpenDB(stdlib.GetConnector(*cfg)), nil
}
