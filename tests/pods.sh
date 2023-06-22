#!/bin/bash
set -Eeuo pipefail
cd "$(dirname "$0")"

kubectl delete pod -n test-jsa test-pod --force 2> /dev/null ||:
kubectl delete pod -n test-jsa test-pod-pending --force 2> /dev/null ||:
kubectl delete deployment -n test-jsa test-deployment --force 2> /dev/null ||:
kubectl apply -f pods.yaml

sleep 1

echo
echo "# test-pod annotations:"
kubectl get pod -n test-jsa test-pod -o json | jq '.metadata.annotations | to_entries[] | [.key,.value] | join("=")' -cr | sed -n /^jsa/p

echo
echo "# test-pod-pending annotations:"
kubectl get pod -n test-jsa test-pod-pending -o json | jq '.metadata.annotations | to_entries[] | [.key,.value] | join("=")' -cr | sed -n /^jsa/p

echo
echo "# jsa CREATE logs"
kubectl get pods -n test-jsa | grep test-jsa- | awk '{print $1}' | xargs -i kubectl logs -n test-jsa {} | grep CREATE
