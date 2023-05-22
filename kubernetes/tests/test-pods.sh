#!/bin/bash
set -Eeuo pipefail
cd "$(dirname "$0")"

kubectl delete pod -n test-jsa test-log --force 2> /dev/null ||:
kubectl apply -f pods.yaml

echo
#sleep 1
kubectl get pod -n test-jsa test-log -o json | jq '.metadata.annotations | to_entries[] | [.key,.value] | join("=")' -cr | sed -n /^jsa/p
