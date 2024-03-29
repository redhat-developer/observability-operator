#!/bin/bash -ex
#
# Copyright (c) 2018 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# This script builds and deploys the Observability Operator. In order to
# work, it needs the following variables defined in the CI/CD configuration of
# the project:
#
# QUAY_USER - The name of the robot account used to push images to
# 'quay.io', for example 'openshift-unified-hybrid-cloud+jenkins'.
#
# QUAY_TOKEN - The token of the robot account used to push images to
# 'quay.io'.
#
# The machines that run this script need to have access to internet, so that
# the built images can be pushed to quay.io.

# Set the variable required to login and push images to the registry
export QUAY_USER=${QUAY_USER_NAME:-$RHOAS_QUAY_USER}
export QUAY_TOKEN=${QUAY_USER_PASSWORD:-$RHOAS_QUAY_TOKEN}

export DOCKER_CONFIG="${PWD}/.docker"
mkdir -p "${DOCKER_CONFIG}"

OPM_VERSION=v1.26.2
OPM_PLATFORM=linux-amd64
OPM_DOWNLOAD_URL="https://github.com/operator-framework/operator-registry/releases/download/$OPM_VERSION/$OPM_PLATFORM-opm"

mkdir -p ${PWD}/.bin

# Download OPM
if [ ! -f "${PWD}/.bin/opm" ]; then
  curl -Lo "${PWD}/.bin/opm" -k $OPM_DOWNLOAD_URL
  chmod +x "${PWD}/.bin/opm"
fi

export OPM=${PWD}/.bin/opm

# Print OPM version
$OPM version

make docker-login
make docker-build
make docker-push
make bundle-build
make bundle-push
make index-add
make index-build
make index-push
