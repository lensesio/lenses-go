FROM scratch
COPY bin/lenses-cli-linux-amd64 /opt/lenses/lenses-cli
ENV PATH /opt/lenses/:$PATH
