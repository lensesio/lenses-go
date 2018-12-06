FROM alpine:latest
ADD bin/lenses-cli-linux-amd64 /opt/lenses/lenses-cli
ENV PATH /opt/lenses/:$PATH
