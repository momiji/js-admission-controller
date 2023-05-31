#!/bin/bash
set -Eeuo pipefail
cd "$(dirname "$0")"

docker stop jsa-k3s --time 0 ||:
docker kill jsa-k3s ||:
sleep 1

# start k3s in docker
docker run --rm -d --name jsa-k3s --hostname jsa-k3s --privileged -p 6443:6443 -p 32000:32000 rancher/k3s:v1.27.2-k3s1 server --disable=traefik --disable=metrics-server --disable=local-storage --disable=coredns

# wait for config
i=60
while ((i-->1)) ; do
  sleep 1
  docker exec jsa-k3s kubectl config view | sed -n /DATA/p | tail -1 | grep -q . || continue
  break
done
[ $i -ne 0 ]
docker exec jsa-k3s kubectl config view --raw > k3s.config
export KUBECONFIG=k3s.config

# wait for node to be ready
i=60
while ((i-->1)) ; do
  sleep 1
  kubectl wait node jsa-k3s --for condition=Ready=True --timeout=90s || continue
  break
done
[ $i -ne 0 ]

# add registry
kubectl create deployment -n kube-system registry --image=registry --port 5000
kubectl create service loadbalancer -n kube-system registry --tcp 32000:5000

# all waits
kubectl wait deployment -n kube-system registry --for condition=Available=True --timeout=90s

# push image - if it fails, this might be issue on ipv6 enabled?
( cd .. && make local )

# run install
./install.sh
kubectl wait deployment -n test-jsa test-jsa --for condition=Available=True --timeout=90s

# rnu pods
echo
echo
echo "********************************************************************************"
echo "** PODS TEST"
echo "********************************************************************************"
echo
echo
./pods.sh
