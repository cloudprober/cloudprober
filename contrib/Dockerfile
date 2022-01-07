# This Dockerfile builds the cloudprober contrib docker image. It uses
# cloudprober/cloudprober as the base image, and adds contrib binaries to the
# /contrib directory.
#
# Docker image built using this can executed in the following manner:
#   docker run --net host -v $PWD/cloudprober.cfg:/etc/cloudprober.cfg \
#                         cloudprober/cloudprober-contrib
ARG CLOUDPROBER_VERSION
FROM cloudprober/cloudprober:${CLOUDPROBER_VERSION} AS base

# This stage is used to find the correct binary for the platform. We store the
# correct binary at /stage-0-workdir/cloudprober, and in the next stage discard
# the rest.
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
COPY --from=stage0 /stage-0-workdir/bigquery_probe /contrib/

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
