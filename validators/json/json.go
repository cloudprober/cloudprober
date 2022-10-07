// Copyright 2022 The Cloudprober Authors.
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

// Package json provides json validator for the Cloudprober's
// validator framework.
package json

import (
	"encoding/json"
	"fmt"

	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/validators/json/proto"
	"github.com/itchyny/gojq"
)

// Validator implements a regex validator.
type Validator struct {
	jqQuery *gojq.Query
}

// Init initializes the regex validator.
// It compiles the regex in the configuration and returns an error if regex
// doesn't compile for some reason.
func (v *Validator) Init(config interface{}, l *logger.Logger) error {
	cfg, ok := config.(*configpb.Validator)
	if !ok {
		return fmt.Errorf("%v is not a valid json validator config", config)
	}

	jqf := cfg.GetJqFilter()
	if jqf != "" {
		q, err := gojq.Parse(jqf)
		if err != nil {
			return fmt.Errorf("error parsing the given jq filter (%s): %v", jqf, err)
		}
		v.jqQuery = q
	}

	return nil
}

// Validate the provided responseBody and return true if responseBody matches
// the configured regex.
func (v *Validator) Validate(responseBody []byte) (bool, error) {
	var in interface{}
	err := json.Unmarshal(responseBody, &in)
	if err != nil {
		return false, err
	}

	if v.jqQuery != nil {
		iter := v.jqQuery.Run(in)

		var lastItem interface{}
		for {
			item, ok := iter.Next()
			if !ok {
				break
			}

			// If there is an error, return false and the error.
			err, ok := item.(error)
			if ok {
				return false, err
			}

			lastItem = item
		}

		b, ok := lastItem.(bool)
		if !ok {
			return false, fmt.Errorf("didn't get bool as the jq_filter output (%v)", lastItem)
		}
		return b, nil
	}

	return true, nil
}
