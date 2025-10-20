// Copyright 2018-2023 The Cloudprober Authors.
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

// Package regex provides regex validator for the Cloudprober's
// validator framework.
package regex

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/cloudprober/cloudprober/logger"
)

// Validator implements a regex validator.
type Validator struct {
	r *regexp.Regexp
}

// Init initializes the regex validator.
// It compiles the regex in the configuration and returns an error if regex
// doesn't compile for some reason.
func (v *Validator) Init(config interface{}) error {
	regexStr, ok := config.(string)
	if !ok {
		return fmt.Errorf("%v is not a valid regex validator config", config)
	}
	if regexStr == "" {
		return errors.New("validator regex string cannot be empty")
	}

	r, err := regexp.Compile(regexStr)
	if err != nil {
		return fmt.Errorf("error compiling the given regex (%s): %v", regexStr, err)
	}

	v.r = r
	return nil
}

// Validate the provided responseBody and return true if responseBody matches
// the configured regex.
func (v *Validator) Validate(responseBody []byte, l *logger.Logger) (bool, error) {
	matched := v.r.Match(responseBody)
	if !matched {
		l.Warningf("Regex validation failure: response %s didn't match the regex %s", string(responseBody), v.r.String())
	}
	return matched, nil
}
