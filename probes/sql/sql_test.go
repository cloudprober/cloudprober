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
	"context"
	"crypto/tls"
	gosql "database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/internal/validators"
	validatorpb "github.com/cloudprober/cloudprober/internal/validators/proto"
	"github.com/cloudprober/cloudprober/metrics"
	payloadconfigpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/sql/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

// fakeConnector and fakeConn implement just enough of database/sql/driver to
// exercise the probe without a real database.
type fakeConnector struct {
	conn *fakeConn
}

func (c *fakeConnector) Connect(context.Context) (driver.Conn, error) { return c.conn, nil }
func (c *fakeConnector) Driver() driver.Driver                        { return nil }

type fakeConn struct {
	pingErr  error
	queryErr error
	cols     []string
	rows     [][]driver.Value
}

func (c *fakeConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (c *fakeConn) Close() error                   { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)      { return nil, errors.New("not implemented") }
func (c *fakeConn) Ping(ctx context.Context) error { return c.pingErr }

func (c *fakeConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.queryErr != nil {
		return nil, c.queryErr
	}
	return &fakeRows{cols: c.cols, rows: c.rows}, nil
}

type fakeRows struct {
	cols []string
	rows [][]driver.Value
	i    int
}

func (r *fakeRows) Columns() []string { return r.cols }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.i >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.i])
	r.i++
	return nil
}

func testProbe(t *testing.T, conf *configpb.ProbeConf, conn *fakeConn) *Probe {
	t.Helper()

	if conf.Flavor == nil {
		conf.Flavor = configpb.ProbeConf_POSTGRES.Enum()
	}
	opts := options.DefaultOptions()
	opts.ProbeConf = conf

	p := &Probe{}
	if err := p.Init("sql_test", opts); err != nil {
		t.Fatalf("Error initializing probe: %v", err)
	}
	if conn != nil {
		p.openDB = func(target endpoint.Endpoint) (*gosql.DB, error) {
			return gosql.OpenDB(&fakeConnector{conn: conn}), nil
		}
	}
	return p
}

func TestInit(t *testing.T) {
	queryFile := filepath.Join(t.TempDir(), "query.sql")
	if err := os.WriteFile(queryFile, []byte("SELECT 1"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		conf      *configpb.ProbeConf
		wantErr   string
		wantQuery string
	}{
		{
			name:    "no flavor",
			conf:    &configpb.ProbeConf{},
			wantErr: "flavor",
		},
		{
			name: "query and query_file both set",
			conf: &configpb.ProbeConf{
				Flavor:    configpb.ProbeConf_POSTGRES.Enum(),
				Query:     proto.String("SELECT 1"),
				QueryFile: proto.String(queryFile),
			},
			wantErr: "only one of query and query_file",
		},
		{
			name: "bad query_file",
			conf: &configpb.ProbeConf{
				Flavor:    configpb.ProbeConf_POSTGRES.Enum(),
				QueryFile: proto.String(filepath.Join(t.TempDir(), "nonexistent.sql")),
			},
			wantErr: "reading query_file",
		},
		{
			name: "query from file",
			conf: &configpb.ProbeConf{
				Flavor:    configpb.ProbeConf_POSTGRES.Enum(),
				QueryFile: proto.String(queryFile),
			},
			wantQuery: "SELECT 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := options.DefaultOptions()
			opts.ProbeConf = tt.conf

			p := &Probe{}
			err := p.Init("sql_test", opts)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantQuery, p.query)
		})
	}
}

func neutralizePGEnv(t *testing.T) {
	t.Helper()
	// pgconn ignores empty environment variables, so setting these to ""
	// makes the test hermetic even if the host has them set.
	for _, k := range []string{"PGHOST", "PGPORT", "PGDATABASE", "PGUSER", "PGPASSWORD", "PGSSLMODE", "PGSERVICE", "PGSERVICEFILE"} {
		t.Setenv(k, "")
	}
}

