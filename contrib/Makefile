VERSION ?= $(shell git describe --tags)
GIT_TAG := $(shell git describe --exact-match --tags HEAD 2>/dev/null || /bin/true)
GIT_COMMIT = $(strip $(shell git rev-parse --short HEAD))
GOBIN ?= ${GOPATH}/bin
DOCKER_IMAGE ?= cloudprober/cloudprober-contrib
CACERTS ?= /etc/ssl/certs/ca-certificates.crt
SOURCES := $(shell find . -name '*.go')
LDFLAGS ?= "-s -w -X main.version=$(VERSION) -extldflags -static"

ifeq "$(GIT_TAG)" ""
	DOCKER_TAGS := -t $(DOCKER_IMAGE):master
	CLOUDPROBER_VERSION := master
else
	DOCKER_TAGS := -t $(DOCKER_IMAGE):$(GIT_TAG) -t $(DOCKER_IMAGE):latest
	CLOUDPROBER_VERSION := latest
endif

# Build bigquery probe
BIGQUERY_BINARY ?= bigquery_probe
BIGQUERY_BINARY_SOURCE ?= ./gcp/cmd/bigquery_probe.go

$(BIGQUERY_BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -o $@ -ldflags $(LDFLAGS) $(BIGQUERY_BINARY_SOURCE)

$(BIGQUERY_BINARY)-linux-amd64: $(SOURCES)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $@ -ldflags $(LDFLAGS) $(BIGQUERY_BINARY_SOURCE)

$(BIGQUERY_BINARY)-linux-arm64: $(SOURCES)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $@ -ldflags $(LDFLAGS) $(BIGQUERY_BINARY_SOURCE)

$(BIGQUERY_BINARY)-linux-armv7: $(SOURCES)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o $@ -ldflags $(LDFLAGS) $(BIGQUERY_BINARY_SOURCE)


docker_multiarch: $(BIGQUERY_BINARY)-linux-amd64 $(BIGQUERY_BINARY)-linux-arm64 $(BIGQUERY_BINARY)-linux-armv7 Dockerfile
	docker buildx build --push \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		--build-arg VERSION=$(VERSION) \
		--build-arg CLOUDPROBER_VERSION=$(CLOUDPROBER_VERSION) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		--platform linux/amd64,linux/arm64,linux/arm/v7 \
		$(DOCKER_TAGS) .

clean:
	rm -f bigquery_probe bigquery_probe-*
