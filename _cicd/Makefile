SHELL := /usr/bin/env bash -o errexit -o nounset -o pipefail
.PHONY: *

# thanks to https://gist.github.com/mpneuried/0594963ad38e68917ef189b4e6a269db
# ENV_FILE ?= _build/local.env
# include $(ENV_FILE)

# JENKINS_URL=
# JENKINS_USER_ID=
# JENKINS_API_TOKEN=
# JENKINS_CLI_PATH=
# JENKINS_JOB=
# JENKINS_JOB_PARAMS=
JENKINS_CLI = java -jar ${JENKINS_CLI_PATH}

# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help: ## This help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / \
            {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

jenkins-cli-help: ##
	${JENKINS_CLI} help
	@printf "\033[36m%-30s\033[0m %s\n" "'$@' finished successfully!"

lint: ##
	${JENKINS_CLI} declarative-linter < ../Jenkinsfile
	@printf "\033[36m%-30s\033[0m %s\n" "'$@' finished successfully!"

build: ##
	@# e.g. make build JENKINS_JOB_PARAMS="DOCKER_GO_IMG=golang:1.11"
	@if [[ -n "${JENKINS_JOB_PARAMS}" ]]; then\
		${JENKINS_CLI} build ${JENKINS_JOB} -v -f -p ${JENKINS_JOB_PARAMS};\
	fi
	${JENKINS_CLI} build ${JENKINS_JOB} -v -f

replay: ##
	${JENKINS_CLI} replay-pipeline ${JENKINS_JOB} < ../Jenkinsfile

restart: ##
	@# JENKINS_STAGE is case sensitive
	${JENKINS_CLI} restart-from-stage --job ${JENKINS_JOB} \
	    --stage "${JENKINS_STAGE}"

console: ##
	@# $JENKINS_JOB should be in the form of <job_name/branch> for mb
	${JENKINS_CLI} console ${JENKINS_JOB}

watch-console: ##
	watch -d -n 0.1 -c "${JENKINS_CLI} console ${JENKINS_JOB} -f | tail -50"

docker-build: ##
	docker run --rm \
	    --volume $(shell dirname $(shell pwd)):/src --workdir /src \
	    --volume /tmp:/tmp \
	    --user $(shell id --user):$(shell id --group) \
	    --env HOME=/tmp/cli-cache/home \
	    --env GOPATH=/tmp/cli-cache/go/ \
	    golang:1.14 /src/_cicd/functions.sh build

docker-cross-build: ##
	docker run --rm --volume $(shell dirname $(shell pwd)):/src --workdir /src \
	    --volume /tmp:/tmp \
	    --user $(shell id --user):$(shell id --group) \
	    --env HOME=/tmp/cli-cache/home \
	    --env GOPATH=/tmp/cli-cache/go \
	    golang:1.14 /src/_cicd/functions.sh cross-build

docker-build-shell: ##
	docker run --user $(shell id --user):$(shell id --group) -it \
	    --volume $(shell dirname $(shell pwd)):/src \
	    --volume /tmp:/tmp \
	    --env HOME=/tmp/cli-cache/home \
	    --env GOPATH=/tmp/cli-cache/go \
	    --workdir /src --rm \
	    golang:1.14 bash

docker-gcloud-build: ##
	docker run -it --volume $(shell cd .. && pwd):/src \
	    --env GCLOUD_SA_KEY=$(shell echo {GCLOUD_SA_KEY} | sed 's/{/\\{/g' | sed 's/}/\\}/g' | sed 's/,/\\,/g') \
	    --workdir /src --rm \
	    --volume ~/Downloads:/tmp/downloads \
	    --user $(shell id --user):$(shell id --group) \
	    --env HOME=/tmp/cli-cache/home \
	    google/cloud-sdk:289.0.0 /src/_cicd/functions.sh archive
docker-gcloud-shell: ##
	docker run -it --volume $(shell cd .. && pwd):/src \
	    --env HOME=/tmp/cli-cache/home \
	    --volume ~/Downloads:/tmp/downloads \
	    --workdir /src --rm \
	    google/cloud-sdk:289.0.0 bash
