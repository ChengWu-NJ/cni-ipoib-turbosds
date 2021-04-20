#!/bin/sh

set -e

CNI_BIN_DIR="/host/opt/cni/bin"
CNI_BIN_FILE="/usr/bin/cni-ipoib-ts"

CNI_CFG_DIR="/host/etc/cni/net.d"
CNI_CFG_FILENAME="10-ipoib-ts.conflist"

cp -f ${CNI_BIN_FILE} ${CNI_BIN_DIR} <<<y
mkdir -p ${CNI_CFG_DIR}
cat > ${CNI_CFG_DIR}/${CNI_CFG_FILENAME} <<EOF
{
  "name": "k8s-pod-network",
  "cniVersion": "0.3.1",
  "plugins": [
    {
      "type": "ipoib-ts",
      "network": "172.24.0.0/13",
      "ibdev": "ib0"
    }
  ]
}
EOF

echo "Finish deploying plugin"

# Sleep forever.
# sleep 2147483647