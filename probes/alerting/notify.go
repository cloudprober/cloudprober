// Copyright 2023 The Cloudprober Authors.
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

package alerting

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/cloudprober/cloudprober/common/template"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/google/shlex"
)

// AlertInfo contains information about an alert.
type AlertInfo struct {
	Name             string
	ProbeName        string
	Target           endpoint.Endpoint
	FailureRatio     float32
	FailureThreshold float32
	FailingSince     time.Time
}

func alertFields(alertInfo *AlertInfo) map[string]string {
	fields := map[string]string{
		"alert":              alertInfo.Name,
		"probe":              alertInfo.ProbeName,
		"target":             alertInfo.Target.Dst(),
		"alerting_value":     fmt.Sprintf("%.2f", alertInfo.FailureRatio),
		"alerting_threshold": fmt.Sprintf("%.2f", alertInfo.FailureThreshold),
		"failing_since":      alertInfo.FailingSince.Format(time.RFC3339),
	}

	for k, v := range alertInfo.Target.Labels {
		fields["target.label."+k] = v
	}

	return fields
}

func (ah *AlertHandler) notify(ep endpoint.Endpoint, ts *targetState, failureRatio float32) {
	ah.l.Warningf("ALERT (%s): target (%s), failures higher than (%.2f) since (%v)", ah.name, ep.Name, ah.failureThreshold, ts.firstFailure)

	ts.alerted = true
	alertInfo := &AlertInfo{
		Name:             ah.name,
		ProbeName:        ah.probeName,
		Target:           ep,
		FailureRatio:     failureRatio,
		FailureThreshold: ah.failureThreshold,
		FailingSince:     ts.firstFailure,
	}

	if ah.notifyCh != nil {
		ah.notifyCh <- alertInfo
	}

	fields := alertFields(alertInfo)

	if ah.notifyConfig != nil && ah.notifyConfig.Command != "" {
		ah.notifyCommand(context.Background(), ah.notifyConfig.Command, fields)
	}
}

func (ah *AlertHandler) notifyCommand(ctx context.Context, command string, fields map[string]string) {
	cmdParts, err := shlex.Split(command)
	if err != nil {
		ah.l.Errorf("Error parsing command line (%s): %v", command, err)
		return
	}

	var parts []string
	for _, part := range cmdParts {
		res, foundAll := template.SubstituteLabels(part, fields)
		if foundAll {
			part = res
		}
		parts = append(parts, part)
	}

	ah.l.Infof("Starting external command: %s", strings.Join(cmdParts, " "))
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	if err = cmd.Start(); err != nil {
		ah.l.Errorf("error while starting the cmd: %s %s. Err: %v", cmd.Path, cmd.Args, err)
	}
}
