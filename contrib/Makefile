VERSION ?= $(shell git describe --tags)
DOCKER_VERSION ?= $(VERSION)
GIT_COMMIT = $(strip $(shell git rev-parse --short HEAD))
GOBIN ?= ${GOPATH}/bin
DOCKER_IMAGE ?= cloudprober/cloudprober-contrib
CACERTS ?= /etc/ssl/certs/ca-certificates.crt
SOURCES := $(shell find . -name '*.go')
LDFLAGS ?= "-s -w -X main.version=$(VERSION) -extldflags -static"

ifeq "$(DOCKER_VERSION)" "master"
	DOCKER_TAGS := -t $(DOCKER_IMAGE):$(DOCKER_VERSION)
else
	DOCKER_TAGS := -t $(DOCKER_IMAGE):$(DOCKER_VERSION) -t $(DOCKER_IMAGE):latest
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


docker_multiarch: $(BIGQUERY_BINARY)-linux-amd64 $(BIGQUERY_BINARY)-linux-arm64 $(BIGQUERY_BINARY)-linux-armv7 Dockerfile.buildx
	docker buildx build --push \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		--build-arg VERSION=$(VERSION) \
		--build-arg DOCKER_VERSION=$(DOCKER_VERSION) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		--platform linux/amd64,linux/arm64,linux/arm/v7 \
		--file Dockerfile.buildx \
		$(DOCKER_TAGS) .

docker_build: bigquery_probe Dockerfile
	docker build \
                --build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
                --build-arg VERSION=$(VERSION) \
                --build-arg DOCKER_VERSION=$(DOCKER_VERSION) \
                --build-arg VCS_REF=$(GIT_COMMIT) \
                -t $(DOCKER_IMAGE):$(DOCKER_VERSION)  .

docker_push:
	docker tag $(DOCKER_IMAGE):$(DOCKER_VERSION) $(DOCKER_IMAGE):$(DOCKER_VERSION)
	docker login -u "${DOCKER_USER}" -p "${DOCKER_PASS}"
	docker push $(DOCKER_IMAGE):$(DOCKER_VERSION)

docker_push_tagged:
	docker tag $(DOCKER_IMAGE):$(DOCKER_VERSION) $(DOCKER_IMAGE):$(DOCKER_VERSION)
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):latest
	docker login -u "${DOCKER_USER}" -p "${DOCKER_PASS}"
	docker image push --all-tags $(DOCKER_IMAGE)

clean:
	rm bigquery_probe
	go get -u ./...
