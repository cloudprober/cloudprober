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

// Package dns provides a DNS validator for the Cloudprober's validator
// framework.
package dns

import (
	"fmt"

	configpb "github.com/cloudprober/cloudprober/internal/validators/dns/proto"
	"github.com/cloudprober/cloudprober/logger"
	mdns "github.com/miekg/dns"
)

// Validator implements a validator for DNS responses.
type Validator struct {
	c *configpb.Validator
}

// Init initializes the DNS validator.
func (v *Validator) Init(c *configpb.Validator) error {
	if c == nil || c.Authoritative == nil {
		return fmt.Errorf("bad dns validator config (%v): no validation criteria specified", c)
	}

	v.c = c
	return nil
}

// Validate the provided input and return true if input is valid. Validate
// expects the input to be of the type: *dns.Msg. Note that it doesn't use the
// responseBody, it's part of the function signature to satisfy the validator
// interface.
func (v *Validator) Validate(input interface{}, unused []byte, l *logger.Logger) (bool, error) {
	resp, ok := input.(*mdns.Msg)
	if !ok {
		return false, fmt.Errorf("input %v is not of type dns.Msg", input)
	}

	if v.c.Authoritative != nil && resp.Authoritative != v.c.GetAuthoritative() {
		l.Errorf("DNS validation failure: got authoritative answer flag: %v, want: %v", resp.Authoritative, v.c.GetAuthoritative())
		return false, nil
	}

	return true, nil
}
