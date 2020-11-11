#!/bin/sh

set -e

[ ! -z "${FDS_ACCESS_KEY}" ] && FLAG="${FLAG} --ak ${FDS_ACCESS_KEY}"
[ ! -z "${FDS_SECRET_KEY}" ] && FLAG="${FLAG} --sk ${FDS_SECRET_KEY}"

BUCKET='kubernetes-bin'
MASTER_PKG=$(ls master*.tar.gz)
NODE_PKG=$(ls node*.tar.gz)
DIR='mi-snapshot'
fds ${FLAG} -m put -e cnbj1.fds.api.xiaomi.com -b ${BUCKET} -d ${MASTER_PKG}  -o ${DIR}/${MASTER_PKG}
fds ${FLAG} -m put -e cnbj1.fds.api.xiaomi.com -b ${BUCKET} -d ${NODE_PKG}  -o ${DIR}/${NODE_PKG}


URL="http://cnbj1-fds.api.xiaomi.net/${BUCKET}/${DIR}/${MASTER_PKG} & http://cnbj1-fds.api.xiaomi.net/${BUCKET}/${DIR}/${MASTER_PKG}"

echo "There is a new avaliable inner release: ${MASTER_PKG}"
echo "Download url is ${URL}"


rm -rf $MASTER_PKG $NODE_PKG
