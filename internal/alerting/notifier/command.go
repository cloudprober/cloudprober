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

package notifier

import (
	"context"
	"os/exec"
	"strings"

	"github.com/cloudprober/cloudprober/common/strtemplate"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/google/shlex"
)

type commandNotifier struct {
	cmdParts []string
	l        *logger.Logger
}

func (cn *commandNotifier) Notify(ctx context.Context, fields map[string]string) error {
	var cmdParts = append([]string{}, cn.cmdParts...)
	for i, part := range cmdParts {
		res, foundAll := strtemplate.SubstituteLabels(part, fields)
		if !foundAll {
			cn.l.Warningf("couldn't substitute all labels in command part: %s", part)
		}
		cmdParts[i] = res
	}

	cn.l.Infof("Starting external command: %s", strings.Join(cmdParts, " "))

	cmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)

	return cmd.Start()
}

func newCommandNotifier(cmd string, l *logger.Logger) (*commandNotifier, error) {
	parts, err := shlex.Split(cmd)
	if err != nil {
		return nil, err
	}
	return &commandNotifier{cmdParts: parts}, nil
}
