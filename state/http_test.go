// Copyright 2018-2025 The Cloudprober Authors.
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

package state

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPServeMux(t *testing.T) {
	if mux := DefaultHTTPServeMux(); mux != nil {
		t.Fatalf("State has http serve mux unexpectedly set. Got %v Want nil", mux)
	}
	testMux := http.NewServeMux()
	if testMux == nil {
		t.Fatal("Unable to create a test http serve mux")
	}
	SetDefaultHTTPServeMux(testMux)
	assert.Equal(t, testMux, DefaultHTTPServeMux(), "http serve mux")
}

func TestAddWebHandlerAndIsHandled(t *testing.T) {
	testMux := http.NewServeMux()
	SetDefaultHTTPServeMux(testMux)

	// Test when no handlers are added
	assert.False(t, IsHandled("/test"), "Expected /test to not be handled")

	// Add a handler and test
	err := AddWebHandler("/test", func(w http.ResponseWriter, r *http.Request) {})
	assert.Nil(t, err, "Expected no error when adding a handler")
	assert.True(t, IsHandled("/test"), "Expected /test to be handled")

	// Test a URL that is not handled
	assert.False(t, IsHandled("/not-handled"), "Expected /not-handled to not be handled")

	// Test with nil ServeMux
	SetDefaultHTTPServeMux(nil)
	assert.False(t, IsHandled("/test"), "Expected /test to not be handled with nil ServeMux")
}

func TestAllLinks(t *testing.T) {
	testMux := http.NewServeMux()
	SetDefaultHTTPServeMux(testMux)
	defer SetDefaultHTTPServeMux(nil)

	// Test when no handlers are added
	links := AllLinks()
	assert.Empty(t, links, "Expected no links when no handlers are added")

	// Add a handler and test
	AddWebHandler("/test", func(w http.ResponseWriter, r *http.Request) {})
	assert.Equal(t, []string{"/test"}, AllLinks())

	// Add another handler and test
	AddWebHandler("/another-test", func(w http.ResponseWriter, r *http.Request) {})
	assert.Equal(t, []string{"/test", "/another-test"}, AllLinks())
}

func TestAddWebHandlerWithArtifactsLink(t *testing.T) {
	testMux := http.NewServeMux()
	SetDefaultHTTPServeMux(testMux)
	defer SetDefaultHTTPServeMux(nil)

	// Add a handler with artifact link option
	err := AddWebHandler("/test-artifact", func(w http.ResponseWriter, r *http.Request) {}, WithArtifactsLink(""))
	assert.Nil(t, err, "Expected no error when adding a handler")

	// Add a handler with artifact link option and custom path
	err = AddWebHandler("/test-handler", func(w http.ResponseWriter, r *http.Request) {}, WithArtifactsLink("/custom-artifact"))
	assert.Nil(t, err, "Expected no error when adding a handler")

	// Verify artifacts URLs
	expected := []string{"/test-artifact", "/custom-artifact"}
	assert.Equal(t, expected, ArtifactsURLs())
}

func TestAddWebHandlerNoStateMutationOnMuxConflict(t *testing.T) {
	testMux := http.NewServeMux()
	SetDefaultHTTPServeMux(testMux)
	defer SetDefaultHTTPServeMux(nil)

	err := AddWebHandler("/foo/{id}", func(w http.ResponseWriter, r *http.Request) {}, WithArtifactsLink("/foo/1"))
	assert.NoError(t, err)

	err = AddWebHandler("/foo/{name}", func(w http.ResponseWriter, r *http.Request) {}, WithArtifactsLink("/foo/2"))
	assert.Error(t, err)
	assert.Equal(t, []string{"/foo/{id}"}, AllLinks())
	assert.Equal(t, []string{"/foo/1"}, ArtifactsURLs())
}

func TestLinksAreReturnedAsCopies(t *testing.T) {
	testMux := http.NewServeMux()
	SetDefaultHTTPServeMux(testMux)
	defer SetDefaultHTTPServeMux(nil)

	err := AddWebHandler("/test-artifact", func(w http.ResponseWriter, r *http.Request) {}, WithArtifactsLink("/custom-artifact"))
	assert.NoError(t, err)

	links := AllLinks()
	artifacts := ArtifactsURLs()

	links[0] = "/mutated-link"
	artifacts[0] = "/mutated-artifact"

	assert.Equal(t, []string{"/test-artifact"}, AllLinks())
	assert.Equal(t, []string{"/custom-artifact"}, ArtifactsURLs())
}
