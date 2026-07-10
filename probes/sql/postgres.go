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
	"strconv"
	"strings"

	"github.com/cloudprober/cloudprober/common/strtemplate"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// quoteLibpqValue quotes a value for a PostgreSQL key/value connection
// string. Unquoted values are whitespace-delimited, so characters like
// spaces would otherwise be parsed as additional parameters (e.g. a host
// of "h password=x" becoming host=h plus password=x).
func quoteLibpqValue(v string) string {
	escaped := strings.ReplaceAll(v, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return "'" + escaped + "'"
}

// connStringConflictsWithTarget reports whether connStr can't be safely
// combined with a real target's host: URL form always carries a host slot
// (filled or not) that we don't attempt to rewrite, and key/value form may
// already specify one explicitly. Detection uses pgx's own parsing so that
// quoted values (e.g. password='p host=x') don't false-positive: a sentinel
// host prepended to the string survives only if the string doesn't set one
// itself (libpq's last-value-wins rule). pgx doesn't map hostaddr to Host --
// it ends up in RuntimeParams -- so look for it there. A parse error is
// returned as such; connStr itself is what failed to parse.
func connStringConflictsWithTarget(connStr string) (bool, error) {
	s := strings.TrimSpace(connStr)
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
		return true, nil
	}
	// impossible host to detect if the connection string already has a host
	const invalidHost = "cloudprober-internal-host-check.invalid"
	// pgx rejects a trailing space after a lone quoted value
	// (host='x' ); only append connStr when non-empty.
	check := "host=" + quoteLibpqValue(invalidHost)
	if s != "" {
		check += " " + s
	}
	cfg, err := pgx.ParseConfig(check)
	if err != nil {
		return false, err
	}
	return cfg.Host != invalidHost || cfg.RuntimeParams["hostaddr"] != "", nil
}

// pgConnConfig builds the connection config for the POSTGRES flavor.
func (p *Probe) pgConnConfig(target endpoint.Endpoint) (*pgx.ConnConfig, error) {
	if target.Port < 0 || target.Port > 65535 {
		return nil, fmt.Errorf("invalid target port: %d", target.Port)
	}

	connStr, _ := strtemplate.SubstituteLabels(p.c.GetConnectionString(), p.targetLabels(target))

	if target.Name != "" {
		// The parse inside the conflict check also validates connection_string
		// on its own before we append host/port below: a malformed value (e.g.
		// a trailing backslash) would otherwise silently absorb the appended
		// "host=..." during parsing, leaving the probe pointed at the default
		// host instead of the target.
		conflict, err := connStringConflictsWithTarget(connStr)
		if err != nil {
			return nil, fmt.Errorf("parsing connection_string: %v", err)
		}
		if conflict {
			return nil, fmt.Errorf("connection_string must be in key/value form and must not specify a host when the probe has a target; remove the host or use dummy_targets")
		}

		host := target.Name
		// Dial the target's IP if the targets provider supplies one (e.g. k8s,
		// rds). Such target names are often not resolvable over DNS.
		if target.IP != nil {
			// target.Resolve will verify the IP version before using it.
			ip, err := target.Resolve(p.opts.IPVersion, p.opts.Targets)
			if err != nil {
				return nil, fmt.Errorf("resolving target %s: %v", target.Name, err)
			}
			host = ip.String()
		}

		// Appending, rather than parsing-then-overwriting, lets pgx derive
		// Fallbacks and TLS ServerName natively and correctly from the final
		// host, instead of us patching its output after the fact. Quote the
		// host so spaces/special characters cannot inject extra parameters.
		inject := "host=" + quoteLibpqValue(host)
		if target.Port != 0 {
			inject += " port=" + strconv.Itoa(target.Port)
		}
		connStr = strings.TrimSpace(connStr + " " + inject)
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
