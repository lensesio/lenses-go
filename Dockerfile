FROM alpine:latest
ADD bin/lenses-cli-linux-amd64 /opt/lenses/lenses-cli
ADD secret-provider /opt/lenses/secret-provider
ENV PATH /opt/lenses/:$PATH
