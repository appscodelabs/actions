#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

IMG=cluster-tool
TAG=v1
RESTIC_VERSION=${RESTIC_VERSION:-0.9.4}
DOCKER_REGISTRY=${DOCKER_REGISTRY:-appscodeci}
REPO_ROOT=$GOPATH/src/github.com/appscodelabs/actions

build() {
  pushd $REPO_ROOT
  docker build -t $DOCKER_REGISTRY/$IMG:$TAG . -f ./cluster-tool/hack/docker/Dockerfile --build-arg RESTIC_VERSION=$RESTIC_VERSION
  popd
}

push(){
    docker push $DOCKER_REGISTRY/$IMG:$TAG
}

case $1 in
    "build")
        build
        ;;
    "push")
        push
        ;;
    *)
        echo "invalid arguments"
esac
