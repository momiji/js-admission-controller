#!/bin/bash
set -Eeuo pipefail
cd "$(dirname "$0")"

kubectl delete pod -n test-jsa test-log --force 2> /dev/null ||:
kubectl delete pod -n test-jsa test-log-pending --force 2> /dev/null ||:
kubectl apply -f pods.yaml

sleep 1

echo
echo "# test-log annotations:"
kubectl get pod -n test-jsa test-log -o json | jq '.metadata.annotations | to_entries[] | [.key,.value] | join("=")' -cr | sed -n /^jsa/p

echo
echo "# test-log-pending annotations:"
kubectl get pod -n test-jsa test-log-pending -o json | jq '.metadata.annotations | to_entries[] | [.key,.value] | join("=")' -cr | sed -n /^jsa/p
