#!/usr/bin/env bash

create_failed=yes
trap shoot_deletion EXIT

function shoot_deletion {
  go test -mod=vendor -timeout=60m ./test/system/shoot_deletion \
    --v -ginkgo.v -ginkgo.progress \
    -kubecfg=$KUBECONFIG \
    -project-namespace=garden-local \
    -shoot-name=e2e-local

  if [ $create_failed = yes ] ; then
    exit 1
  fi
}

go test -mod=vendor -timeout=60m ./test/system/shoot_creation \
  --v -ginkgo.v -ginkgo.progress \
  -kubecfg=$KUBECONFIG \
  -project-namespace=garden-local \
  -shoot-name=e2e-local \
  -annotations=shoot.gardener.cloud/cleanup-infrastructure-resources-grace-period-seconds=0 \
  -k8s-version=1.21.0 \
  -cloud-profile=local \
  -seed=local \
  -region=local \
  -secret-binding=local \
  -provider-type=local \
  -networking-type=local \
  -workers-config-filepath=<(cat <<EOF
- name: local
  machine:
    type: local
  cri:
    name: containerd
  maximum: 1
  minimum: 1
  maxSurge: 1
  maxUnavailable: 0
EOF
) \
  -shoot-template-path=<(cat <<EOF
apiVersion: core.gardener.cloud/v1beta1
kind: Shoot
EOF
)

if [ $? = 0 ] ; then
  create_failed=no
fi
