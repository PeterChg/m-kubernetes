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

binaries=(kubeadm kube-aggregator kube-apiserver kube-controller-manager kubectl kubelet kube-proxy kube-scheduler)

K8S_VERSION=$KUBE_GIT_VERSION

for binary in "${binaries[@]}"
do
  echo "Upload $binary to FDS"
  bash ${KUBE_ROOT}/build/upload-to-fds.sh _output/bin/$binary release/$K8S_VERSION/bin/linux/amd64/$binary
done

PACKAGE=kubernetes-bin-linux-amd64-all.tar.gz
tar -zcvf $PACKAGE -C _output/bin/ kubeadm kube-aggregator kube-apiserver kube-controller-manager kubectl kubelet kube-proxy kube-scheduler
bash ${KUBE_ROOT}/build/upload-to-fds.sh ${PACKAGE} release/$K8S_VERSION/${PACKAGE}
rm -f ${PACKAGE}

PACKAGE=kubernetes-bin-linux-amd64-client.tar.gz
tar -zcvf $PACKAGE -C _output/bin/ kubeadm kubectl kubelet
bash ${KUBE_ROOT}/build/upload-to-fds.sh ${PACKAGE} release/$K8S_VERSION/${PACKAGE}
rm -f ${PACKAGE}

echo "Upload all k8s binaries to FDS success~"
echo "Please see: https://cloud.mioffice.cn/#/product/file-store/objects/kubernetes"
