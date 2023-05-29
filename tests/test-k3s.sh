#!/bin/bash
set -Eeuo pipefail
cd "$(dirname "$0")"

docker stop jsa-k3s --time 0 ||:
docker kill jsa-k3s ||:

# start k3s in docker
docker run --rm -d --name jsa-k3s --hostname jsa-k3s --privileged -p 6443:6443 -p 32000:32000 rancher/k3s:v1.24.10-k3s1 server

# wait for node to be ready
i=10
while ((i-->0)) ; do
  sleep 1
  docker exec jsa-k3s kubectl wait node jsa-k3s --for condition=Ready=True --timeout=90s || continue
  docker exec jsa-k3s kubectl wait deployment -n kube-system metrics-server --for condition=Available=True --timeout=90s || continue
  break
done
[ $i -ne 0 ]

# extract kube config.yaml
docker exec jsa-k3s kubectl config view --raw > k3s.config

# add registry
kubectl create deployment -n kube-system registry --image=registry --port 5000
kubectl create service loadbalancer -n kube-system registry --tcp 32000:5000

# all waits
kubectl wait deployment -n kube-system registry --for condition=Available=True --timeout=90s
kubectl wait deployment -n kube-system metrics-server --for condition=Available=True --timeout=90s

# test kube config
export KUBECONFIG=k3s.config

# push image
( cd .. && make docker )
docker save localhost:32000/js-admissions-controller:latest | gzip > jsa.tgz
IP=$( docker inspect jsa-k3s | jq '.[].NetworkSettings.IPAddress' -r )
docker run --rm -it -v $PWD/jsa.tgz:/jsa.tgz:ro ananace/skopeo copy docker-archive:/jsa.tgz docker://$IP:32000/js-admissions-controller:latest --dest-tls-verify=false

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
