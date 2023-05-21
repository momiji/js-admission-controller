#!/bin/bash
set -Eeuo pipefail
cd "$(dirname "$0")"

HOSTNAME=devlinux
SERVICE=jsadmissions.kube-jsadmissions.svc
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
kubectl create secret tls -n kube-jsadmissions jsadmission-tls --cert=certs/tls.crt --key=certs/tls.key --dry-run=client -oyaml | kubectl apply -f -
for f in `ls rbac*.yaml` ; do
    kubectl apply -f "$f"
done
kubectl apply -f deploy.yaml
for f in `ls hooks*.yaml` ; do
    cat "$f" | sed "s/CABUNDLE/$CA/g" | kubectl apply -f -
done
