#!/bin/bash
set -Eeuo pipefail
cd "$(dirname "$0")"

HOSTNAME=$(hostname -s)
SERVICE=test-jsa.test-jsa.svc
DOCKER=$( which podman &> /dev/null && echo podman || echo docker )

# certificates
install -m 0750 -d certs
$DOCKER run --rm -u $(id -u) -v $PWD/certs:/certs \
    -e CA_EXPIRE=3600 -e SSL_EXPIRE=3600 -e SSL_SUBJECT="$HOSTNAME" -e SSL_DNS="$HOSTNAME,$SERVICE" -e SSL_IP="127.0.0.1" \
    -e CA_KEY=ca.key -e CA_CERT=ca.crt \
    -e SSL_KEY=tls.key -e SSL_CERT=tls.crt -e SSL_CSR=tls.csr \
    paulczar/omgwtfssl &> /dev/null
CA=$(cat certs/ca.crt | base64 -w0)

# deploy
kubectl apply -f crds.yaml
kubectl apply -f namespace.yaml
kubectl apply -f rbac.yaml
kubectl create secret tls -n test-jsa test-jsa --cert=certs/tls.crt --key=certs/tls.key --dry-run=client -oyaml | kubectl apply -f -
cat hooks-${1:-kube}.yaml | sed "s/SERVERNAME/$HOSTNAME/g" | sed "s/CABUNDLE/$CA/g" | kubectl apply -f -
kubectl apply -f admissions.yaml
