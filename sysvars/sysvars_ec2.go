// Copyright 2019-2020 The Cloudprober Authors.
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

package sysvars

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/cloudprober/cloudprober/logger"
)

var ec2Vars = func(sysVars map[string]string, tryHard bool, l *logger.Logger) (bool, error) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRetryer(func() aws.Retryer {
		if !tryHard {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 1)
		}
		return nil
	}))
	if err != nil {
		l.Warningf("sysvars_ec2: failed to load default config: %v", err)
		return false, nil
	}

	client := imds.NewFromConfig(cfg)

	if !tryHard {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()
	}

	id, err := client.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
	// the premise behind the error handling here, is that we want to evaluate if we are running in ec2 or not.
	if err != nil {
		sysVars["EC2_METADATA_Available"] = "false"
		return false, fmt.Errorf("sysvars_ec2: could not get instance identity document %v", err)
	}

	sysVars["EC2_METADATA_Available"] = "true"
	sysVars["EC2_AvailabilityZone"] = id.AvailabilityZone
	sysVars["EC2_PrivateIP"] = id.PrivateIP
	sysVars["EC2_Region"] = id.Region
	sysVars["EC2_InstanceID"] = id.InstanceID
	sysVars["EC2_InstanceType"] = id.InstanceType
	sysVars["EC2_ImageID"] = id.ImageID
	sysVars["EC2_KernelID"] = id.KernelID
	sysVars["EC2_RamdiskID"] = id.RamdiskID
	sysVars["EC2_Architecture"] = id.Architecture
	return true, nil
}