func TestPgConnConfig(t *testing.T) {
	neutralizePGEnv(t)

	tests := []struct {
		name     string
		conf     *configpb.ProbeConf
		target   endpoint.Endpoint
		wantHost string
		wantPort uint16
		wantUser string
		wantPass string
		wantDB   string
	}{
		{
			name: "connection string only",
			conf: &configpb.ProbeConf{
				ConnectionString: proto.String("postgres://cpuser:cppass@dbhost:5433/cpdb?sslmode=disable"),
			},
			wantHost: "dbhost",
			wantPort: 5433,
			wantUser: "cpuser",
			wantPass: "cppass",
			wantDB:   "cpdb",
		},
		{
			name: "fields override connection string",
			conf: &configpb.ProbeConf{
				ConnectionString: proto.String("postgres://cpuser:cppass@dbhost:5433/cpdb?sslmode=disable"),
				User:             proto.String("u2"),
				Password:         proto.String("p2"),
				Database:         proto.String("d2"),
			},
			wantHost: "dbhost",
			wantPort: 5433,
			wantUser: "u2",
			wantPass: "p2",
			wantDB:   "d2",
		},
		{
			name: "target overrides host and port",
			conf: &configpb.ProbeConf{
				ConnectionString: proto.String("host=orig.host port=5432 user=u dbname=db sslmode=disable"),
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
				ConnectionString: proto.String("host=ignored.host port=1234 dbname=@target.label.db@ user=u sslmode=disable"),
			},
			target:   endpoint.Endpoint{Name: "sub.host", Labels: map[string]string{"db": "mydb"}},
			wantHost: "sub.host",
			wantPort: 1234,
			wantUser: "u",
			wantDB:   "mydb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testProbe(t, tt.conf, nil)
			cfg, err := p.pgConnConfig(tt.target)
			if err != nil {
				t.Fatalf("pgConnConfig: %v", err)
			}
			assert.Equal(t, tt.wantHost, cfg.Host)
			assert.Equal(t, tt.wantPort, cfg.Port)
			assert.Equal(t, tt.wantUser, cfg.User)
			assert.Equal(t, tt.wantPass, cfg.Password)
			assert.Equal(t, tt.wantDB, cfg.Database)
			for _, fb := range cfg.Fallbacks {
				assert.Equal(t, tt.wantHost, fb.Host)
				assert.Equal(t, tt.wantPort, fb.Port)
			}
		})
	}
}

func TestPgConnConfigFallbacks(t *testing.T) {
	neutralizePGEnv(t)

	// sslmode=prefer (the default) generates a plaintext fallback; target
	// override should redirect it.
	p := testProbe(t, &configpb.ProbeConf{
		ConnectionString: proto.String("host=orig.host user=u sslmode=prefer"),
	}, nil)
	cfg, err := p.pgConnConfig(endpoint.Endpoint{Name: "tgt.host", Port: 9432})
	if err != nil {
		t.Fatalf("pgConnConfig: %v", err)
	}
	if assert.NotEmpty(t, cfg.Fallbacks, "expected a fallback for sslmode=prefer") {
		for _, fb := range cfg.Fallbacks {
			assert.Equal(t, "tgt.host", fb.Host)
			assert.Equal(t, uint16(9432), fb.Port)
		}
	}

	// Explicit TLS config replaces connection string's TLS parameters and
	// drops the plaintext fallback.
	p = testProbe(t, &configpb.ProbeConf{
		ConnectionString: proto.String("host=orig.host user=u sslmode=prefer"),
	}, nil)
	p.tlsConfig = &tls.Config{ServerName: "tls.host"}
	cfg, err = p.pgConnConfig(endpoint.Endpoint{})
	if err != nil {
		t.Fatalf("pgConnConfig: %v", err)
	}
	assert.Equal(t, p.tlsConfig, cfg.TLSConfig)
	assert.Empty(t, cfg.Fallbacks)
}

