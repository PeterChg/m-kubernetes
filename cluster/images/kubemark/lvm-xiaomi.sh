#!/bin/sh

# k8s fds http://git.n.xiaomi.com/containercloud/k8s-fds/blob/master/fds
DOCKER_NAME_PREFIX="k8s_"

usage() {
    echo "Invalid usage of fds provisioner CLI.. :" >> /tmp/k8s-lvm.log
    echo $@ >> /tmp/k8s-lvm.log
    echo "Invalid usage/options of fds provisioner CLI: ${@}"
    echo $@
}

err() {
    echo $* 1>&2
}

log() {
    echo $* >&1
}

attach() {
    echo "attach() called" >> /tmp/k8s-lvm.log
    log "{\"status\": \"Success\"}"
    exit 0
}

detach() {
    echo "detach() called" >> /tmp/k8s-lvm.log
    log "{\"status\": \"Success\"}"
    exit 0
}

domount() {
    echo "domount() called" >> /tmp/k8s-lvm.log
    MNTPATH=$1

    ID=$(echo $2|jq -r '.["kubernetes.io/pod.uid"]') # may podname or pod uuid
    SIZE=$(echo $2|jq -r '.["DiskSize"]') # disk size
    ID=`echo $ID | sed 's/-//g'` #remove -
    CNAME="${DOCKER_NAME_PREFIX}${ID}"
    echo "id = ${ID} size=${SIZE} cname=${CNAME}" >> /tmp/k8s-lvm.log
    # mkdir mount point
    mkdir -p ${MNTPATH} &>> /tmp/k8s-lvm.log
    # succ
    log "{\"status\": \"Success\", \"message\":\"${DOCKER_OUT}\"}"
    echo "mount success" >>  /tmp/k8s-lvm.log
    exit 0
}

unmount() {
    echo "unmount() called" >> /tmp/k8s-lvm.log
    local MNTPATH=$1
    rm -rf MNTPATH 
    log "{\"status\": \"Success\"}"
    echo "unmount success" >> /tmp/k8s-lvm.log
    exit 0
}

echo "--------------------------------------------------------------`date`------------------------------------" >> /tmp/k8s-lvm.log
echo $@ >> /tmp/k8s-lvm.log

op=$1

if [ "$op" = "init" ]; then
    log "{\"status\": \"Success\", \"capabilities\": {\"attach\": false}}"
    exit 0
fi

shift
case "$op" in
    # attach)
    # 	attach $*
    # 	;;
    # detach)
    # 	detach $*
    # 	;;
    mount)
        domount $*
        ;;
    unmount)
        unmount $*
        ;;
    *)
        log "{ \"status\": \"Not supported\" }"
        echo "${op} not support" >> /tmp/k8s-lvm.log
        exit 0
esac
exit 1
