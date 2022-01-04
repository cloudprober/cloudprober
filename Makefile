VERSION ?= $(shell git describe --tags)
DOCKER_VERSION ?= $(VERSION)
GIT_COMMIT = $(strip $(shell git rev-parse --short HEAD))
GOBIN ?= ${GOPATH}/bin
BINARY ?= cloudprober
DOCKER_IMAGE ?= cloudprober/cloudprober
CACERTS ?= /etc/ssl/certs/ca-certificates.crt
SOURCES := $(shell find . -name '*.go')

test:
	go test -v -race -covermode=atomic ./...

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -o $@ -ldflags "-X main.version=$(VERSION) -extldflags -static" ./cmd/cloudprober.go

$(BINARY)-windows-amd64: $(SOURCES)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $@ -ldflags "-X main.version=$(VERSION) -extldflags -static" ./cmd/cloudprober.go

$(BINARY)-linux-amd64: $(SOURCES)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $@ -ldflags "-X main.version=$(VERSION) -extldflags -static" ./cmd/cloudprober.go

$(BINARY)-linux-arm64: $(SOURCES)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $@ -ldflags "-X main.version=$(VERSION) -extldflags -static" ./cmd/cloudprober.go

$(BINARY)-linux-armv7: $(SOURCES)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o $@ -ldflags "-X main.version=$(VERSION) -extldflags -static" ./cmd/cloudprober.go

$(BINARY)-macos-amd64: $(SOURCES)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $@ -ldflags "-X main.version=$(VERSION) -extldflags -static" ./cmd/cloudprober.go

$(BINARY)-macos-arm64: $(SOURCES)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $@ -ldflags "-X main.version=$(VERSION) -extldflags -static" ./cmd/cloudprober.go

ca-certificates.crt: $(CACERTS)
	cp $(CACERTS) ca-certificates.crt

docker_build: $(BINARY) ca-certificates.crt Dockerfile
	docker build \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		--build-arg VERSION=$(VERSION) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE)  .

docker_multiarch: $(BINARY)-linux-amd64 $(BINARY)-linux-arm64 $(BINARY)-linux-armv7 ca-certificates.crt Dockerfile
	docker buildx build --push \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		--build-arg VERSION=$(VERSION) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		--platform linux/amd64,linux/arm64,linux/arm/v7 \
		--file Dockerfile.buildx \
		-t $(DOCKER_IMAGE):$(DOCKER_VERSION)  .

docker_push:
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):$(DOCKER_VERSION)
	docker login -u "${DOCKER_USER}" -p "${DOCKER_PASS}"
	docker push $(DOCKER_IMAGE):$(DOCKER_VERSION)

docker_push_tagged:
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):$(DOCKER_VERSION)
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):latest
	docker login -u "${DOCKER_USER}" -p "${DOCKER_PASS}"
	docker image push --all-tags $(DOCKER_IMAGE)

install:
	GOBIN=$(GOBIN) CGO_ENABLED=0 go install -ldflags "-X main.version=$(VERSION) -extldflags -static" ./cmd/cloudprober.go

clean:
	rm cloudprober
	go get -u ./...
