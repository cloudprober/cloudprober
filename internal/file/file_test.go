// Copyright 2020 The Cloudprober Authors.
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

package file

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createTempFile(t *testing.T, b []byte) string {
	tmpfile, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
		return ""
	}

	defer tmpfile.Close()
	if _, err := tmpfile.Write(b); err != nil {
		t.Fatal(err)
	}

	return tmpfile.Name()
}

func testReadFile(ctx context.Context, path string) ([]byte, error) {
	return []byte("content-for-" + path), nil
}

func TestReadFile(t *testing.T) {
	prefixToReadfunc["test://"] = testReadFile

	os.Setenv("TESTDATAX", "test-data-env")
	defer os.Unsetenv("TESTDATAX")

	testData := map[string]string{
		"test://test-file":                             "content-for-test-file",
		createTempFile(t, []byte("temp-content")):      "temp-content",
		createTempFile(t, []byte("temp-$TESTDATAX")):   "temp-test-data-env",
		createTempFile(t, []byte("temp-${TESTDATAX}")): "temp-test-data-env",
	}

	for path, expectedContent := range testData {
		t.Run("ReadFile("+path+")", func(t *testing.T) {
			var b []byte
			var err error
			if strings.Contains(expectedContent, "env") {
				b, err = ReadFile(context.Background(), path, WithEnvSubstitution())
			} else {
				b, err = ReadFile(context.Background(), path)
			}
			if err != nil {
				t.Fatalf("Error while reading the file: %s", path)
			}

			if string(b) != expectedContent {
				t.Errorf("ReadFile(%s) = %s, expected=%s", path, string(b), expectedContent)
			}
		})
	}
}

func TestReadWithCache(t *testing.T) {
	f, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatalf("Error creating temp file: %v", err)
	}
	defer os.Remove(f.Name())

	writeToTempFile := func(s string) {
		if _, err := f.WriteAt([]byte(s), 0); err != nil {
			t.Fatalf("Error writing to temp file (%s): %v", f.Name(), err)
		}
	}

	readAndVerify := func(expectedContent string, reloadInterval time.Duration) {
		b, err := ReadWithCache(context.Background(), f.Name(), reloadInterval)
		assert.NoError(t, err, "reading file")
		assert.Equal(t, expectedContent, string(b))
	}

	testContent := "test-content"
	writeToTempFile(testContent)

	// No cache
	readAndVerify(testContent, 0)
	writeToTempFile(testContent + "-updated")
	readAndVerify(testContent+"-updated", 0)

	// with cache
	writeToTempFile(testContent + "-updated-2")
	// we see old content before cache expiration.
	// Note: we use 2 seconds here to make sure that the test doesn't fail due
	// to timing issues.
	readAndVerify(testContent+"-updated", 2*time.Second)
	// wait for cache to expire
	time.Sleep(time.Second)
	readAndVerify(testContent+"-updated-2", 1*time.Second)
}
