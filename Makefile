SHELL := /bin/bash
VERSION ?= $(shell git describe --tags --exclude tip)
BUILD_DATE ?= $(shell date +%s)
DIRTY = $(shell git diff --shortstat 2> /dev/null | wc -l | xargs) # xargs strips whitespace.
GIT_TAG := $(shell git describe --exact-match --exclude tip --tags HEAD 2>/dev/null || /bin/true)
GIT_COMMIT = $(strip $(shell git rev-parse --short HEAD))
GOBIN ?= ${GOPATH}/bin
BINARY ?= cloudprober
DOCKER_IMAGE ?= cloudprober/cloudprober
DOCKER_BUILD_ARGS ?= --build-arg BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") --build-arg VERSION=$(VERSION) --build-arg VCS_REF=$(GIT_COMMIT)
SOURCES := $(shell find . -name '*.go')
VAR_LD_FLAGS := -X main.version=$(VERSION) -X main.buildTimestamp=$(BUILD_DATE) -X main.dirty=$(DIRTY)
LDFLAGS ?= "$(VAR_LD_FLAGS) -s -w -extldflags -static"
BINARY_SOURCE ?= "./cmd/cloudprober/."

BINARY_BASE ?= cloudprober

BUILD_TAG_FLAG :=
# We don't include xds build by default due to its size. You can build binaries
# with xds support by setting BUILD_WITH_XDS=true.
# e.g. BUILD_WITH_XDS=true make cloudprober-xds-macos-arm64
BUILD_WITH_XDS ?= false
ifeq "$(BUILD_WITH_XDS)" "true"
	BINARY_BASE := cloudprober-xds
	BUILD_TAG_FLAG := -tags grpc_xds
endif

LINUX_PLATFORMS := linux-amd64 linux-arm64 linux-armv7
BINARIES := $(addprefix $(BINARY_BASE)-, $(LINUX_PLATFORMS) macos-amd64 macos-arm64 windows-amd64)

ifeq "$(GIT_TAG)" ""
	DOCKER_TAGS := -t $(DOCKER_IMAGE):master -t $(DOCKER_IMAGE):main
	DOCKER_FIPS_TAGS := -t $(DOCKER_IMAGE):master-fips -t $(DOCKER_IMAGE):main-fips
	DOCKER_PW_TAGS := -t $(DOCKER_IMAGE):master-pw -t $(DOCKER_IMAGE):main-pw
else
	DOCKER_TAGS := -t $(DOCKER_IMAGE):$(GIT_TAG) -t $(DOCKER_IMAGE):latest
	DOCKER_FIPS_TAGS := -t $(DOCKER_IMAGE):$(GIT_TAG)-fips -t $(DOCKER_IMAGE):latest-fips
	DOCKER_PW_TAGS := -t $(DOCKER_IMAGE):$(GIT_TAG)-pw -t $(DOCKER_IMAGE):latest-pw
endif

define make-binary-target
$1: $(SOURCES)
	BINNAME=$1; \
	BINNAME=$$$${BINNAME#$(BINARY_BASE)-}; \
	GOOS=$$$$(echo $$$${BINNAME} | cut -d- -f1); \
	GOOS=$$$${GOOS/macos/darwin}; \
	GOARCH=$$$$(echo $$$${BINNAME} | cut -d- -f2); \
	GOARCH=$$$${GOARCH/armv7/arm}; \
	GOARM=; \
	[[ "$$$${BINNAME}" == *"armv7"* ]] && GOARM=7; \
	CGO_ENABLED=0 GOOS=$$$${GOOS} GOARCH=$$$${GOARCH} GOARM=$$$${GOARM} go build $(BUILD_TAG_FLAG) -o $1 -ldflags $(LDFLAGS) $(BINARY_SOURCE)
endef

# To cross-compile, for linux arm64 e.g., set GO_BUILD_FLAGS like this:
# make cloudprober-fips GO_BUILD_FLAGS="GOARCH=arm64 CC_FOR_TARGET=gcc-aarch64-linux-gnu CC=aarch64-linux-gnu-gcc"
cloudprober-fips: $(SOURCES)
	$(GO_BUILD_FLAGS) CGO_ENABLED=1 GOEXPERIMENT=boringcrypto go build -o cloudprober-fips -tags netgo,osusergo -ldflags "$(VAR_LD_FLAGS) -w -linkmode external -extldflags -static" $(BINARY_SOURCE)
	go tool nm cloudprober-fips | grep crypto/internal/boring/sig.BoringCrypto.abi0 > /dev/null || (echo "FIPS build failed: BoringCrypto not used" && rm cloudprober-fips && exit 1)
	strip cloudprober-fips

test:
	go test -v -race -covermode=atomic ./...

$(foreach bin,$(BINARIES),$(eval $(call make-binary-target,$(bin))))

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -o $@ -ldflags $(LDFLAGS) $(BINARY_SOURCE)

docker_multiarch: $(addprefix cloudprober-, $(LINUX_PLATFORMS)) Dockerfile
	docker buildx build --push $(DOCKER_BUILD_ARGS) \
		--platform linux/amd64,linux/arm64,linux/arm/v7 \
		$(DOCKER_TAGS) .

docker_multiarch_fips: cloudprober-fips-linux-amd64 cloudprober-fips-linux-arm64 Dockerfile.fips
	docker buildx build --push $(DOCKER_BUILD_ARGS) \
		--platform linux/amd64,linux/arm64 \
		$(DOCKER_FIPS_TAGS) -f Dockerfile.fips .

docker_multiarch_pw: Dockerfile.pw
	docker buildx build --push $(DOCKER_BUILD_ARGS) \
		--platform linux/amd64,linux/arm64 \
		$(DOCKER_PW_TAGS) -f Dockerfile.pw .

dist: $(BINARIES)
	for bin in $(BINARIES) ; do \
	  bindir=$${bin/amd64/x86_64}; \
	  bindir=$(BINARY_BASE)-$(VERSION)-$${bindir/$(BINARY_BASE)-/}; \
	  mkdir -p $${bindir}; cp $${bin} $${bindir}/cloudprober; \
	  chmod a+rx $${bindir}/cloudprober; \
	  [[ "$${bin}" == *"windows"* ]] && mv $${bindir}/cloudprober{,.exe}; \
	  zip -r $${bindir}.zip $${bindir}/; rm -rf $${bindir}; \
	done

PYVERSION := $(subst v,,$(VERSION))
PYVERSION := $(word 1,$(subst -, ,$(PYVERSION)))-$(word 2,$(subst -, ,$(PYVERSION)))
PYVERSION := $(patsubst %-,%,$(PYVERSION))
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
	rm -f cloudprober cloudprober-* cloudprober-xds-*
