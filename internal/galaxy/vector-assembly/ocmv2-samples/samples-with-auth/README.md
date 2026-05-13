## Quick Start Guide

Follow these steps in order to set up a complete local development environment:

### Step 1: Start the Local OCI Registry

Start the Zot registry using Docker Compose:

```bash
cd ../docker
docker-compose up -d
```

This will start the Zot registry on `http://localhost:5100` with basic authentication (`user:user`).

**Verify the registry is running:**
[http://localhost:5100/home](http://localhost:5100/home)

### Step 2: Setup the Local Registry with OCM Components

Populate the registry with sample OCM component versions:

```bash
./setup-local-registry.sh
```

This script will add the following components to the registry:
- `konfidence.cloud/services/service1` (v1.5.2)
- `konfidence.cloud/services/service2` (v1.2.0)
- `konfidence.cloud/services/service3` (v4.1.0)

### Step 3: Setup Kubernetes Resources

Apply the namespace, secrets, and configuration to your cluster:

```bash
cd samples
./setup-cluster.sh
```

This creates:
- **Namespace**: `konfidence-system`
- **Secret**: `registry-credentials` (for accessing localhost:5100)

**Verify the resources:**
```bash
kubectl get all,secret -n konfidence-system
```

### Step 4: Install CRDs

Install the Custom Resource Definitions (CRDs) to your Kubernetes cluster:
Clone the repository [https://github.com/konfidence-project/crds](https://github.com/konfidence-project/crds) if you haven't already, and navigate to the project root:
```bash
make install
```

### Step 5: Start the Controller

Run the controller locally:

```bash
cd ../../
make run
```

### Step 6: Apply a VectorTemplate

Apply one of the sample VectorTemplates to test the controller:

```bash
kubectl apply -f samples/vectortemplate_base.yaml
```

Or apply the more complex sample:

```bash
kubectl apply -f ocmv2-samples/vectortemplate_sample.yaml
```

**Watch the VectorTemplate status:**
```bash
kubectl get vectortemplate -n default -w
```

**Check the controller logs:**
```bash
# If running locally with `make run`, check the terminal output
# If deployed to cluster:
kubectl logs -n konfidence-system -l control-plane=controller-manager -f
```
