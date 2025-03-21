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

// Package lameduck implements a lameducks provider. Lameduck provider fetches
// lameducks from the RTC (Runtime Configurator) service. This functionality
// allows an operator to do hitless VM upgrades. If a target is set to be in
// lameduck by the operator, it is taken out of the targets list.
package lameduck

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"cloud.google.com/go/compute/metadata"
	rdsclient "github.com/cloudprober/cloudprober/internal/rds/client"
	rdsclientpb "github.com/cloudprober/cloudprober/internal/rds/client/proto"
	"github.com/cloudprober/cloudprober/internal/rds/gcp"
	rdspb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/internal/rds/server"
	serverconfigpb "github.com/cloudprober/cloudprober/internal/rds/server/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/state"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	configpb "github.com/cloudprober/cloudprober/targets/lameduck/proto"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"github.com/cloudprober/cloudprober/targets/rtc/rtcservice"
	"google.golang.org/protobuf/proto"
)

// Lameducker provides an interface to Lameduck/Unlameduck an instance.
//
// Cloudprober doesn't currently (as of July, 2018) use this interface by
// itself. It's provided here so that other software (e.g. probing deployment
// management software) can lameduck/unlameduck instances in a way that
// Cloudprober understands.
type Lameducker interface {
	Lameduck(name string) error
	Unlameduck(name string) error
}

// global.lister is a singleton Lister. It caches data from the upstream config
// service, allowing for multiple consumers to lookup for lameducks without
// increasing load on the upstream service.
var global struct {
	mu     sync.RWMutex
	lister endpoint.Lister
}

// service provides methods to do lameduck operations on VMs.
type service struct {
	rtc  rtcservice.Config
	opts *configpb.Options
	l    *logger.Logger
}

// Lameduck puts the target in lameduck mode.
func (ldSvc *service) Lameduck(name string) error {
	return ldSvc.rtc.Write(name, []byte{0})
}

// Unlameduck removes the target from lameduck mode.
func (ldSvc *service) Unlameduck(name string) error {
	err := ldSvc.rtc.Delete(name)
	return err
}

// NewService creates a new lameduck service using the provided config options
// and an oauth2 enabled *http.Client; if the client is set to nil, an oauth
// enabled client is created automatically using GCP default credentials.
func newService(opts *configpb.Options, proj string, hc *http.Client, l *logger.Logger) (*service, error) {
	if opts == nil {
		return nil, fmt.Errorf("lameduck.Init: failed to construct lameduck Service: no lameDuckOptions given")
	}
	if l == nil {
		l = &logger.Logger{}
	}

	cfg := opts.GetRuntimeconfigName()

	rtc, err := rtcservice.New(proj, cfg, hc)
	if err != nil {
		return nil, fmt.Errorf("lameduck.Init : rtcconfig service initialization failed : %v", err)
	}

	return &service{
		rtc:  rtc,
		opts: opts,
		l:    l,
	}, nil
}

func getProject(opts *configpb.Options) (string, error) {
	project := opts.GetRuntimeconfigProject()
	if project == "" {
		var err error
		project, err = metadata.ProjectID()
		if err != nil {
			return "", fmt.Errorf("lameduck.getProject: error while getting project id: %v", err)
		}
	}
	return project, nil
}

// NewLameducker creates a new lameducker using the provided config and an
// oauth2 enabled *http.Client; if the client is set to nil, an oauth enabled
// client is created automatically using GCP default credentials.
func NewLameducker(opts *configpb.Options, hc *http.Client, l *logger.Logger) (Lameducker, error) {
	project, err := getProject(opts)
	if err != nil {
		return nil, err
	}
	return newService(opts, project, hc, l)
}

func (li *lister) newRDSServer() (*server.Server, error) {
	resTypes := make(map[string]string)
	if li.rtcConfig != "" {
		resTypes[gcp.ResourceTypes.RTCVariables] = li.rtcConfig
	}
	if li.pubsubTopic != "" {
		resTypes[gcp.ResourceTypes.PubsubMessages] = li.pubsubTopic
	}

	pc := gcp.DefaultProviderConfig([]string{li.project}, resTypes, int(li.opts.GetReEvalSec()), "")
	return server.New(context.Background(), &serverconfigpb.ServerConf{Provider: []*serverconfigpb.Provider{pc}}, nil, li.l)
}

