// Copyright 2024 The Cloudprober Authors.
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

package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/browser/artifacts/proto"
)

type S3 struct {
	client *s3.Client
	bucket string
	path   string
	l      *logger.Logger
}

func InitS3(ctx context.Context, s3config *configpb.S3, storagePath string, l *logger.Logger) (*S3, error) {
	region := s3config.GetRegion()
	if region == "" && os.Getenv("AWS_REGION") != "" {
		region = os.Getenv("AWS_REGION")
	}

	if region == "" {
		return nil, fmt.Errorf("region is required for S3 storage, either set it in the config or in the environment variable AWS_REGION")
	}

	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}

	if s3config.GetAccessKeyId() != "" && s3config.GetSecretAccessKey() != "" {
		credsProvider := credentials.NewStaticCredentialsProvider(s3config.GetAccessKeyId(), s3config.GetSecretAccessKey(), "")
		opts = append(opts, config.WithCredentialsProvider(credsProvider))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if s3config.GetEndpoint() != "" {
			o.BaseEndpoint = s3config.Endpoint
		}
	})

	s3Storage := &S3{
		client: s3Client,
		bucket: s3config.GetBucket(),
		path:   storagePath,
		l:      l,
	}

	// Verify that we have access to the bucket
	if _, err = s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: &s3Storage.bucket,
	}); err != nil {
		return nil, fmt.Errorf("failed to access bucket %s: %v", s3config.GetBucket(), err)
	}

	return s3Storage, nil
}

// store syncs a local directory to an S3 path
func (s *S3) Store(ctx context.Context, localPath string, destPathFn func(string) string) error {
	s.l.Infof("Uploading artifacts from %s to: s3://%s/%s", localPath, s.bucket, s.path)

	return walkAndSave(ctx, localPath, destPathFn, func(ctx context.Context, r io.Reader, relPath string) error {
		s3Key := filepath.Join(s.path, relPath)

		if _, err := s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: &s.bucket,
			Key:    &s3Key,
			Body:   r,
		}); err != nil {
			return err
		}
		return nil
	})
}
