SHELL := /bin/bash
VERSION ?= $(shell git describe --tags)
DOCKER_VERSION ?= $(VERSION)
GIT_COMMIT = $(strip $(shell git rev-parse --short HEAD))
GOBIN ?= ${GOPATH}/bin
BINARY ?= cloudprober
DOCKER_IMAGE ?= cloudprober/cloudprober
CACERTS ?= /etc/ssl/certs/ca-certificates.crt
SOURCES := $(shell find . -name '*.go')
LDFLAGS ?= "-s -w -X main.version=$(VERSION) -extldflags -static"
BINARY_SOURCE ?= "./cmd/cloudprober.go"

LINUX_PLATFORMS := linux-amd64 linux-arm64 linux-armv7
BINARIES := $(addprefix cloudprober-, $(LINUX_PLATFORMS) macos-amd64 macos-arm64 windows-amd64)

ifeq "$(DOCKER_VERSION)" "master"
	DOCKER_TAGS := -t $(DOCKER_IMAGE):$(DOCKER_VERSION)
else
	DOCKER_TAGS := -t $(DOCKER_IMAGE):$(DOCKER_VERSION) -t $(DOCKER_IMAGE):latest
endif

define make-binary-target
$1: $(SOURCES)
	GOOS=$(subst macos,darwin,$(word 2,$(subst -, ,$1))) ; \
	GOARCH=$(subst armv7,arm,$(word 3,$(subst -, ,$1))) ; \
	GOARM=$(subst armv7,7,$(filter armv7,$(word 3,$(subst -, ,$1)))) ; \
	CGO_ENABLED=0 GOOS=$$$${GOOS} GOARCH=$$$${GOARCH} GOARM=$$$${GOARM} go build -o $1 -ldflags $(LDFLAGS) $(BINARY_SOURCE)
endef

test:
	go test -v -race -covermode=atomic ./...

$(foreach bin,$(BINARIES),$(eval $(call make-binary-target,$(bin))))

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -o $@ -ldflags $(LDFLAGS) $(BINARY_SOURCE)

ca-certificates.crt: $(CACERTS)
	cp $(CACERTS) ca-certificates.crt

docker_multiarch: $(addprefix cloudprober-, $(LINUX_PLATFORMS)) ca-certificates.crt Dockerfile
	docker buildx build --push \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		--build-arg VERSION=$(VERSION) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		--platform linux/amd64,linux/arm64,linux/arm/v7 \
		--file Dockerfile.buildx \
		$(DOCKER_TAGS) .

docker_build: $(BINARY) ca-certificates.crt Dockerfile
	docker build \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		--build-arg VERSION=$(VERSION) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE) .

docker_push:
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):$(DOCKER_VERSION)
	docker login -u "${DOCKER_USER}" -p "${DOCKER_PASS}"
	docker push $(DOCKER_IMAGE):$(DOCKER_VERSION)

docker_push_tagged:
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):$(DOCKER_VERSION)
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):latest
	docker login -u "${DOCKER_USER}" -p "${DOCKER_PASS}"
	docker image push --all-tags $(DOCKER_IMAGE)

dist: $(BINARIES)
	for bin in $(BINARIES) ; do \
	  bindir=$${bin/amd64/x86_64}; \
	  bindir=cloudprober-$(VERSION)-$${bindir/cloudprober-/}; \
	  mkdir -p $${bindir}; cp $${bin} $${bindir}/cloudprober; \
	  chmod a+rx $${bindir}/cloudprober; \
	  [[ "$${bin}" == *"windows"* ]] && mv $${bindir}/cloudprober{,.exe}; \
	  zip -r $${bindir}.zip $${bindir}/; rm -rf $${bindir}; \
	done

install:
	GOBIN=$(GOBIN) CGO_ENABLED=0 go install -ldflags $(LDFLAGS) $(BINARY_SOURCE)

clean:
	rm -f cloudprober cloudprober-*
