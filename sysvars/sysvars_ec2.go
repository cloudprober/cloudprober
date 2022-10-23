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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/cloudprober/cloudprober/logger"
)

var ec2Vars = func(sysVars map[string]string, tryHard bool, l *logger.Logger) (bool, error) {
	ctx := context.Background()

	client, err := createAWSIMDSClient(ctx, tryHard)
	if err != nil {
		l.Warningf("sysvars_ec2: could not create AWS session: %v", err)
		return false, err
	}

	id, err := client.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
	// The premise behind the error handling here, is that we want to evaluate
	// if we are running in ec2 or not.
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

// createAWSIMDSClient creates a new IMDS client, with retries enabled if tryHard is true,
// and retries disabled if tryHard is false
func createAWSIMDSClient(ctx context.Context, tryHard bool) (*imds.Client, error) {
	cfg, err := loadAWSConfig(ctx, tryHard)
	if err != nil {
		return nil, err
	}

	return imds.NewFromConfig(cfg), nil
}

// loadAWSConfig will evaluate if we should apply retries or not to the AWS config
// https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/retries-timeouts/
func loadAWSConfig(ctx context.Context, tryHard bool) (aws.Config, error) {
	if !tryHard {
		return config.LoadDefaultConfig(ctx, config.WithRetryer(func() aws.Retryer {
			return aws.NopRetryer{}
		}))
	}

	return config.LoadDefaultConfig(ctx)
}
