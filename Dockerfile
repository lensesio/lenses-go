FROM alpine:latest
ARG TARGETARCH TARGETOS

RUN apk upgrade --no-cache && apk add --no-cache bash jq curl gettext
ADD bin/lenses-cli-linux-${TARGETARCH} /opt/lenses/lenses-cli
ENV PATH /opt/lenses/:$PATH
