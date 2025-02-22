// Copyright 2017-2021 The Cloudprober Authors.
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

package options

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/cloudprober/cloudprober/targets/endpoint"

	configpb "github.com/cloudprober/cloudprober/probes/proto"
)

// targetLabelType for target based additional labels
type targetLabelType int

// TargetLabelType enum values.
const (
	notTargetLabel targetLabelType = iota
	label
	name
	ip
	port
)

var targetLabelRegex = regexp.MustCompile(`target.label.(.*)`)

type targetToken struct {
	tokenType targetLabelType
	labelKey  string // target's label key.
}

// AdditionalLabel encapsulates additional labels to attach to probe results.
type AdditionalLabel struct {
	mu  sync.RWMutex
	Key string

	// If non-empty, additional label's value is independent of the target/
	staticValue string

	// This map will allow for quick value lookup for a target. It will be
	// updated by the probe while updating targets.
	valueForTarget map[string]string

	// At the time of parsing we split the label value at the delimiters ('@').
	// When we update an additional label for a target, we update the value
	// parts that correspond to the substitution tokens and join them back.
	valueParts []string

	// Target based substitution tokens.
	tokens []targetToken
}

// UpdateForTarget updates addtional label based on target's name and labels.
func (al *AdditionalLabel) UpdateForTarget(ep endpoint.Endpoint, ipAddr string, probePort int) {
	al.mu.Lock()
	defer al.mu.Unlock()

	// Return early if this label has a static value.
	if al.staticValue != "" {
		return
	}

	if al.valueForTarget == nil {
		al.valueForTarget = make(map[string]string)
	}

	if probePort == 0 {
		probePort = ep.Port
	}
	if ipAddr == "" && ep.IP != nil {
		ipAddr = ep.IP.String()
	}

	parts := append([]string{}, al.valueParts...)
	for i, tok := range al.tokens {
		switch tok.tokenType {
		case name:
			parts[2*i+1] = ep.Name
		case port:
			parts[2*i+1] = strconv.Itoa(probePort)
		case ip:
			parts[2*i+1] = ipAddr
		case label:
			parts[2*i+1] = ep.Labels[tok.labelKey]
		}
	}
	al.valueForTarget[ep.Key()] = strings.Join(parts, "")
}

// KeyValueForTarget returns key, value pair for the given target.
func (al *AdditionalLabel) KeyValueForTarget(ep endpoint.Endpoint) (key, val string) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	if al.staticValue != "" {
		return al.Key, al.staticValue
	}
	return al.Key, al.valueForTarget[ep.Key()]
}

// ParseAdditionalLabel parses an additional label proto message into an
// AdditionalLabel struct.
func ParseAdditionalLabel(alpb *configpb.AdditionalLabel) *AdditionalLabel {
	al := &AdditionalLabel{
		Key: alpb.GetKey(),
	}

	al.valueParts = strings.Split(alpb.GetValue(), "@")

	// No tokens
	if len(al.valueParts) == 1 {
		al.staticValue = alpb.GetValue()
		return al
	}

	// If there are even number of parts after the split above, that means we
	// don't have an even number of delimiters ('@'). Assume that the last
	// token is incomplete and attach '@' to the front of the last part.
	// e.g. @target.name@:@target.port
	//   valueParts: ["", "target.name", ":", "@target.port"]
	lenParts := len(al.valueParts)
	if lenParts%2 == 0 {
		al.valueParts[lenParts-1] = "@" + al.valueParts[lenParts-1]
	}
	// tokens[i] -> parts[2*i+1]
	// e.g. proto:@target.name@/@target.label.url@ -->
	//   valueParts: ["proto:", "target.name", "/", "target.label.url", ""]
	//   tokens:     ["target.name", "target.label.url"]
	numTokens := (len(al.valueParts) - 1) / 2
	for i := 0; i < numTokens; i++ {
		tokStr := al.valueParts[2*i+1]
		if tokStr == "target.name" {
			al.tokens = append(al.tokens, targetToken{tokenType: name})
			continue
		}
		if tokStr == "target.port" {
			al.tokens = append(al.tokens, targetToken{tokenType: port})
			continue
		}
		if tokStr == "target.ip" {
			al.tokens = append(al.tokens, targetToken{tokenType: ip})
			continue
		}
		matches := targetLabelRegex.FindStringSubmatch(tokStr)
		if len(matches) == 2 {
			al.tokens = append(al.tokens, targetToken{tokenType: label, labelKey: matches[1]})
		}
	}

	// if no valid tokens found, assign the value as it is.
	if len(al.tokens) == 0 {
		al.staticValue = alpb.GetValue()
	}

	return al
}

func parseAdditionalLabels(p *configpb.ProbeDef) []*AdditionalLabel {
	var aLabels []*AdditionalLabel

	for _, pb := range p.GetAdditionalLabel() {
		aLabels = append(aLabels, ParseAdditionalLabel(pb))
	}

	return aLabels
}
