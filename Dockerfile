FROM alpine:latest
RUN apk add --no-cache bash jq curl
ADD bin/lenses-cli-linux-amd64 /opt/lenses/lenses-cli
ENV PATH /opt/lenses/:$PATH
