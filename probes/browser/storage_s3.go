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

package browser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/browser/proto"
)

type s3Storage struct {
	client *s3.Client
	bucket string
	path   string
	l      *logger.Logger
}

func initS3(ctx context.Context, s3config *configpb.S3, l *logger.Logger) (*s3Storage, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(s3config.GetRegion()),
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

	s3Storage := &s3Storage{
		client: s3Client,
		bucket: s3config.GetBucket(),
		path:   s3config.GetPath(),
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

// uploadDirToS3 syncs a local directory to an S3 path
func (s *s3Storage) uploadDirToS3(ctx context.Context, localPath, basePath string) error {
	// Check if the local directory exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("local directory %s does not exist", localPath)
	}

	// Walk the local directory and upload each file to S3
	err := filepath.Walk(localPath, func(localFilePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Get the relative path of the file
		relPath, err := filepath.Rel(basePath, localFilePath)
		if err != nil {
			return err
		}

		// Construct the S3 key
		s3Key := filepath.Join(s.path, relPath)

		// Open the local file
		file, err := os.Open(localFilePath)
		if err != nil {
			return err
		}
		defer file.Close()

		// Upload the file to S3
		_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: &s.bucket,
			Key:    &s3Key,
			Body:   file,
		})
		if err != nil {
			return err
		}

		s.l.Debugf("Uploaded %s to %s", localFilePath, s3Key)

		return nil
	})

	return err
}
