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

KUBE_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

source "${KUBE_ROOT}/hack/lib/version.sh"
kube::version::get_version_vars

IMAGES=(kube-apiserver kube-controller-manager kube-proxy kube-scheduler)

K8S_DOCKER_REGISTRY="${KUBE_DOCKER_REGISTRY:-cr.d.xiaomi.net/containercloud}"
K8S_VERSION=$KUBE_GIT_VERSION

for image in "${IMAGES[@]}"
do
  k8s_docker_image=$K8S_DOCKER_REGISTRY/$image:$K8S_VERSION
  docker tag $K8S_DOCKER_REGISTRY/$image-amd64:$K8S_VERSION $k8s_docker_image
  echo "Upload $k8s_docker_image to docker registry"
  docker push $k8s_docker_image
done

echo "Upload all k8s images to xiaomi docker registry success~"
echo "Please see: https://cloud.mioffice.cn/products/docker/#/namespaceDetail?namespaceId=95&namespaceName=containercloud"
