#!/usr/bin/env bash

# Copyright 2014 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# shellcheck disable=SC2034 # Variables sourced in other scripts.

# Common utilities, variables and checks for all build scripts.
set -o errexit
set -o nounset
set -o pipefail

# Unset CDPATH, having it set messes up with script import paths
unset CDPATH

if [ $# != 2 ]; then
  echo "Usage: $0 src dst"
  exit
fi

SRC=$1
DST=$2
FLAG=""

set -e
[ ! -z "${FDS_ACCESS_KEY}" ] && FLAG="${FLAG} --ak ${FDS_ACCESS_KEY}"
[ ! -z "${FDS_SECRET_KEY}" ] && FLAG="${FLAG} --sk ${FDS_SECRET_KEY}"

echo "FDS flags: $FLAG"

BUCKET='kubernetes'
fds ${FLAG} -m put -e cnbj1.fds.api.xiaomi.com -b ${BUCKET} -d $SRC  -o $DST

echo "There is a new available inner release: $DST"
URL="http://cnbj1-fds.api.xiaomi.net/${BUCKET}/${DST}"
echo "Download url is ${URL}"
