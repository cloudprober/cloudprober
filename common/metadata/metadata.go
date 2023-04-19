// Copyright 2021-2023 The Cloudprober Authors.
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

/*
Package metadata implements metadata related utilities.
*/
package metadata

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)

var uniqueID = struct {
	id string
	mu sync.Mutex
}{}

// IsKubernetes return true if running on Kubernetes.
// It uses the environment variable KUBERNETES_SERVICE_HOST to decide if we
// we are running Kubernetes.
func IsKubernetes() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

// KubernetesNamespace returns the Kubernetes namespace. It returns an empty
// string if there is an error in retrieving the namespace.
func KubernetesNamespace() string {
	namespaceBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err == nil {
		return string(namespaceBytes)
	}
	return ""
}

// IsCloudRunJob return true if we are running as a CloudRun job.
func IsCloudRunJob() bool {
	return os.Getenv("CLOUD_RUN_JOB") != ""
}

// IsCloudRunService return true if we are running as a CloudRun service.
func IsCloudRunService() bool {
	return os.Getenv("K_SERVICE") != ""
}

func UniqueID() string {
	uniqueID.mu.Lock()
	defer uniqueID.mu.Unlock()

	if uniqueID.id != "" {
		return uniqueID.id
	}

	t := time.Now()

	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	var randomStrLen = 6
	rand.Seed(t.UnixNano())

	b := make([]rune, randomStrLen)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	uniqueID.id = fmt.Sprintf("cloudprober-%s-%04d", strings.ToLower(string(b)), t.Unix()%10000)
	return uniqueID.id
}
