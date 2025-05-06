SHELL := /bin/bash
VERSION ?= $(shell git describe --tags --exclude tip)
BUILD_DATE ?= $(shell date +%s)
DIRTY = $(shell git diff --shortstat 2> /dev/null | wc -l | xargs) # xargs strips whitespace.
GIT_TAG := $(shell git describe --exact-match --exclude tip --tags HEAD 2>/dev/null || /bin/true)
GIT_COMMIT = $(strip $(shell git rev-parse --short HEAD))
GOBIN ?= ${GOPATH}/bin
BINARY ?= cloudprober
DOCKER_IMAGE ?= cloudprober/cloudprober
SOURCES := $(shell find . -name '*.go')
LDFLAGS ?= "-s -w -X main.version=$(VERSION) -X main.buildTimestamp=$(BUILD_DATE) -X main.dirty=$(DIRTY) -extldflags -static"
BINARY_SOURCE ?= "./cmd/cloudprober.go"

LINUX_PLATFORMS := linux-amd64 linux-arm64 linux-armv7
BINARIES := $(addprefix cloudprober-, $(LINUX_PLATFORMS) macos-amd64 macos-arm64 windows-amd64)

ifeq "$(GIT_TAG)" ""
	DOCKER_TAGS := -t $(DOCKER_IMAGE):master -t $(DOCKER_IMAGE):main
else
	DOCKER_TAGS := -t $(DOCKER_IMAGE):$(GIT_TAG) -t $(DOCKER_IMAGE):latest
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

config_docs:
	go install github.com/manugarg/protodoc/cmd/protodoc@latest
	protodoc --proto_root_dir=. --package_prefix=github.com/cloudprober/cloudprober \
		--format=yaml --out_dir=docs/_config_docs/yaml
	protodoc --proto_root_dir=. --package_prefix=github.com/cloudprober/cloudprober \
		--format=textpb --out_dir=docs/_config_docs/textpb

docker_multiarch: $(addprefix cloudprober-, $(LINUX_PLATFORMS)) Dockerfile
	docker buildx build --push \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		--build-arg VERSION=$(VERSION) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		--platform linux/amd64,linux/arm64,linux/arm/v7 \
		$(DOCKER_TAGS) .

dist: $(BINARIES)
	for bin in $(BINARIES) ; do \
	  bindir=$${bin/amd64/x86_64}; \
	  bindir=cloudprober-$(VERSION)-$${bindir/cloudprober-/}; \
	  mkdir -p $${bindir}; cp $${bin} $${bindir}/cloudprober; \
	  chmod a+rx $${bindir}/cloudprober; \
	  [[ "$${bin}" == *"windows"* ]] && mv $${bindir}/cloudprober{,.exe}; \
	  zip -r $${bindir}.zip $${bindir}/; rm -rf $${bindir}; \
	done

PYVERSION := $(subst v,,$(VERSION))
PYVERSION := $(word 1,$(subst -, ,$(PYVERSION)))-$(word 2,$(subst -, ,$(PYVERSION)))
py_serverutils:
	cp README.md probes/external/serverutils/py/README.md && \
	cd probes/external/serverutils/py && \
	sed -i "s/version = \"[^\"]*\"/version = \"$(PYVERSION)\"/" pyproject.toml && \
	python3 -m pip install build --user && \
	python3 -m build && \
	git checkout pyproject.toml && \
	rm README.md

install:
	GOBIN=$(GOBIN) CGO_ENABLED=0 go install -ldflags $(LDFLAGS) $(BINARY_SOURCE)

clean:
	rm -f cloudprober cloudprober-*
