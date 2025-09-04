#!/bin/sh

docker pull alpine:3.22.1
docker pull stefanprodan/podinfo:6.9.1

ocm add componentversions --create --file ocm-transfer/artdeployser1 ../ocm-samples/sample-service-1.yaml
ocm add componentversions --create --file ocm-transfer/artdeployser2 ../ocm-samples/sample-service-2.yaml
ocm add componentversions --create --file ocm-transfer/vector1 ../ocm-samples/vector.yaml

ocm transfer ctf ./ocm-transfer/artdeployser1 https://registry.kdenv.lab/sample-project --overwrite
ocm transfer ctf ./ocm-transfer/artdeployser2 https://registry.kdenv.lab/sample-project --overwrite
ocm transfer ctf ./ocm-transfer/vector1 https://registry.kdenv.lab/sample-project --overwrite