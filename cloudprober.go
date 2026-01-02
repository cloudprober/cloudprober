// Copyright 2017-2019 The Cloudprober Authors.
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

/*
Package cloudprober provides a prober for running a set of probes.

Cloudprober takes in a config proto which dictates what probes should be created
with what configuration, and manages the asynchronous fan-in/fan-out of the
metrics data from these probes.
*/
package cloudprober

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/cloudprober/cloudprober/common/tlsconfig"
	"github.com/cloudprober/cloudprober/config"
	configpb "github.com/cloudprober/cloudprober/config/proto"
	"github.com/cloudprober/cloudprober/internal/servers"
	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/internal/configwatcher"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics/singlerun"
	"github.com/cloudprober/cloudprober/prober"
	"github.com/cloudprober/cloudprober/probes"
	"github.com/cloudprober/cloudprober/state"
	"github.com/cloudprober/cloudprober/surfacers"
	"github.com/cloudprober/cloudprober/web"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

const (
	sysvarsModuleName = "sysvars"
)

// Constants defining the default server host and port.
const (
	DefaultServerHost   = ""
	DefaultServerPort   = 9313
	ServerHostEnvVar    = "CLOUDPROBER_HOST"
	ServerPortEnvVar    = "CLOUDPROBER_PORT"
	DisableHTTPDebugVar = "CLOUDPROBER_DISABLE_HTTP_PPROF"
)

// Global prober.Prober instance protected by a mutex.
var cloudProber struct {
	prober          *prober.Prober
	defaultServerLn net.Listener
	defaultGRPCLn   net.Listener
	configSource    config.ConfigSource
	config          *configpb.ProberConfig
	cancelInitCtx   context.CancelFunc
	sync.RWMutex
}

func getServerHost(c *configpb.ProberConfig) string {
	serverHost := c.GetHost()
	if serverHost == "" {
		serverHost = DefaultServerHost
		// If ServerHostEnvVar is defined, it will override the default
		// server host.
		if host := os.Getenv(ServerHostEnvVar); host != "" {
			serverHost = host
		}
	}
	return serverHost
}

func getDefaultServerPort(c *configpb.ProberConfig, l *logger.Logger) (int, error) {
	if c.GetPort() != 0 {
		return int(c.GetPort()), nil
	}

	// If ServerPortEnvVar is defined, it will override the default
	// server port.
	portStr := os.Getenv(ServerPortEnvVar)
	if portStr == "" {
		return DefaultServerPort, nil
	}

	if strings.HasPrefix(portStr, "tcp://") {
		l.Warningf("%s environment variable likely set by Kubernetes (to %s), ignoring it", ServerPortEnvVar, portStr)
		return DefaultServerPort, nil
	}

	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse default port from the env var: %s=%s", ServerPortEnvVar, portStr)
	}

	return int(port), nil
}

func initDefaultServer(c *configpb.ProberConfig, l *logger.Logger) (net.Listener, error) {
	serverHost := getServerHost(c)
	serverPort, err := getDefaultServerPort(c, l)

	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(serverHost, strconv.Itoa(serverPort)))
	if err != nil {
		return nil, fmt.Errorf("error while creating listener for default HTTP server: %v", err)
	}

	return ln, nil
}

