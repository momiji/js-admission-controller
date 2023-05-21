#!/bin/bash
set -Eeuo pipefail
cd "$(dirname "$0")"

kubectl delete pod -n test-jsa test-log --force 2> /dev/null
kubectl apply -f pod.yaml

echo
#sleep 1
kubectl get pod -n test-jsa test-log -o yaml | yq -y .metadata.annotations
