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

# This script builds and deploys the Observability Operator Index using the OPM CLI.

OPM_VERSION=v1.17.5
OPM_PLATFORM=linux-amd64
OPM_DOWNLOAD_URL="https://github.com/operator-framework/operator-registry/releases/download/$OPM_VERSION/$OPM_PLATFORM-opm"

mkdir -p ${PWD}/.bin

# Download OPM
if [ ! -f "${PWD}/.bin/opm" ]; then
  curl -Lo "${PWD}/.bin/opm" -k $OPM_DOWNLOAD_URL
  chmod +x "${PWD}/.bin/opm"
fi

# Check if bundle list is up to date
# If not append the bundle image for the current version
if ! grep -q "$BUNDLE_IMG" "bundle_history.txt"; then
  printf "%s\n" "$BUNDLE_IMG" >> bundle_history.txt
fi

OPM=${PWD}/.bin/opm

# Print OPM version
$OPM version

# Prepare the command to include all previous bundles
BUNDLES=""
while read -r bundle; do
  BUNDLES+="--bundles ${bundle} "
done <bundle_history.txt

$OPM index add --build-tool=docker $BUNDLES --tag $INDEX_IMG