func setDebugHandlers(srvMux *http.ServeMux) {
	if os.Getenv(DisableHTTPDebugVar) != "" {
		return
	}
	srvMux.HandleFunc("/debug/pprof/", pprof.Index)
	srvMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	srvMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	srvMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	srvMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

// InitFromConfig initializes Cloudprober using the provided config.
// Deprecated: This function is kept only for compatibility reasons. It's
// recommended to use Init() or InitWithConfigSource() instead.
func InitFromConfig(configFile string) error {
	return InitWithConfigSource(config.ConfigSourceWithFile(configFile))
}

// Init initializes Cloudprober using the default config source.
func Init() error {
	return InitWithConfigSource(config.DefaultConfigSource())
}

func InitWithConfigSource(configSrc config.ConfigSource) error {
	if err := initWithConfigSource(configSrc); err != nil {
		return err
	}
	return web.InitWithDataFuncs(web.DataFuncs{
		GetRawConfig:    GetRawConfig,
		GetParsedConfig: GetParsedConfig,
		GetInfo:         GetInfo,
	})
}

func initWithConfigSource(configSrc config.ConfigSource) error {
	// Return immediately if prober is already initialized.
	cloudProber.Lock()
	defer cloudProber.Unlock()

	if cloudProber.prober != nil {
		return nil
	}

	// Initialize sysvars module
	if err := sysvars.Init(logger.NewWithAttrs(slog.String("component", sysvarsModuleName)), nil); err != nil {
		return err
	}

	cfg, err := configSrc.GetConfig()
	if err != nil {
		return err
	}

	globalLogger := logger.NewWithAttrs(slog.String("component", "global"))

	// Start default HTTP server. It's used for profile handlers and
	// prometheus exporter.
	ln, err := initDefaultServer(cfg, globalLogger)
	if err != nil {
		return err
	}
	srvMux := http.NewServeMux()
	setDebugHandlers(srvMux)
	state.SetDefaultHTTPServeMux(srvMux)

	var grpcLn net.Listener
	if cfg.GetGrpcPort() != 0 {
		serverHost := getServerHost(cfg)

		grpcLn, err = net.Listen("tcp", net.JoinHostPort(serverHost, strconv.Itoa(int(cfg.GetGrpcPort()))))
		if err != nil {
			return fmt.Errorf("error while creating listener for default gRPC server: %v", err)
		}

		// Create the default gRPC server now, so that other modules can register
		// their services with it in the prober.Init() phase.
		var serverOpts []grpc.ServerOption

		if cfg.GetGrpcTlsConfig() != nil {
			tlsConfig := &tls.Config{}
			if err := tlsconfig.UpdateTLSConfig(tlsConfig, cfg.GetGrpcTlsConfig()); err != nil {
				return err
			}
			tlsConfig.ClientCAs = tlsConfig.RootCAs
			serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsConfig)))
		}

		s := grpc.NewServer(serverOpts...)
		reflection.Register(s)
		// register channelz service to the default grpc server port
		service.RegisterChannelzServiceToServer(s)
		state.SetDefaultGRPCServer(s)
	}

	// initCtx is used to clean up in case of partial initialization failures. For
	// example, user-configured servers open listeners during initialization and
	// if initialization fails at a later stage, say in probers or surfacers,
	// pr.Init returns an error and we cancel the initCtx, which makes servers
	// close their listeners.
	// TODO(manugarg): Plumb init context from cmd/cloudprober.
	initCtx, cancelFunc := context.WithCancel(context.TODO())
	pr, err := prober.Init(initCtx, cfg, globalLogger)
	if err != nil {
		cancelFunc()
		ln.Close()
		return err
	}

	cloudProber.prober = pr
	cloudProber.config = cfg
	cloudProber.configSource = configSrc
	cloudProber.defaultServerLn = ln
	cloudProber.defaultGRPCLn = grpcLn
	cloudProber.cancelInitCtx = cancelFunc

	// Start config watcher for hot-reload if enabled by flag/env var later.
	// Default debounce 1000ms.
	go func() {
		// Use background context; watcher will be stopped when process exits.
		ctx := context.Background()
		w, err := configwatcher.NewWatcher(ctx, configSrc, globalLogger, 1000)
		if err != nil {
			globalLogger.Errorf("config watcher init error: %v", err)
			return
		}

		for range w.Reload {
			if err := reloadConfigAndApply(); err != nil {
				globalLogger.Errorf("config reload failed: %v", err)
			}
		}
	}()

	return nil
}

// RunOnce runs a single probe.
func RunOnce(ctx context.Context, format, indent string) error {
	cloudProber.RLock()
	defer cloudProber.RUnlock()

	prrs, err := cloudProber.prober.Run(ctx)

	fmt.Println(singlerun.FormatProbeRunResults(prrs, singlerun.Format(format), indent))
	return err
}

// Start starts a previously initialized Cloudprober.
func Start(ctx context.Context) {
	cloudProber.Lock()
	defer cloudProber.Unlock()

	// Default servers
	srvMux := state.DefaultHTTPServeMux()
	httpSrv := &http.Server{Handler: srvMux}
	grpcSrv := state.DefaultGRPCServer()

	// Set up a goroutine to cleanup if context ends.
	go func() {
		<-ctx.Done()
		httpSrv.Close() // This will close the listener as well.
		if grpcSrv != nil {
			grpcSrv.Stop()
		}
		cloudProber.cancelInitCtx()
		cloudProber.Lock()
		defer cloudProber.Unlock()
		cloudProber.defaultServerLn = nil
		cloudProber.defaultGRPCLn = nil
		cloudProber.config = nil
		cloudProber.configSource = nil
		cloudProber.prober = nil
	}()

	go httpSrv.Serve(cloudProber.defaultServerLn)
	if grpcSrv != nil && cloudProber.defaultGRPCLn != nil {
		go grpcSrv.Serve(cloudProber.defaultGRPCLn)
	}

	if cloudProber.prober == nil {
		panic("Prober is not initialized. Did you call cloudprober.InitFromConfig first?")
	}

	cloudProber.prober.Start(ctx)
	srvMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	})
}

// GetConfig returns the prober config.
func GetConfig() *configpb.ProberConfig {
	cloudProber.RLock()
	defer cloudProber.RUnlock()
	return cloudProber.config
}

// GetRawConfig returns the prober config in text proto format.
func GetRawConfig() string {
	cloudProber.RLock()
	defer cloudProber.RUnlock()
	return cloudProber.configSource.RawConfig()
}

// GetParsedConfig returns the parsed prober config.
func GetParsedConfig() string {
	cloudProber.RLock()
	defer cloudProber.RUnlock()
	return cloudProber.configSource.ParsedConfig()
}

// GetInfo returns information on all the probes, servers and surfacers.
func GetInfo() (map[string]*probes.ProbeInfo, []*surfacers.SurfacerInfo, []*servers.ServerInfo) {
	cloudProber.RLock()
	defer cloudProber.RUnlock()
	return cloudProber.prober.Probes, cloudProber.prober.Surfacers, cloudProber.prober.Servers
}

func GetProber() *prober.Prober {
	cloudProber.RLock()
	defer cloudProber.RUnlock()
	return cloudProber.prober
}
