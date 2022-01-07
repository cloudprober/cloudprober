# This Dockerfile expects "cloudprober" binary and ca-certificates to exist
# in the working directory.
#
# Docker image built using this can executed in the following manner:
#   docker run --net host -v $PWD/cloudprober.cfg:/etc/cloudprober.cfg \
#                         cloudprober/cloudprober

# This stage is used to find the correct binary for the platform. We store the
# correct binary at /stage-0-workdir/cloudprober, and in the next stage discard
# the rest.
FROM busybox AS stage0
WORKDIR /stage-0-workdir
COPY cloudprober-linux-* ./

ARG TARGETPLATFORM
RUN if [ "$TARGETPLATFORM" = "linux/amd64" ]; then \
  mv cloudprober-linux-amd64 cloudprober; fi
RUN if [ "$TARGETPLATFORM" = "linux/arm64" ]; then \
  mv cloudprober-linux-arm64 cloudprober; fi
RUN if [ "$TARGETPLATFORM" = "linux/arm/v7" ]; then \
  mv cloudprober-linux-armv7 cloudprober; fi

FROM busybox
COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=stage0 /stage-0-workdir/cloudprober /

# Metadata params
ARG BUILD_DATE
ARG VERSION
ARG VCS_REF
# Metadata
LABEL org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.name="Cloudprober" \
      org.label-schema.vcs-url="https://github.com/cloudprober/cloudprober" \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.version=$VERSION \
      com.microscaling.license="Apache-2.0"

ENTRYPOINT ["/cloudprober", "--logtostderr"]
