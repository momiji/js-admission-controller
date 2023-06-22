#!/bin/bash
set -Eeuo pipefail
cd "$(dirname "$0")"

[ -z "${DOCKER:-}" ] && DOCKER=$( which podman &> /dev/null && echo podman || echo docker )
[ -z "${REGISTRY_PORT:-}" ] && REGISTRY_PORT=$( python3 -c 'import socket; s=socket.socket(); s.bind(("", 0)); print(s.getsockname()[1]); s.close()' )

echo "container cmd: sudo $DOCKER"
echo "registry port: $REGISTRY_PORT"
export DOCKER
export REGISTRY_PORT

echo
echo "********************************************************************************"
echo "** START K3S"
echo "********************************************************************************"
echo

sudo $DOCKER stop jsa-k3s --time 0 2> /dev/null ||:
sudo $DOCKER kill jsa-k3s 2> /dev/null ||:
sudo $DOCKER rm jsa-k3s 2> /dev/null ||:
sleep 1

# start k3s in docker
[ -f k3s-registries.yaml ] || :> k3s-registries.yaml
sudo $DOCKER run --rm -d --name jsa-k3s --hostname localhost --privileged -p 6443:6443 -p $REGISTRY_PORT:32000 -v $PWD/k3s-registries.yaml:/etc/rancher/k3s/registries.yaml rancher/k3s:v1.27.2-k3s1 server --disable=traefik --disable=metrics-server --disable=local-storage --disable=coredns

# wait for config
i=60
while ((i-->1)) ; do
  sleep 1
  sudo $DOCKER exec jsa-k3s kubectl config view | sed -n /DATA/p | tail -1 | grep -q . || continue
  break
done
[ $i -ne 0 ]
sudo $DOCKER exec jsa-k3s kubectl config view --raw > k3s.config
export KUBECONFIG=k3s.config
echo "k3s.config written"

# wait for node to be ready
i=60
while ((i-->1)) ; do
  sleep 1
  kubectl wait node localhost --for condition=Ready=True --timeout=90s || continue
  break
done
[ $i -ne 0 ]

echo
echo "********************************************************************************"
echo "** INSTALL REGISTRY"
echo "********************************************************************************"
echo

# add registry
kubectl create deployment -n kube-system registry --image=registry --port 5000
kubectl create service loadbalancer -n kube-system registry --tcp 32000:5000

# all waits
kubectl wait deployment -n kube-system registry --for condition=Available=True --timeout=90s
sleep 2

echo
echo "********************************************************************************"
echo "** BUILD JSA"
echo "********************************************************************************"
echo

# push image - if it fails, this might be issue on ipv6 enabled?
( cd .. && make local )

echo
echo "********************************************************************************"
echo "** INSTALL JSA"
echo "********************************************************************************"
echo

# run install
./install.sh
kubectl wait deployment -n test-jsa test-jsa --for condition=Available=True --timeout=90s

# rnu pods
echo
echo "********************************************************************************"
echo "** TEST PODS"
echo "********************************************************************************"
echo
./pods.sh

echo
echo "********************************************************************************"
echo "** STOP K3S"
echo "********************************************************************************"
echo
echo "To remove jsa-k3s, run: sudo $DOCKER stop jsa-k3s"
echo
