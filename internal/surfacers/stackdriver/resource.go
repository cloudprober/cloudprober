// Copyright 2021 The Cloudprober Authors.
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

package stackdriver

import (
	"os"
	"strings"

	"cloud.google.com/go/compute/metadata"
	md "github.com/cloudprober/cloudprober/common/metadata"
	"github.com/cloudprober/cloudprober/logger"
	monitoring "google.golang.org/api/monitoring/v3"
)

func kubernetesResource(projectID string) (*monitoring.MonitoredResource, error) {
	namespace := md.KubernetesNamespace()

	// We ignore error in getting cluster name and location. These attributes
	// should be set while running on GKE.
	clusterName, _ := metadata.InstanceAttributeValue("cluster-name")
	location, _ := metadata.InstanceAttributeValue("cluster-location")

	// We can likely use cluster-location instance attribute for location. Using
	// zone provides more granular scope though.
	return &monitoring.MonitoredResource{
		Type: "k8s_container",
		Labels: map[string]string{
			"cluster_name":   clusterName,
			"location":       location,
			"project_id":     projectID,
			"pod_name":       os.Getenv("HOSTNAME"),
			"namespace_name": namespace,
			// To get the `container_name` label, users need to explicitly provide it.
			"container_name": os.Getenv("CONTAINER_NAME"),
		},
	}, nil
}

func cloudRunResource(projectID, job, taskID string, l *logger.Logger) *monitoring.MonitoredResource {
	region, err := metadata.Get("instance/region")
	if err != nil {
		l.Warningf("Stackdriver surfacer: Error getting Cloud Run region (%v), ignoring..", err)
		region = ""
	}
	location := region[strings.LastIndex(region, "/")+1:]

	// As per stackdriver custom metrics policy[1], there is no monitored
	// resource type corresponding to a Cloud Run job or service, so we use the
	// type "generic_task".
	// [1] https://cloud.google.com/monitoring/custom-metrics/creating-metrics#md-create
	return &monitoring.MonitoredResource{
		Type: "generic_task",
		Labels: map[string]string{
			"project_id": projectID,
			"location":   location,
			"job":        job,
			"task_id":    md.UniqueID(),
			"namespace":  "cloudprober",
		},
	}
}

func gceResource(projectID string, l *logger.Logger) (*monitoring.MonitoredResource, error) {
	name, err := metadata.InstanceName()
	if err != nil {
		name, err = metadata.Hostname()
		if err != nil {
			l.Warningf("Stackdriver surfacer: Error getting GCE instance or host name (%v), ignoring..", err)
		}
	}

	zone, err := metadata.Zone()
	if err != nil {
		l.Warningf("Stackdriver surfacer: Error getting GCE zone (%v), ignoring..", err)
		zone = ""
	}

	return &monitoring.MonitoredResource{
		Type: "gce_instance",
		Labels: map[string]string{
			"project_id": projectID,
			// Note that following is not correct, as we are adding instance name as
			// instance id, but this is what we have been doing all along, and for
			// monitoring instance name may be more useful, at least as of now there
			// is no automatic instance to custom-monitoring mapping.
			"instance_id": name,
			"zone":        zone,
		},
	}, nil
}

func monitoredResourceOnGCE(projectID string, l *logger.Logger) (*monitoring.MonitoredResource, error) {
	if md.IsKubernetes() {
		return kubernetesResource(projectID)
	}
	if md.IsCloudRunJob() {
		return cloudRunResource(projectID, os.Getenv("CLOUD_RUN_JOB"), os.Getenv("CLOUD_RUN_TASK_INDEX"), l), nil
	}
	if md.IsCloudRunService() {
		return cloudRunResource(projectID, os.Getenv("K_SERVICE"), os.Getenv("K_REVISION"), l), nil
	}
	return gceResource(projectID, l)
}
