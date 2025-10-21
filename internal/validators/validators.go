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

// Package validators provides an entrypoint for the cloudprober's validators
// framework.
package validators

import (
	"fmt"

	"github.com/cloudprober/cloudprober/internal/validators/http"
	"github.com/cloudprober/cloudprober/internal/validators/integrity"
	"github.com/cloudprober/cloudprober/internal/validators/json"
	configpb "github.com/cloudprober/cloudprober/internal/validators/proto"
	"github.com/cloudprober/cloudprober/internal/validators/regex"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
)

// Validator implements a validator.
//
// A validator runs a test on the provided input, usually the probe response,
// and returns the test result. If test cannot be run successfully for some
// reason (e.g. for malformed input), an error is returned.
type Validator struct {
	Name     string
	Validate func(input *Input, l *logger.Logger) (bool, error)
}

// Init initializes the validators defined in the config.
func Init(validatorConfs []*configpb.Validator) ([]*Validator, error) {
	var validators []*Validator
	names := make(map[string]bool)

	for _, vc := range validatorConfs {
		if vc.Name == "" {
			return nil, fmt.Errorf("validator name is required")
		}

		if names[vc.Name] {
			return nil, fmt.Errorf("validator %s is defined twice", vc.GetName())
		}

		v, err := initValidator(vc)
		if err != nil {
			return nil, err
		}

		validators = append(validators, v)
		names[vc.Name] = true
	}

	return validators, nil
}

func initValidator(validatorConf *configpb.Validator) (validator *Validator, err error) {
	validator = &Validator{Name: validatorConf.Name}

	switch validatorConf.Type.(type) {
	case *configpb.Validator_HttpValidator:
		v := &http.Validator{}
		if err := v.Init(validatorConf.GetHttpValidator()); err != nil {
			return nil, err
		}
		validator.Validate = func(input *Input, l *logger.Logger) (bool, error) {
			return v.Validate(input.Response, input.ResponseBody, l)
		}
		return

	case *configpb.Validator_IntegrityValidator:
		v := &integrity.Validator{}
		if err := v.Init(validatorConf.GetIntegrityValidator()); err != nil {
			return nil, err
		}
		validator.Validate = func(input *Input, l *logger.Logger) (bool, error) {
			return v.Validate(input.ResponseBody, l)
		}
		return

	case *configpb.Validator_JsonValidator:
		v := &json.Validator{}
		if err := v.Init(validatorConf.GetJsonValidator()); err != nil {
			return nil, err
		}
		validator.Validate = func(input *Input, l *logger.Logger) (bool, error) {
			return v.Validate(input.ResponseBody, l)
		}
		return

	case *configpb.Validator_Regex:
		v := &regex.Validator{}
		if err := v.Init(validatorConf.GetRegex()); err != nil {
			return nil, err
		}
		validator.Validate = func(input *Input, l *logger.Logger) (bool, error) {
			return v.Validate(input.ResponseBody, l)
		}
		return
	default:
		err = fmt.Errorf("unknown validator type: %v", validatorConf.Type)
		return
	}
}

// Input encapsulates the input for validators.
type Input struct {
	Response     interface{}
	ResponseBody []byte
}

// RunValidators runs the list of validators on the given response and
// responseBody, updates the given validationFailure map and returns the list
// of failures.
func RunValidators(vs []*Validator, input *Input, validationFailure *metrics.Map[int64], l *logger.Logger) []string {
	var failures []string

	for _, v := range vs {
		success, err := v.Validate(input, l)
		if err != nil {
			l.Error("Error while running the validator ", v.Name, ": ", err.Error())
			continue
		}
		if !success {
			if validationFailure != nil {
				validationFailure.IncKey(v.Name)
			}
			failures = append(failures, v.Name)
		}
	}

	return failures
}

// ValidationFailureMap returns an initialized validation failures map.
func ValidationFailureMap(vs []*Validator) *metrics.Map[int64] {
	m := metrics.NewMap("validator")
	// Initialize validation failure map with validator keys, so that we always
	// export the metrics.
	for _, v := range vs {
		m.IncKeyBy(v.Name, 0)
	}
	return m
}
