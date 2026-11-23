package setup

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, directory, name, data string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func validOptions(t *testing.T) Options {
	t.Helper()
	directory := t.TempDir()
	kubeconfig := writeFile(t, directory, "kubeconfig", `
apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: https://127.0.0.1:1
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
users:
- name: test
  user:
    token: sensitive-token
`)
	return Options{
		Kubeconfig:        kubeconfig,
		ArtifactReference: "registry.example.com/conformance/app:1.0.0",
		DeploymentTargetSpec: writeFile(t, directory, "target.yaml", `
artifactType: app.example.com
parameters:
  kind: Secret
  name: target-credentials
`),
	}
}

func TestLoad(t *testing.T) {
	options := validOptions(t)
	directory := t.TempDir()
	options.TargetSetup = []string{
		writeFile(t, directory, "secret.yaml", `
apiVersion: v1
kind: Secret
metadata:
  name: target-credentials
stringData:
  password: sensitive-password
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: configuration
`),
		writeFile(t, directory, "extra.json", `{"apiVersion":"v1","kind":"Secret","metadata":{"name":"extra"}}`),
	}
	inputs, err := Load(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if inputs.Cluster.Host != "https://127.0.0.1:1" || inputs.Cluster.BearerToken != "sensitive-token" {
		t.Fatal("kubeconfig credentials were not preserved")
	}
	if len(inputs.TargetSetup) != 3 || inputs.TargetSetup[2].GetName() != "extra" {
		t.Fatal("repeatable, multi-document manifests were not loaded in order")
	}
	if inputs.DeploymentTarget.Parameters.Name != "target-credentials" || len(inputs.RegistrySecrets) != 0 {
		t.Fatal("unexpected target or anonymous credentials")
	}
}

func TestInvalidInputs(t *testing.T) {
	tests := []struct {
		name   string
		change func(*testing.T, *Options)
		want   string
	}{
		{"missing artifact", func(_ *testing.T, o *Options) { o.ArtifactReference = "" }, "artifact-reference"},
		{"untagged artifact", func(_ *testing.T, o *Options) {
			o.ArtifactReference = "registry.example.com/app"
		}, "artifact-reference"},
		{"missing kubeconfig", func(_ *testing.T, o *Options) { o.Kubeconfig += ".missing" }, "Kubernetes client"},
		{"missing target", func(_ *testing.T, o *Options) { o.DeploymentTargetSpec += ".missing" }, "deployment-target-spec"},
		{"full resource", func(t *testing.T, o *Options) {
			o.DeploymentTargetSpec = writeFile(t, t.TempDir(), "target", "apiVersion: konfidence.cloud/v1alpha1\nkind: DeploymentTarget")
		}, "DeploymentTarget.spec"},
		{"invalid artifact type", func(t *testing.T, o *Options) {
			o.DeploymentTargetSpec = writeFile(t, t.TempDir(), "target", "artifactType: invalid")
		}, "artifactType"},
		{"multiple target documents", func(t *testing.T, o *Options) {
			o.DeploymentTargetSpec = writeFile(t, t.TempDir(), "target",
				"artifactType: app.example.com\n---\nartifactType: other.example.com")
		}, "exactly one document"},
		{"duplicate target fields", func(t *testing.T, o *Options) {
			o.DeploymentTargetSpec = writeFile(t, t.TempDir(), "target",
				"artifactType: app.example.com\nartifactType: other.example.com")
		}, "DeploymentTarget.spec"},
		{"invalid parameters", func(t *testing.T, o *Options) {
			o.DeploymentTargetSpec = writeFile(t, t.TempDir(), "target", "artifactType: app.example.com\nparameters: {}")
		}, "parameters"},
		{"empty manifest", func(t *testing.T, o *Options) {
			o.TargetSetup = []string{writeFile(t, t.TempDir(), "setup", "# empty")}
		}, "no resources"},
		{"malformed secret", func(t *testing.T, o *Options) {
			o.TargetSetup = []string{writeFile(t, t.TempDir(), "setup", "password: [sensitive-password")}
		}, "invalid target-setup"},
		{"pinned namespace", func(t *testing.T, o *Options) {
			o.TargetSetup = []string{writeFile(t, t.TempDir(), "setup",
				"apiVersion: v1\nkind: Secret\nmetadata:\n  name: credentials\n  namespace: existing")}
		}, "must omit metadata.namespace"},
		{"list wrapper", func(t *testing.T, o *Options) {
			o.TargetSetup = []string{writeFile(t, t.TempDir(), "setup",
				"apiVersion: v1\nkind: List\nmetadata:\n  name: resources\nitems: []")}
		}, "not a List"},
		{"explicit missing registry config", func(_ *testing.T, o *Options) {
			o.OCMRegistryConfig = filepath.Join(t.TempDir(), "missing")
		}, "ocm-registry-config"},
		{"invalid registry config", func(t *testing.T, o *Options) {
			o.OCMRegistryConfig = writeFile(t, t.TempDir(), "config.json", `{"auths":{"sensitive-password":`)
		}, "ocm-registry-config"},
		{"registry config is not a Secret", func(t *testing.T, o *Options) {
			o.OCMRegistryConfig = writeFile(t, t.TempDir(), "registry.yaml",
				"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: registry")
		}, "only v1 Secret"},
		{"registry Secret with namespace", func(t *testing.T, o *Options) {
			o.OCMRegistryConfig = writeFile(t, t.TempDir(), "registry.yaml",
				"apiVersion: v1\nkind: Secret\nmetadata:\n  name: registry\n  namespace: existing")
		}, "must omit metadata.namespace"},
		{"invalid registry Secret data", func(t *testing.T, o *Options) {
			o.OCMRegistryConfig = writeFile(t, t.TempDir(), "registry.yaml",
				"apiVersion: v1\nkind: Secret\nmetadata:\n  name: registry\ndata:\n  .dockerconfigjson: sensitive-password")
		}, "cannot decode"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := validOptions(t)
			test.change(t, &options)
			_, err := Load(context.Background(), options)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want error containing %q", err, test.want)
			}
			if strings.Contains(err.Error(), "sensitive-") {
				t.Fatal("error exposed credentials")
			}
		})
	}
}

