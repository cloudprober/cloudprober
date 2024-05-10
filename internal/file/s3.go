// Copyright 2020-2023 The Cloudprober Authors.
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
Package file implements utilities to read files from various backends.
*/
package file

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// readFileFromS3 reads a file using an S3 Bucket URL
// s3://bucket-name/path/to/file.txt
func readFileFromS3(ctx context.Context, objectPath string) ([]byte, error) {
	bucket, object, err := parseObjectURL(objectPath)
	if err != nil {
		return nil, err
	}

	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load default config: %v", err)
	}
	s3Client := s3.NewFromConfig(sdkConfig)

	result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve file (%s): %v", objectPath, err)
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

func s3ModTime(ctx context.Context, objectPath string) (time.Time, error) {
	bucket, object, err := parseObjectURL(objectPath)
	if err != nil {
		return time.Time{}, err
	}

	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to load default config: %v", err)
	}
	s3Client := s3.NewFromConfig(sdkConfig)

	result, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to retrieve file (%s): %v", objectPath, err)
	}

	return *result.LastModified, nil
}
