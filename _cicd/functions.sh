#!/usr/bin/env bash

# Be extra strict
set -o xtrace   # print out each command before executed
set -o errexit  # exit when a command fails
set -o nounset  # exit when try to use undefined variable
set -o pipefail # return exit code of the last piped command

VERSION="${LENSES_CLI_CUR_VERSION}"
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
    echo "Linting is temporarily disabled"
}

format-check() {
    # pin version so that a new goimports does not tank our build.
    go install golang.org/x/tools/cmd/goimports@v0.3.0

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
        for GOARCH in amd64 arm64; do
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
    go tool cover -html=cover.out -o cover.html
}

# Builds the archives. In release mode we add the version to the archive
# name. We expect it to run in a directory containing the binaries
archive() {
    # Args: [build mode] → $1, [version] → $2

    # If we are in release mode, we append the version to the archive filename
    local _ARCHIVE_VERSION=""
    if [[ "$1" == "release" ]]; then
        _ARCHIVE_VERSION="-${2}"
    fi

    for GOOS in linux darwin windows; do
        for GOARCH in amd64 arm64; do
            local _ARCHIVE_DIRECTORY=lenses-cli-$GOOS-${GOARCH}-${2}

            mkdir ${_ARCHIVE_DIRECTORY}
            cp ${WORKSPACE}/{LICENSE,README.md,NOTICE} ${_ARCHIVE_DIRECTORY}

            case $GOOS in
                linux|darwin)
                    mv lenses-cli-$GOOS-${GOARCH} ${_ARCHIVE_DIRECTORY}/lenses-cli
                    tar czf lenses-cli-$GOOS-${GOARCH}${_ARCHIVE_VERSION}.tar.gz --owner=root --group=root ${_ARCHIVE_DIRECTORY}
                    sha256sum lenses-cli-$GOOS-${GOARCH}${_ARCHIVE_VERSION}.tar.gz > lenses-cli-$GOOS-${GOARCH}${_ARCHIVE_VERSION}.tar.gz.sha256
                    ;;
                windows)
                    mv lenses-cli-$GOOS-${GOARCH} ${_ARCHIVE_DIRECTORY}/lenses-cli.exe
                    zip -r lenses-cli-$GOOS-${GOARCH}${_ARCHIVE_VERSION}.zip ${_ARCHIVE_DIRECTORY}
                    sha256sum lenses-cli-$GOOS-${GOARCH}${_ARCHIVE_VERSION}.zip > lenses-cli-$GOOS-${GOARCH}${_ARCHIVE_VERSION}.zip.sha256
                    ;;
                *)
                    echo "Unexepected GOOS: $GOOS"
                    exit 1
                    ;;
            esac

            rm -rf ${_ARCHIVE_DIRECTORY}
        done
    done

    find .
}

# This converts a branch name to a name for a docker tag.
# For now it replaces slashes / with hyphens but we can adjust the logic if we discover other
# breaking characters
branchNameToDockerTag() {
    # Args: [branch name] → $1
    # Out: [docker tag name]

    # Yes we could use 'echo ${1//\//-}' but let's be friendly
    echo "$1" | tr / -
}

build-docker-img() {
    # Args: [image name] → 1, [image tag] → 2, [branch_name] → $3

    # Prepare a folder with the necssary files for the gcloud builder
    mkdir -p cli-docker/bin
    cp lenses-cli-linux-amd64 lenses-cli-linux-arm64 cli-docker/bin
    cp ${WORKSPACE}/Dockerfile cli-docker/

    cat <<EOF | tee cli-docker/cloudbuild.yaml
steps:
- name: 'docker'
  args: [ 'buildx', 'create', '--name', 'mybuilder', '--use' ]
- name: 'docker'
  args: [ 'buildx', 'build', '--platform', 'linux/arm64,linux/amd64', '-t', '${1}:${3}', '--push', '.']
EOF

    # Submit the build job to gcloud builder
    gcloud builds submit cli-docker --config cli-docker/cloudbuild.yaml --timeout=5m
    gcloud container images add-tag ${1}:${3} ${1}:${2}
}

clean() {
    rm -rf bin/
    rm environment
    rm cover.out
}

# Activates GCloud Account
activateGCloudAccount() {
    # Args: GCLOUD_PROJECT → $1
    # Vars: GLOUD_KEY_PATH

    set +o xtrace
    cat "${GLOUD_KEY_PATH}" | gcloud auth activate-service-account --key-file=- --project "${1}"
}

# Deactivates GCloud Account
revokeGCloudAccount() {
    gcloud auth revoke
}

# Upload Artifacts to GCloud
gcloudUploadArtifacts() {
    # Args: [files] → $1, GGLOUD_BUCKET_PATH → $2

    gsutil --version
    gsutil -m cp -r $1 "gs://${2}/"
}

# Run the function at $1, pass the rest of the args
$1 "${@:2}"