func TestRunProbe(t *testing.T) {
	queryConf := func() *configpb.ProbeConf {
		return &configpb.ProbeConf{
			Query: proto.String("SELECT 'db_users', count(*) FROM users"),
		}
	}
	queryRows := &fakeConn{
		cols: []string{"name", "value"},
		rows: [][]driver.Value{{"db_users", int64(42)}},
	}

	tests := []struct {
		name           string
		conf           *configpb.ProbeConf
		conn           *fakeConn
		validatorRegex string
		wantSuccess    int64
		wantPayload    string // metric expected in payload metrics
	}{
		{
			name:        "ping success",
			conf:        &configpb.ProbeConf{},
			conn:        &fakeConn{},
			wantSuccess: 1,
		},
		{
			name:        "ping failure",
			conf:        &configpb.ProbeConf{},
			conn:        &fakeConn{pingErr: errors.New("connection refused")},
			wantSuccess: 0,
		},
		{
			name:        "query success",
			conf:        queryConf(),
			conn:        queryRows,
			wantSuccess: 1,
		},
		{
			name:        "query failure",
			conf:        queryConf(),
			conn:        &fakeConn{queryErr: errors.New("relation does not exist")},
			wantSuccess: 0,
		},
		{
			name:           "validation success",
			conf:           queryConf(),
			conn:           queryRows,
			validatorRegex: "db_users 42",
			wantSuccess:    1,
		},
		{
			name:           "validation failure",
			conf:           queryConf(),
			conn:           queryRows,
			validatorRegex: "db_users 43",
			wantSuccess:    0,
		},
		{
			name: "payload metrics",
			conf: func() *configpb.ProbeConf {
				c := queryConf()
				c.ResponseMetricsOptions = &payloadconfigpb.OutputMetricsOptions{}
				return c
			}(),
			conn:        queryRows,
			wantSuccess: 1,
			wantPayload: "db_users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testProbe(t, tt.conf, tt.conn)

			if tt.validatorRegex != "" {
				vs, err := validators.Init([]*validatorpb.Validator{{
					Name: "regex",
					Type: &validatorpb.Validator_Regex{Regex: tt.validatorRegex},
				}})
				if err != nil {
					t.Fatalf("validators.Init: %v", err)
				}
				p.opts.Validators = vs
			}

			runReq := &sched.RunProbeForTargetRequest{
				Target:  endpoint.Endpoint{Name: "db1", Port: 5432},
				LastRun: &sched.LastRunResult{},
			}
			p.runProbe(context.Background(), runReq)

			result := runReq.Result.(*probeResult)
			assert.Equal(t, int64(1), result.total, "total")
			assert.Equal(t, tt.wantSuccess, result.success, "success")
			assert.Equal(t, tt.wantSuccess == 1, runReq.LastRun.Success, "LastRun.Success")

			if tt.wantPayload != "" {
				if assert.NotEmpty(t, result.payloadMetrics, "expected payload metrics") {
					assert.NotNil(t, result.payloadMetrics[0].Metric(tt.wantPayload), "expected metric %q in %v", tt.wantPayload, result.payloadMetrics[0])
				}
			}

			// Metrics() should include standard metrics and any payload
			// metrics, and reset the payload metrics.
			ems := result.Metrics(time.Now(), 0, p.opts)
			assert.Equal(t, metrics.NewInt(1), ems[0].Metric("total"))
			if tt.wantPayload != "" {
				assert.Len(t, ems, 2)
				assert.Nil(t, result.payloadMetrics)
			}
		})
	}
}

func TestSerializeRows(t *testing.T) {
	conn := &fakeConn{
		cols: []string{"a", "b", "c"},
		rows: [][]driver.Value{
			{"row1", int64(1), []byte("bytes")},
			{"row2", 2.5, nil},
		},
	}
	db := gosql.OpenDB(&fakeConnector{conn: conn})
	defer db.Close()

	rows, err := db.QueryContext(context.Background(), "SELECT ...")
	if err != nil {
		t.Fatalf("QueryContext: %v", err)
	}
	defer rows.Close()

	b, err := serializeRows(rows)
	if err != nil {
		t.Fatalf("serializeRows: %v", err)
	}
	assert.Equal(t, "row1 1 bytes\nrow2 2.5 <nil>\n", string(b))
}
