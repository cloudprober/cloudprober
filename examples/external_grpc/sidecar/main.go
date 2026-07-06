// Binary sidecar is an example cloudprober probe sidecar. It serves two
// probe types over the EXTERNAL_GRPC contract:
//
//   - "http": stateful — keeps a per-target HTTP client (connection pool)
//     across probe cycles, fetches a URL, and returns extra metrics.
//   - "tcp": stateless — just dials the target.
//
// Run it, then point cloudprober at it with the config in the parent
// directory:
//
//	go run ./sidecar
//	cloudprober --config_file=cloudprober.cfg
package main

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudprober/cloudprober/sidecar"
)

type httpConfig struct {
	Scheme string `json:"scheme"` // default: https
	Port   int    `json:"port"`   // default: target port, or scheme default
	Path   string `json:"path"`   // default: /
}

var httpProbe = sidecar.ProbeType[httpConfig, *http.Client]{
	// New runs once per target; the returned client (and its connection
	// pool) is cached across probe cycles via the state_handle session
	// mechanism.
	New: func(ctx context.Context, t sidecar.Target, c httpConfig) (*http.Client, error) {
		log.Printf("http: creating client for target %s", t.Name)
		return &http.Client{}, nil
	},
	Probe: func(ctx context.Context, t sidecar.Target, c httpConfig, client *http.Client) *sidecar.Result {
		scheme := c.Scheme
		if scheme == "" {
			scheme = "https"
		}
		host := t.Name
		if port := cmp.Or(c.Port, t.Port); port != 0 {
			host = net.JoinHostPort(host, strconv.Itoa(port))
		}
		path := c.Path
		if path == "" {
			path = "/"
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, scheme+"://"+host+path, nil)
		if err != nil {
			return sidecar.Internal(err)
		}

		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			return sidecar.Fail(err)
		}
		defer resp.Body.Close()
		respBytes, _ := io.Copy(io.Discard, resp.Body)
		latency := time.Since(start)

		if resp.StatusCode >= 400 {
			return sidecar.Fail(fmt.Errorf("bad response status: %s", resp.Status))
		}
		return sidecar.OK(latency).
			Metric("resp_bytes", respBytes, "status", strconv.Itoa(resp.StatusCode))
	},
	Close: func(client *http.Client) { client.CloseIdleConnections() },
}

type tcpConfig struct {
	Port int `json:"port"` // default: target port
}

var tcpProbe = sidecar.ProbeType[tcpConfig, any]{
	Probe: func(ctx context.Context, t sidecar.Target, c tcpConfig, _ any) *sidecar.Result {
		port := cmp.Or(c.Port, t.Port)
		if port == 0 {
			return sidecar.Internal(errors.New("no port in config or target"))
		}
		host := t.Name
		if t.IP != "" {
			host = t.IP
		}

		var d net.Dialer
		start := time.Now()
		conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			return sidecar.Fail(err)
		}
		conn.Close()
		return sidecar.OK(time.Since(start))
	},
}

func main() {
	addr := flag.String("addr", "unix:///tmp/cloudprober-sidecar.sock", "Address to listen on: unix:///path/to/socket or a TCP address like :9314")
	flag.Parse()

	log.Fatal(sidecar.Serve(
		sidecar.Listen(*addr),
		sidecar.Register("http", httpProbe),
		sidecar.Register("tcp", tcpProbe),
	))
}
