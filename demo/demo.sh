#!/bin/bash
# demo.sh - Control demo saladcloud-virtual-kublet instance in K8s cluster

# Typically a .env file similar to the following is used to configure the virtual kubelet
# export LOG_LEVEL=INFO
# export SALAD_API_KEY=<api-key>
# export SALAD_ORGANIZATION_NAME=salad
# export SALAD_PROJECT_NAME=demo
# export NAMESPACE=saladcloud-demo
# export NODE_NAME=${SCE_PROJECT_NAME}
# export CHART_NAME=oci://ghcr.io/saladtechnologies/virtual-kubelet-saladcloud-chart

TOP_DIR=$(cd $(dirname "$0") && pwd)

# Script lives in a subdirectory of the top repo dir
pushd $TOP_DIR/.. >/dev/null

IMAGE_TAG=${IMAGE_TAG:-latest}
NAMESPACE=${NAMESPACE:-saladcloud-demo}
NODE_NAME=${NODE_NAME:-demo}
CHART_NAME=${CHART_NAME:-chart/virtual-kubelet-saladcloud-chart}

if [[ "$1" == "start" ]]; then
    shift
    CMD=" \
    helm install \
      --create-namespace \
      --namespace ${NAMESPACE} \
      --set salad.organizationName=${SALAD_ORGANIZATION_NAME} \
      --set salad.projectName=${SALAD_PROJECT_NAME} \
      --set salad.imageTag=${IMAGE_TAG} \
      --set salad.nodeName=${NODE_NAME}-vk \
      --version 0.0.0 \
      ${NODE_NAME} \
      ${CHART_NAME}
    echo $CMD
    $CMD \
      --set salad.apiKey=${SALAD_API_KEY} \
      --set salad.logLevel=${LOG_LEVEL:-info}

elif [[ "$1" == "stop" ]]; then
    shift
    helm uninstall \
      --namespace ${NAMESPACE} \
      ${NODE_NAME}
elif [[ "$1" == "status" ]]; then
    echo ""
    echo "$ kubectl get node"
    kubectl get node
    echo ""
    echo "$ kubectl --namespace ${NAMESPACE} get pod"
    kubectl --namespace ${NAMESPACE} get pod
elif [[ "$1" == "logs" ]]; then
    podname=$(kubectl --namespace ${NAMESPACE} get pod|awk "/${NODE_NAME}/ { print \$1 }")
    kubectl --namespace ${NAMESPACE} logs $podname
elif [[ "$1" == "apply" ]]; then
    # Launch the qr-code
    kubectl apply -f demo/qr.yaml
elif [[ "$1" == "delete" ]]; then
    # Delete the qr-code
    kubectl delete -f demo/qr.yaml
fi

popd >/dev/null
