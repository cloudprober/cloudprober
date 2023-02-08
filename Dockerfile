# This Dockerfile expects "cloudprober" multi-platform binaries to exist in the
# same directory.
#
# Docker image built using this can executed in the following manner:
#   docker run --net host -v $PWD/cloudprober.cfg:/etc/cloudprober.cfg \
#                         cloudprober/cloudprober
FROM alpine
COPY cloudprober-linux-* ./

ARG TARGETPLATFORM
RUN if [ "$TARGETPLATFORM" = "linux/amd64" ]; then \
  mv cloudprober-linux-amd64 cloudprober && rm cloudprober-linux-*; fi
RUN if [ "$TARGETPLATFORM" = "linux/arm64" ]; then \
  mv cloudprober-linux-arm64 cloudprober && rm cloudprober-linux-*; fi
RUN if [ "$TARGETPLATFORM" = "linux/arm/v7" ]; then \
  mv cloudprober-linux-armv7 cloudprober && rm cloudprober-linux-*; fi

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