func TestKubeconfigEnvironment(t *testing.T) {
	options := validOptions(t)
	t.Setenv("KUBECONFIG", options.Kubeconfig)
	options.Kubeconfig = ""
	if _, err := Load(context.Background(), options); err != nil {
		t.Fatal(err)
	}
}

func TestRegistrySecrets(t *testing.T) {
	options := validOptions(t)
	auth := base64.StdEncoding.EncodeToString([]byte("username:sensitive-password"))
	config := `{"auths":{"registry.example.com":{"auth":"` + auth + `"}}}`
	path := writeFile(t, t.TempDir(), "registry.yaml", `
apiVersion: v1
kind: Secret
metadata:
  name: registry
type: kubernetes.io/dockerconfigjson
stringData:
  .dockerconfigjson: '`+config+`'
---
apiVersion: v1
kind: Secret
metadata:
  name: second-registry
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: `+base64.StdEncoding.EncodeToString([]byte(config))+`
`)
	options.OCMRegistryConfig = path
	inputs, err := Load(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs.RegistrySecrets) != 2 || len(inputs.TargetSetup) != 0 {
		t.Fatal("shared registry Secrets must be retained separately from target setup")
	}
	if inputs.RegistrySecrets[0].StringData[".dockerconfigjson"] != config ||
		string(inputs.RegistrySecrets[1].Data[".dockerconfigjson"]) != config {
		t.Fatal("registry Secret data was not preserved")
	}
	for _, secret := range inputs.RegistrySecrets {
		if secret.Namespace != "" {
			t.Fatal("namespace must be assigned by suite execution, not local setup")
		}
	}
	// No ambient Docker configuration is used without an explicit manifest input.
	options.OCMRegistryConfig = ""
	t.Setenv("DOCKER_CONFIG", filepath.Dir(path))
	writeFile(t, filepath.Dir(path), "config.json", config)
	if inputs, err = Load(context.Background(), options); err != nil || len(inputs.RegistrySecrets) != 0 {
		t.Fatalf("expected no registry Secrets without explicit input: %v", err)
	}
}

func TestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Load(ctx, Options{}); err != context.Canceled {
		t.Fatalf("got %v, want context cancellation", err)
	}
}

func TestDuplicateRegistrySecrets(t *testing.T) {
	options := validOptions(t)
	secret := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: registry\n"
	options.OCMRegistryConfig = writeFile(t, t.TempDir(), "registry.yaml", secret+"---\n"+secret)
	if _, err := Load(context.Background(), options); err == nil || !strings.Contains(err.Error(), "unique") {
		t.Fatalf("expected duplicate registry Secret error, got %v", err)
	}
}

func TestDigestReference(t *testing.T) {
	options := validOptions(t)
	options.ArtifactReference = "registry.example.com/app@sha256:" + strings.Repeat("a", 64)
	if _, err := Load(context.Background(), options); err != nil {
		t.Fatal(err)
	}
}
