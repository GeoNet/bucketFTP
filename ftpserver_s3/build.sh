#!/bin/bash -e

# Builds Docker images for the arg list.  These must be project directories
# where this script is executed.
#
# Builds a statically linked executable and adds it to the container.
# Adds the assets dir from each project to the container e.g., origin/assets
# It is not an error for the assets dir to not exist.
# Any assets needed by the application should be read from the assets dir
# relative to the executable.
#
# usage: ./build.sh project [project]

# code will be compiled in this container
BUILD_CONTAINER=golang:1.7.0-alpine

DOCKER_TMP=$(mktemp -d)

VERSION='git-'`git rev-parse --short HEAD`

# The current working dir to use in GOBIN etc e.g., geonet-web
CWD=${PWD##*/}

# Assemble common resource for ssl and timezones from the build container
docker run --rm -v "$PWD":"$PWD"  ${BUILD_CONTAINER} \
	apk add --update ca-certificates tzdata; \
	mkdir -p "$PWD"/${DOCKER_TMP}/etc/ssl/certs; \
	mkdir -p "$PWD"/${DOCKER_TMP}/usr/share; \
	cp /etc/ssl/certs/ca-certificates.crt "$PWD"/${DOCKER_TMP}/etc/ssl/certs; \
	cp -Ra /usr/share/zoneinfo "$PWD"/${DOCKER_TMP}/usr/share

# Assemble common resource for user.
mkdir -p ${DOCKER_TMP}/etc
echo "nobody:x:65534:65534:Nobody:/:" > ${DOCKER_TMP}/etc/passwd

#for i in "$@"
#do
pkgname=${CWD}
#docker run -e "GOBIN=/usr/src/go/src/github.com/GeoNet/${CWD}/${DOCKER_TMP}" -e "GOPATH=/usr/src/go" -e "CGO_ENABLED=0" -e "GOOS=linux" -e "BUILD=$BUILD" --rm \
#    -v "$PWD":/usr/src/go/src/github.com/GeoNet/${CWD} \
#    -w /usr/src/go/src/github.com/GeoNet/${CWD} ${BUILD_CONTAINER} \
#    go install -a -ldflags "-X main.Prefix=${pkgname}/${VERSION}" -installsuffix cgo ./${pkgname}

# build the executable in the golang alpine container, mounting $PWD in the container so it builds in the local dir
#docker run -e GOPATH=/ -e "CGO_ENABLED=0" -e "GOOS=linux" -e "BUILD=$BUILD" --rm -v
#    "$PWD":/src/${pkgname} -w /src/${pkgname} golang:1.7.0-alpine go build
docker run -e "GOBIN=/docker_tmp" -e GOPATH=/ -e "CGO_ENABLED=0" -e "GOOS=linux" -e "BUILD=$BUILD" --rm \
    -v "$PWD":/src/${pkgname} -v "${DOCKER_TMP}":/docker_tmp -w /src/${pkgname} golang:1.7.0-alpine \
    go install -a -ldflags "-X main.Prefix=${pkgname}/${VERSION}" -installsuffix cgo .

cp Dockerfile ${DOCKER_TMP}
cd ${DOCKER_TMP}
docker build .

# TODO: add tagging here for easy pushing to AWS's ECR
## tag latest.  Makes it easier to test with compose.
#docker tag 862640294325.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:$VERSION 862640294325.dkr.ecr.ap-southeast-2.amazonaws.com/${i}:latest

# run with:  docker run --env-file env_test.list -p3000:21 <image_name>

rm -rf ${DOCKER_TMP}
