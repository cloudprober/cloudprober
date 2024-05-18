// Copyright 2018-2019 The Cloudprober Authors.
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

// Package validators implements helper functions to run all validators on
// probe responses and to initialize validation failure map. This packages is
// primarily intended to be used by extension probes. Built-in and external
// probes should use the internal validators package.
package validators

import (
	"github.com/cloudprober/cloudprober/internal/validators"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/options"
)

// RunValidators runs the list of validators on the given response and
// responseBody, updates the given validationFailure map and returns the list
// of failures.
func RunValidators(opts *options.Options, response []byte, validationFailure *metrics.Map[int64], l *logger.Logger) []string {
	return validators.RunValidators(opts.Validators, &validators.Input{ResponseBody: response}, validationFailure, l)
}

// ValidationFailureMap returns an initialized validation failures map.
func ValidationFailureMap(opts *options.Options) *metrics.Map[int64] {
	return validators.ValidationFailureMap(opts.Validators)
}
