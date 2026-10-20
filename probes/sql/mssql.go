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
	"strconv"
	"strings"

	"github.com/cloudprober/cloudprober/common/strtemplate"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	mssql "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/msdsn"
)

// quoteADOValue quotes a value for a SQL Server ADO-style (semicolon-
// delimited key=value) connection string. Unquoted values are terminated
// by the next semicolon, so a host like "evil;password=hunter2" would
// otherwise be parsed as two parameters instead of one value.
func quoteADOValue(v string) string {
	return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
}

// connStringConflictsWithTargetMSSQL mirrors connStringConflictsWithTarget
// (postgres.go) for the ADO connection-string format: a "server" key, or
// URL form (which always carries a host slot, filled or not), can't be
// safely combined with a real target's host. Detection parses the ADO
// string with a sentinel "server" prepended via msdsn's own parser, same
// trick connStringConflictsWithTarget uses with pgx: the sentinel survives
// only if connStr doesn't set "server" itself (ADO's last-value-wins
// rule). The URL form is checked separately since prepending a sentinel
// key/value pair in front of it would not parse as a URL at all.
func connStringConflictsWithTargetMSSQL(connStr string) (bool, error) {
	s := strings.TrimSpace(connStr)
	if strings.HasPrefix(strings.ToLower(s), "sqlserver://") {
		return true, nil
	}

	const invalidHost = "cloudprober-internal-host-check.invalid"
	check := "server=" + quoteADOValue(invalidHost)
	if s != "" {
		check += ";" + s
	}
	cfg, err := msdsn.Parse(check)
	if err != nil {
		return false, err
	}
	return cfg.Host != invalidHost, nil
}

// msConnConfig builds the connection config for the MSSQL flavor.
func (p *Probe) msConnConfig(target endpoint.Endpoint) (*msdsn.Config, error) {
	if target.Port < 0 || target.Port > 65535 {
		return nil, fmt.Errorf("invalid target port: %d", target.Port)
	}

	connStr, _ := strtemplate.SubstituteLabels(p.c.GetConnectionString(), p.targetLabels(target))

	if target.Name != "" {
		conflict, err := connStringConflictsWithTargetMSSQL(connStr)
		if err != nil {
			return nil, fmt.Errorf("parsing connection_string: %v", err)
		}
		if conflict {
			return nil, fmt.Errorf("connection_string must be in key/value form and must not specify a server when the probe has a target; remove the server or use dummy_targets")
		}

		host := target.Name
		// Dial the target's IP if the targets provider supplies one (e.g.
		// k8s, rds). Such target names are often not resolvable over DNS.
		if target.IP != nil {
			// target.Resolve will verify the IP version before using it.
			ip, err := target.Resolve(p.opts.IPVersion, p.opts.Targets)
			if err != nil {
				return nil, fmt.Errorf("resolving target %s: %v", target.Name, err)
			}
			host = ip.String()
		}

		// Appending, rather than parsing-then-overwriting, lets msdsn derive
		// the TLS hostname-in-certificate default natively from the final
		// host. Quote the host so semicolons/quotes cannot inject extra
		// parameters.
		inject := "server=" + quoteADOValue(host)
		if target.Port != 0 {
			inject += ";port=" + strconv.Itoa(target.Port)
		}
		if connStr != "" {
			connStr += ";"
		}
		connStr += inject
	}

	cfg, err := msdsn.Parse(connStr)
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

	// Explicit tls_config replaces connection string's TLS parameters and
	// forces a fully encrypted session, rather than the default's
	// login-packet-only "optional" encryption.
	if p.tlsConfig != nil {
		// Clone: the shared config would otherwise be mutated per target.
		tlsCfg := p.tlsConfig.Clone()
		if tlsCfg.ServerName == "" {
			if target.Name != "" {
				tlsCfg.ServerName = target.Name
			} else {
				tlsCfg.ServerName = cfg.Host
			}
		}
		cfg.TLSConfig = tlsCfg
		cfg.Encryption = msdsn.EncryptionRequired
	} else if target.IP != nil && cfg.TLSConfig != nil {
		// We dialed the target's IP above, so msdsn derived the
		// hostname-in-certificate default from the IP too; certificate
		// verification must run against the target's name instead.
		cfg.TLSConfig.ServerName = target.Name
	}

	return &cfg, nil
}

func (p *Probe) msOpenDB(target endpoint.Endpoint) (*gosql.DB, error) {
	cfg, err := p.msConnConfig(target)
	if err != nil {
		return nil, err
	}
	return gosql.OpenDB(mssql.NewConnectorConfig(*cfg)), nil
}
