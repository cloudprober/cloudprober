package mytargets

import (
	"fmt"
	"net"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/cloudprober/cloudprober/targets/resolver"
)

var globalResolver = resolver.New()

type myTargets struct {
	c *MyTargetsConf
}

func (ut *myTargets) ListEndpoints() []endpoint.Endpoint {
	return []endpoint.Endpoint{{
		Name: ut.c.GetHostname(),
		Port: int(ut.c.GetPort()),
	}}
}

func (ut *myTargets) Resolve(name string, ipVer int) (net.IP, error) {
	return globalResolver.Resolve(name, ipVer)
}

func Init() {
	extensionNumber := int(E_Mytargets.TypeDescriptor().Number())
	targets.RegisterTargetsType(extensionNumber, func(c interface{}, l *logger.Logger) (targets.Targets, error) {
		utConf, ok := c.(*MyTargetsConf)
		if !ok {
			return nil, fmt.Errorf("invalid target config type: %T", c)
		}
		return &myTargets{
			c: utConf,
		}, nil
	})
}
