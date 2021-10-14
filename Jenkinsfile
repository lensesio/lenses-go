#!/usr/bin/env groovy

pipeline {
    agent {
        label 'docker && ephemeral'
    }

    options {
        ansiColor('xterm')
        copyArtifactPermission('*')
    }

    parameters {
        booleanParam(name: 'DEBUG_BUILD', defaultValue: true,
            description: 'Enable verbose output')
        booleanParam(name: 'DOCKER_BUILD_CACHE', defaultValue: true,
            description: 'If set then use build cache')
    }

    environment {
        // Build stage variables
        DOCKER_GO_IMG = 'golang:1.14'
        DOCKER_GO_CACHE = '/tmp/cli-cache'
        DOCKER_GO_ARGS = "--volume /tmp:/tmp " +
          "--env HOME=${DOCKER_GO_CACHE}/home " +
          "--env GOPATH=${DOCKER_GO_CACHE}/go"

        // Archive stage variables
        GCLOUD_PROJECT = 'lenses-ci'

        // Dev. vars.
        // GCLOUD_SA_KEY_PATH = credentials("gcloud-dev")
        // GCLOUD_PROJECT = 'k8-engine'
        GCLOUD_DOCKER_IMAGE = 'google/cloud-sdk:360.0.0-slim'
        GCLOUD_DOCKER_ARGS = '--env HOME=/tmp'
        GCLOUD_BUCKET = 'cli'
        GCLOUD_BUCKET_DEVELOPMENT = 'lenses-artifacts-development'
        GCLOUD_BUCKET_RELEASE = 'lenses-artifacts-release'

    }

    stages {
        // Sets necessary variables and may print additional debug info
        stage('Setup'){
            steps {
                script {
                    // Acceptable release mode tags: v3.1.1, v3.1.2-alpha.0, v3.1.2-beta,
                    // v3.1.2-beta.1, v3.1.2-rc.2 v3.1.1+sha.d0a63755a, v3.1.2-beta+20200421
                    env.LENSES_CLI_CUR_VERSION = sh(script: 'git describe --tags', returnStdout: true).trim()
                    // LENSES_CLI_CUR_VERSION = 'v0.0.0'

                    env.BUILD_MODE = 'development'

                    SEMVER_REGEX = /v[0-9]\.[0-9]\.[0-9]+(-(alpha|beta|rc)(\.[0-9]+)?)?(\+[A-Za-z0-9.]+)?/
                    if (LENSES_CLI_CUR_VERSION ==~ SEMVER_REGEX) {
                        env.BUILD_MODE = 'release'
                        // For the release version we strip the prefix v from tags
                        env.LENSES_CLI_CUR_VERSION = LENSES_CLI_CUR_VERSION - ~/^v/
                    }

                    // TO-DO
                    // if (env.BUILD_MODE == 'release' || params.DOCKER_BUILD_CACHE == false){
                    //     echo "Cleaning up docker build cache"
                    //     // sh 'sudo rm -rf /tmp/cli-cache'
                    // }

                    echo "Set BUILD_MODE to: ${env.BUILD_MODE}"

                    switch(BUILD_MODE) {
                        case "development":
                            env.GCLOUD_BUCKET_PATH = "${GCLOUD_BUCKET_DEVELOPMENT}/$GCLOUD_BUCKET/${GIT_COMMIT}"
                            break
                        case "release":
                            env.GCLOUD_BUCKET_PATH = "${GCLOUD_BUCKET_RELEASE}/$GCLOUD_BUCKET/${LENSES_CLI_CUR_VERSION}"
                            break
                        default:
                            error("Build mode unknown: ${BUILD_MODE}")
                            break
                    }

                    sh 'printenv | sort | tee environment'
                    echo "Build cause: ${currentBuild.getBuildCauses()}"

                    archiveArtifacts artifacts: 'environment'
                }
            }
        }

        // This stage is responsible for:
        // - download necessary go dependencies (if not found in cache)
        // - lint the code according to Go standars
        // - run unit tests
        // - cross build and produce the binaries for all supported platforms
        // Also it produces this necessary stash:
        // - binaries
        // And these artifacts:
        // - cover.out
        // - cover.html
        stage('Build'){
            agent {
                docker {
                    image "${DOCKER_GO_IMG}"
                    args "${DOCKER_GO_ARGS}"
                    reuseNode true
                }
            }
            steps {
                dir ('bin') { deleteDir() }

                sh '_cicd/functions.sh setup'
                sh '_cicd/functions.sh lint'
                sh '_cicd/functions.sh format-check'
                // The following only builds for the platform found at runtime
                // It's used for quick iterations during local developemnt
                // sh '_cicd/functions.sh build'
                sh '_cicd/functions.sh test'
                sh '_cicd/functions.sh cross-build'

                dir ('bin') {
                    stash name: 'binaries'
                    deleteDir()
                }

                archiveArtifacts artifacts: 'cover.out'
                archiveArtifacts artifacts: 'cover.html'
            }
        }

        // Collect and upload artifacts to Jenkins and Google cloud storage
        // For developers: any new artifact from any stage should be added
        // to the bottom of this stage.
        // Requires this stash:
        // - binaries
        // Creates these artifacts, where SUFFIX is empty for development mode,
        // and VERSION for release mode:
        // - lenses-cli-darwin-amd64${ARCHIVE_SUFFIX}.tar.gz
        // - lenses-cli-darwin-amd64${ARCHIVE_SUFFIX}.tar.gz.sha256
        // - lenses-cli-linux-amd64${ARCHIVE_SUFFIX}.tar.gz
        // - lenses-cli-linux-amd64${ARCHIVE_SUFFIX}.tar.gz.sha256
        // - lenses-cli-windows-amd64${ARCHIVE_SUFFIX}.zip
        // - lenses-cli-windows-amd64${ARCHIVE_SUFFIX}.zip.sha256
        stage('Archive and Upload'){
            agent {
                dockerfile {
                    filename 'Dockerfile'
                    dir '_cicd/gcloud'
                    additionalBuildArgs  "--build-arg GCLOUD_DOCKER_IMAGE=${GCLOUD_DOCKER_IMAGE}"
                    args "${GCLOUD_DOCKER_ARGS}"
                    reuseNode true
                }
            }
            // Env. vars. for local development
            // environment {
            //   GCLOUD_BUCKET_PATH = "lenses-cli-dev/lenses-artifacts-development/cli/${GIT_COMMIT}"
            // }
            steps {
                dir ('archives') { deleteDir() }
                dir ('archives') {
                    unstash 'binaries'
                    script {
                        // For the release mode, we add the version to the archive filename
                        env.ARCHIVE_SUFFIX = ''
                        if ( BUILD_MODE == 'release' ) {
                            env.ARCHIVE_SUFFIX = '-' + LENSES_CLI_CUR_VERSION
                        }
                    }
                    sh '${WORKSPACE}/_cicd/functions.sh archive ${BUILD_MODE} ${LENSES_CLI_CUR_VERSION}'

                    archiveArtifacts artifacts: "lenses-cli-darwin-amd64${ARCHIVE_SUFFIX}.tar.gz"
                    archiveArtifacts artifacts: "lenses-cli-darwin-amd64${ARCHIVE_SUFFIX}.tar.gz.sha256"
                    archiveArtifacts artifacts: "lenses-cli-linux-amd64${ARCHIVE_SUFFIX}.tar.gz"
                    archiveArtifacts artifacts: "lenses-cli-linux-amd64${ARCHIVE_SUFFIX}.tar.gz.sha256"
                    archiveArtifacts artifacts: "lenses-cli-windows-amd64${ARCHIVE_SUFFIX}.zip"
                    archiveArtifacts artifacts: "lenses-cli-windows-amd64${ARCHIVE_SUFFIX}.zip.sha256"
                }

                dir ('upload-artifacts') { deleteDir() }
                dir ('upload-artifacts') {
                    copyArtifacts (
                        projectName: env.JOB_NAME,
                        selector: specific(env.BUILD_NUMBER)
                    )

                    // Secret used: Lenses CI G2/GCloud lenses-ci jenkins account for artifact deployment
                    withCredentials([
                        file(credentialsId: 'bb733438-f2d7-48a4-9465-4d5152ac4247', variable: 'GLOUD_KEY_PATH')
                    ]) { // Activate GCloud Account
                        sh '${WORKSPACE}/_cicd/functions.sh activateGCloudAccount "${GCLOUD_PROJECT}"'
                    }

                    echo 'Uploading artifacts'
                    sh '${WORKSPACE}/_cicd/functions.sh gcloudUploadArtifacts "*" "${GCLOUD_BUCKET_PATH}"'

                    sh '${WORKSPACE}/_cicd/functions.sh revokeGCloudAccount'
                    deleteDir()

                    echo 'You can find everything under:'
                    echo 'https://console.cloud.google.com/storage/browser/' + env.GGLOUD_BUCKET_PATH
                }
            }
        }

        // Build and tag the docker image producing the following tags at registry:
        // - eu.gcr.io/lenses-ci/lenses-cli-${BRANCH_NAME}
        // - eu.gcr.io/lenses-ci/lenses-cli-${LENSES_CLI_CUR_VERSION}
        // Note: When on PR then BRANCH_NAME is in the form of "PR-x"
        stage('Build docker'){
            agent {
                docker {
                    image "${GCLOUD_DOCKER_IMAGE}"
                    args "${GCLOUD_DOCKER_ARGS}"
                    reuseNode true
                }
            }
            steps {
                dir ('build-docker') { deleteDir() }
                dir ('build-docker') {
                    unstash 'binaries'

                    // Secret used: Lenses CI G2/GCloud lenses-ci jenkins account for artifact deployment
                    withCredentials([
                        file(credentialsId: 'bb733438-f2d7-48a4-9465-4d5152ac4247', variable: 'GLOUD_KEY_PATH')
                    ]) { // Activate GCloud Account
                        sh '${WORKSPACE}/_cicd/functions.sh activateGCloudAccount "${GCLOUD_PROJECT}"'
                    }

                    // Convert branch name to docker tag
                    script {
                        env.DOCKER_BRANCH_TAG = sh (script: "${WORKSPACE}/_cicd/functions.sh branchNameToDockerTag $BRANCH_NAME", returnStdout: true).trim()
                    }
                    // Building image
                    sh '${WORKSPACE}/_cicd/functions.sh build-docker-img eu.gcr.io/${GCLOUD_PROJECT}/lenses-cli ${LENSES_CLI_CUR_VERSION} ${DOCKER_BRANCH_TAG}'

                    sh '${WORKSPACE}/_cicd/functions.sh revokeGCloudAccount'
                    deleteDir()
                }
            }
        }
        // TO-DO - maybe do a docker run version and compare it ?
        // stage('Test docker'){
        //   steps {
        //   }
        // }


        // Remove build artifacts from workspace
        stage('Teardown'){
            steps {
                sh '_cicd/functions.sh clean'
            }
        }
    }
}

// Jenkins job description
/*
<h2>CICD generation 2 of lenses-go </h2>
<h3>Description</h3>
<p>
    Lenses CLI and go library. Some branches (master and release) are *automatically synced to public*.
</p>
<p>
    <b>Build artifacts</b>: <a href="https://console.cloud.google.com/storage/browser?forceOnBucketsSortingFiltering=false&project=lenses-ci">Google cloud storage bucket</a><br>
    <b>Docker images (internal)</b>: <a href="https://console.cloud.google.com/gcr/images/lenses-ci/EU/lenses-cli?project=lenses-ci&gcrImageListsize=30">Google container registry</a>
</p>
<hr>
<h3>For developers</h3>
<p>
    <b>Repo. name</b>: lenses-go <br>
    <b>Repo URL</b>: <a href="https://github.com/lensesio-dev/lenses-go/">lensesio-dev@Github</a><br>
    <b>Maintainer</b>: Dean
</p>
*/