func (li *lister) rdsClient(baseResourcePath string, additionalFilter *rdspb.Filter) (*rdsclient.Client, error) {
	rdsClientConf := &rdsclientpb.ClientConf{
		ServerOptions: li.rdsServerOpts,
		Request: &rdspb.ListResourcesRequest{
			Provider:     proto.String("gcp"),
			ResourcePath: proto.String(fmt.Sprintf("%s/%s", baseResourcePath, li.project)),
			Filter: []*rdspb.Filter{
				{
					Key:   proto.String("updated_within"),
					Value: proto.String(fmt.Sprintf("%ds", li.opts.GetExpirationSec())),
				},
			},
		},
		ReEvalSec: proto.Int32(li.opts.GetReEvalSec()),
	}

	if additionalFilter != nil {
		rdsClientConf.Request.Filter = append(rdsClientConf.Request.Filter, additionalFilter)
	}

	return rdsclient.New(rdsClientConf, li.listResourcesFunc, li.l)
}

func (li *lister) initClients() error {
	if li.rtcConfig != "" {
		li.l.Infof("lameduck: creating RDS client for RTC variables")

		additionalFilter := &rdspb.Filter{
			Key:   proto.String("config_name"),
			Value: proto.String(li.opts.GetRuntimeconfigName()),
		}

		cl, err := li.rdsClient("rtc_variables", additionalFilter)
		if err != nil {
			return err
		}
		li.clients = append(li.clients, cl)
	}

	if li.pubsubTopic != "" {
		li.l.Infof("lameduck: creating RDS client for PubSub messages")

		// Here we assume that subscription name contains the topic name. This is
		// true for the RDS implmentation.
		additionalFilter := &rdspb.Filter{
			Key:   proto.String("subscription"),
			Value: proto.String(li.pubsubTopic),
		}

		cl, err := li.rdsClient("pubsub_messages", additionalFilter)
		if err != nil {
			return err
		}
		li.clients = append(li.clients, cl)
	}

	return nil
}

func (li *lister) ListEndpoints() []endpoint.Endpoint {
	var result []endpoint.Endpoint
	for _, cl := range li.clients {
		result = append(result, cl.ListEndpoints()...)
	}

	if len(result) != 0 {
		li.l.Infof("Lameducked targets: %v", result)
	}
	return result
}

type lister struct {
	opts              *configpb.Options
	project           string
	rtcConfig         string
	pubsubTopic       string
	rdsServerOpts     *rdsclientpb.ClientConf_ServerOptions
	listResourcesFunc rdsclient.ListResourcesFunc
	clients           []*rdsclient.Client
	l                 *logger.Logger
}

func newLister(globalOpts *targetspb.GlobalTargetsOptions, l *logger.Logger) (*lister, error) {
	opts := globalOpts.GetLameDuckOptions()
	li := &lister{
		opts:          opts,
		rdsServerOpts: globalOpts.GetRdsServerOptions(),
		rtcConfig:     opts.GetRuntimeconfigName(),
		pubsubTopic:   opts.GetPubsubTopic(),
		l:             l,
	}

	var err error
	li.project, err = getProject(opts)
	if err != nil {
		return nil, err
	}

	// If there are lameduck specific RDS server options, use them.
	if li.opts.GetRdsServerOptions() != nil {
		li.rdsServerOpts = li.opts.GetRdsServerOptions()
	}

	// If no RDS server options are configured, look for a local one.
	if li.rdsServerOpts == nil {
		localRDSServer := state.LocalRDSServer()
		if localRDSServer == nil {
			li.l.Infof("rds_server_address not given and found no local RDS server, creating a new one.")

			var err error
			localRDSServer, err = li.newRDSServer()
			if err != nil {
				return nil, fmt.Errorf("error while creating local RDS server: %v", err)
			}
		}
		li.listResourcesFunc = localRDSServer.ListResources
	}

	return li, li.initClients()
}

// InitDefaultLister initializes the package using the given arguments. If a
// lister is given in the arguments, global.lister is set to that, otherwise a
// new lameduck service is created using the config options, and global.lister
// is set to that service. Initiating the package from a given lister is useful
// for testing pacakges that depend on this package.
func InitDefaultLister(globalOpts *targetspb.GlobalTargetsOptions, lister endpoint.Lister, l *logger.Logger) error {
	global.mu.Lock()
	defer global.mu.Unlock()

	// Make sure we initialize global.lister only once.
	if global.lister != nil {
		return nil
	}

	// If a lister has been provided, use that. It's useful for testing.
	if lister != nil {
		global.lister = lister
		return nil
	}

	if globalOpts.GetLameDuckOptions().GetUseRds() {
		l.Warning("lameduck: use_rds doesn't do anything anymore and will soon be removed.")
	}

	lister, err := newLister(globalOpts, l)
	if err != nil {
		return err
	}

	global.lister = lister
	return nil
}

// GetDefaultLister returns the global Lister. If global lister is
// uninitialized, it returns an error.
func GetDefaultLister() (endpoint.Lister, error) {
	global.mu.RLock()
	defer global.mu.RUnlock()
	if global.lister == nil {
		return nil, errors.New("global lameduck service not initialized")
	}
	return global.lister, nil
}
