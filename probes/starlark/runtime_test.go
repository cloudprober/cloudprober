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

package starlark

import (
	"context"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewRuntime_TopLevelExecutionHonorsContext(t *testing.T) {
	source := `
def spin():
    for _ in range(1000000000000000000):
        pass

spin()

def probe(target):
    pass
`
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := newRuntime(ctx, &runtimeOpts{
		name:       "script-load-timeout",
		source:     source,
		entryPoint: "probe",
		l:          &logger.Logger{},
	})
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.Less(t, elapsed, time.Second, "load-time Starlark did not honor probe timeout")
}
