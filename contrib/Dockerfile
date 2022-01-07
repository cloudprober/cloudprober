# This Dockerfile expects "cloudprober" binary and ca-certificates to exist
# in the working directory.
#
# Docker image built using this can executed in the following manner:
#   docker run --net host -v $PWD/cloudprober.cfg:/etc/cloudprober.cfg \
#                         cloudprober/cloudprober-contrib

# This stage is used to find the correct binary for the platform. We store the
# correct binary at /stage-0-workdir/cloudprober, and in the next stage discard
# the rest.
ARG DOCKER_VERSION
FROM cloudprober/cloudprober:${DOCKER_VERSION} AS base

FROM busybox as stage0
WORKDIR /stage-0-workdir
COPY bigquery_probe-linux-* ./

ARG TARGETPLATFORM
RUN if [ "$TARGETPLATFORM" = "linux/amd64" ]; then \
  mv bigquery_probe-linux-amd64 bigquery_probe; fi
RUN if [ "$TARGETPLATFORM" = "linux/arm64" ]; then \
  mv bigquery_probe-linux-arm64 bigquery_probe; fi
RUN if [ "$TARGETPLATFORM" = "linux/arm/v7" ]; then \
  mv bigquery_probe-linux-armv7 bigquery_probe; fi

FROM base
COPY --from=stage0 /stage-0-workdir/bigquery_probe /contrib

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
