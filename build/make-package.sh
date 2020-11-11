
#!/bin/sh

set -e

VERSION=${CI_COMMIT_TAG}

# build bin 
KUBE_BUILD_PLATFORMS=linux/amd64 make all WHAT=cmd/kube-scheduler 
KUBE_BUILD_PLATFORMS=linux/amd64 make all WHAT=cmd/kube-controller-manager 
KUBE_BUILD_PLATFORMS=linux/amd64 make all WHAT=cmd/kube-apiserver
KUBE_BUILD_PLATFORMS=linux/amd64 make all WHAT=cmd/kubectl 
KUBE_BUILD_PLATFORMS=linux/amd64 make all WHAT=cmd/kubelet 
KUBE_BUILD_PLATFORMS=linux/amd64 make all WHAT=cmd/kube-proxy


# tar package
bin_dir="_output/bin"
tar -zcvf master-bin-${VERSION}.tar.gz -C $bin_dir/ kube-scheduler kube-controller-manager kube-apiserver kubectl
tar -zcvf node-bin-${VERSION}.tar.gz -C $bin_dir/ kubelet kube-proxy

