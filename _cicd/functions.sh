#!/usr/bin/env bash

# Be extra strict
set -o xtrace   # print out each command before executed
set -o errexit  # exit when a command fails
set -o nounset  # exit when try to use undefined variable
set -o pipefail # return exit code of the last piped command

VERSION="${LENSES_VERSION:-$(git describe --tags)}"
REVISION="${GIT_COMMIT:=$(git rev-parse HEAD)}"
LDFLAGS="-s -w -X main.buildVersion=${VERSION} -X main.buildRevision=${REVISION} -X main.buildTime=$(date +%s)"

# thanks to http://mywiki.wooledge.org/BashFAQ/028
LOCAL_ENV="${BASH_SOURCE%/*}/local.env"
# shellcheck disable=SC1090
if [[ -f "$LOCAL_ENV" ]]; then
    source "$LOCAL_ENV"
fi

setup() {
    # print version of tools used
    go version
    make --version

    # download deps for all target platforms
    for GOOS in linux darwin windows; do
        # 386 not needed
        for GOARCH in amd64; do
            export GOOS GOARCH
            go mod download
        done
    done


    # remove go cache if 'release' mode
    # if [[ $BUILD_MODE == "release" ]]; then
    #     rm -rf /tmp/cli-cache/go
    # fi
}

lint() {
    # download misc. deps
    go get -u golang.org/x/lint/golint

    export PATH=$PATH:$GOPATH/bin
    golint -set_exit_status $(go list ./... | grep -v /vendor/)
}

format-check() {
    go get -u golang.org/x/tools/cmd/goimports

    export PATH=$PATH:$GOPATH/bin
    GOFORMATOUT=$(goimports -l cmd/ pkg/ test/)
    if [[ -z $GOFORMATOUT ]]; then
        echo "Go code format-check complete!"
    else
        echo "The following files contain formatting issues:"
        echo $GOFORMATOUT
        exit 1
    fi
}

# Builds lenses-cli
build() {
    go build -v -o ./bin/lenses-cli ./cmd/lenses-cli
}

cross-build() {
    export CGO_ENABLED=0
    for GOOS in linux darwin windows; do
        # 386 not needed
        for GOARCH in amd64; do
            export GOOS GOARCH
            if [[ $BUILD_MODE == 'release' ]]; then
                go build -ldflags "${LDFLAGS}" -v -o \
                    ./bin/lenses-cli-$GOOS-$GOARCH ./cmd/lenses-cli
            else
                LDFLAGS_DEV="-X main.buildVersion=${VERSION} \
                    -X main.buildRevision=${REVISION} \
                    -X main.buildTime=$(date +%s)"
                go build -ldflags "$LDFLAGS_DEV" -v -o ./bin/lenses-cli-$GOOS-$GOARCH \
                    ./cmd/lenses-cli
            fi
        done
    done
}

test() {
    go test -coverprofile=cover.out ./...
    go tool cover -func=cover.out
}

archive() {
    # Archive and calculate sha256 for each file
    mkdir -p bin/bucket
    pushd bin/
    cp ${WORKSPACE}/{LICENSE,README.md,NOTICE} .
    for GOOS in linux darwin windows; do
        local _BIN_SUFFIX=""
        if [[ $GOOS == 'windows' ]]; then
                _BIN_SUFFIX=".exe"
            else
                _BIN_SUFFIX=""
            fi
        mv lenses-cli-$GOOS-amd64 "lenses-cli${_BIN_SUFFIX}"
        tar --create --gzip --file bucket/lenses-cli-$GOOS-amd64.tar.gz \
            --owner=root --group=root \
            "lenses-cli${_BIN_SUFFIX}" \
            {LICENSE,README.md,NOTICE}
        sha256sum bucket/lenses-cli-$GOOS-amd64.tar.gz > \
            bucket/lenses-cli-$GOOS-amd64.tar.gz.sha256
        mv "lenses-cli${_BIN_SUFFIX}" lenses-cli-$GOOS-amd64
    done
    popd

    # Copy go test artifacts
    cp cover.out bin/bucket

    # Persist env. vars of job
    cp environment bin/bucket

    # Gcloud setup and upload all contents from bucket folder
    gcloud auth activate-service-account --key-file=$GCLOUD_SA_KEY_PATH \
        --project=$GCLOUD_PROJECT
    gsutil -m cp bin/bucket/* gs://$GCLOUD_BUCKET_PATH/
}

build-docker-img() {
    # gcloud authentication
    gcloud auth activate-service-account --key-file=$GCLOUD_SA_KEY_PATH \
        --project=$GCLOUD_PROJECT

    # Prepare a folder with the necssary files for the gcloud builder
    mkdir -p bin/cloud/bin
    cp bin/lenses-cli-linux-amd64 bin/cloud/bin
    cp Dockerfile bin/cloud/


    # Submit the build job to gcloud builder
    gcloud builds submit bin/cloud \
        --timeout=3m \
        --tag eu.gcr.io/lenses-ci/lenses-cli:${BRANCH_NAME//\//-}
    gcloud container images add-tag eu.gcr.io/lenses-ci/lenses-cli:${BRANCH_NAME//\//-} \
        eu.gcr.io/lenses-ci/lenses-cli:v${VERSION}
}

clean() {
    rm -rf bin/
    rm environment
    rm cover.out
}

# Run the function at $1, pass the rest of the args
$1 "${@:2}"
