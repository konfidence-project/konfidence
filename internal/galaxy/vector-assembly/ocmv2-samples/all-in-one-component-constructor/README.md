# example 2 - all in one vector

**Note::** this readme is just a scratchpad. It is not a finished documentation.

**Create a lokal CTF archiv** (this is optional)
```shell
./ocm-v2 add cv \
  --loglevel debug \
  --config ocm-config.yaml \
  --constructor vector-component-constructor.yaml \
  --repository ./vector-bundle
```

**Transfer the bundle to OCI Registry**
```shell
 ./ocm-v2 transfer component-version \
  --loglevel debug \
  --config ocm-config.yaml \
  ./vector-bundle//konfidence.cloud/a-fancy-application/vector:1.0.0 \
  https://konfidence.common.repositories.cloud.sap/ocmv2-sandbox-alex-examples 
```

**Create and transfer to OCI Registry - all in one step**
```shell
 ./ocm-v2 add-and-transfer cv \
  --loglevel debug \
  --config ocm-config.yaml \
  --constructor vector-component-constructor.yaml \
  https://konfidence.common.repositories.cloud.sap/ocmv2-sandbox-alex-examples 
```

**Get Vector from OCI**
```shell
./ocm-v2 get cv \
 --config ocm-config.yaml \
 --recursive \
 -o json \
 https://konfidence.common.repositories.cloud.sap/ocmv2-sandbox-alex-examples//konfidence.cloud/a-fancy-application/vector:1.0.0 | jq
```

## Problem: No componentReferences are included. Why?
```shell
./ocm-v2 get cv \
  --config ocm-config.yaml \
  -o json \
  https://konfidence.common.repositories.cloud.sap/ocmv2-sandbox-alex-examples//konfidence.cloud/a-fancy-application/vector:1.0.0 | jq
[
  {
    "meta": {
      "schemaVersion": "v2"
    },
    "component": {
      "name": "konfidence.cloud/a-fancy-application/vector",
      "version": "1.0.0",
      "repositoryContexts": null,
      "provider": "konfidence",
      "resources": null,
      "sources": null,
      "componentReferences": null
    }
  }
]
```
-> Is *transfer* the archive or *building the archive* the problem? 
Check lokal archive first!
```shell
 ./ocm-v2 get cv ./vector-bundle -o json | jq
[
  {
    "meta": {
      "schemaVersion": "v2"
    },
    "component": {
      "name": "konfidence.cloud/a-fancy-application/vector",
      "version": "1.0.0",
      "repositoryContexts": null,
      "provider": "konfidence",
      "resources": null,
      "sources": null,
      "componentReferences": null
    }
  }
]
```
-> Creating the archive is the problem. Let's check the `./ocm-v2 add` command.
-> It is not possible to create multiple components in one file and separated by `---` (like k8s does).

-> It is not possible to transfer all componentes at once. First transfer the leaves than the root-component
```shell
# 1. Sub-Component A
./ocm-v2 transfer cv \
  --config ocm-config.yaml \
  ./vector-bundle//konfidence.cloud/somewhere/vector-artifact-a:1.0.0 \
  https://konfidence.common.repositories.cloud.sap/ocmv2-sandbox-alex-examples

# 2. Sub-Component B
./ocm-v2 transfer cv \
  --config ocm-config.yaml \
  ./vector-bundle//konfidence.cloud/somewhere/vector-artifact-b:1.0.0 \
  https://konfidence.common.repositories.cloud.sap/ocmv2-sandbox-alex-examples

# 3. Root-Component (Vector)
./ocm-v2 transfer cv \
  --config ocm-config.yaml \
  ./vector-bundle//konfidence.cloud/a-fancy-application/vector:1.0.0 \
  https://konfidence.common.repositories.cloud.sap/ocmv2-sandbox-alex-examples
```


# local setup
Zot registry runs on localhost with port 5100 (`curl http://localhost:5100/v2/_catalog`). See `docker-compose.yaml` for more details.

1. new config `local-ocm-config.yaml` was created
2. Create new component version
```
./ocm-v2 add component-version \
  --loglevel debug \
  --config local-ocm-config.yaml \
  --repository "http://localhost:5100" \
  --constructor vector-component-constructor.yaml
```