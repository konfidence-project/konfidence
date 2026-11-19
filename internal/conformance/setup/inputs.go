// Package setup loads caller-owned inputs without contacting or modifying the cluster.
package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/validation"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"oras.land/oras-go/v2/registry"
	"sigs.k8s.io/yaml"
)

// Options contains file paths and CLI values, never inline credentials.
type Options struct {
	Kubeconfig           string
	ArtifactReference    string
	DeploymentTargetSpec string
	TargetSetup          []string
	OCMRegistryConfig    string
}

// Inputs is the validated local configuration passed to the suite.
// It contains secrets and must not be logged or serialized.
type Inputs struct {
	Cluster           *rest.Config
	ArtifactReference registry.Reference
	DeploymentTarget  konfidence.DeploymentTargetSpec
	TargetSetup       []unstructured.Unstructured
	RegistrySecrets   []*corev1.Secret
}

// Load validates local inputs. Remote prerequisite checks belong to the suite.
func Load(ctx context.Context, options Options) (*Inputs, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ref, err := registry.ParseReference(options.ArtifactReference)
	if err != nil || ref.Reference == "" {
		return nil, fmt.Errorf("artifact-reference must be an OCI image reference with a tag or digest")
	}
	target, err := loadTarget(options.DeploymentTargetSpec)
	if err != nil {
		return nil, err
	}
	var resources []unstructured.Unstructured
	for _, path := range options.TargetSetup {
		objects, err := loadManifests(path, "target-setup")
		if err != nil {
			return nil, err
		}
		resources = append(resources, objects...)
	}
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.ExplicitPath = options.Kubeconfig
	overrides := &clientcmd.ConfigOverrides{}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		// Kubeconfig parser errors can contain credential material.
		return nil, fmt.Errorf("cannot load Kubernetes client configuration; check kubeconfig and its current-context")
	}
	secrets, err := loadRegistrySecrets(options.OCMRegistryConfig)
	if err != nil {
		return nil, err
	}
	return &Inputs{
		Cluster: config, ArtifactReference: ref, DeploymentTarget: target,
		TargetSetup: resources, RegistrySecrets: secrets,
	}, nil
}

func loadTarget(path string) (konfidence.DeploymentTargetSpec, error) {
	var spec konfidence.DeploymentTargetSpec
	data, err := os.ReadFile(path)
	if err != nil {
		return spec, fmt.Errorf("cannot read deployment-target-spec %q", path)
	}
	if err := yaml.UnmarshalStrict(data, &spec); err != nil {
		return spec, fmt.Errorf("deployment-target-spec must contain only valid DeploymentTarget.spec fields")
	}
	decoder := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	var document json.RawMessage
	if err := decoder.Decode(&document); err != nil {
		return spec, fmt.Errorf("invalid deployment-target-spec document")
	}
	if err := decoder.Decode(&document); err != io.EOF {
		return spec, fmt.Errorf("deployment-target-spec must contain exactly one document")
	}
	if !spec.ArtifactType.Valid() {
		return spec, fmt.Errorf("deployment-target-spec has an invalid artifactType")
	}
	if p := spec.Parameters; p != nil {
		if p.Kind == "" || len(p.Kind) > 63 || len(validation.IsDNS1123Subdomain(p.Name)) != 0 {
			return spec, fmt.Errorf("deployment-target-spec parameters require a valid kind and name")
		}
		if p.APIGroup != "" && len(validation.IsDNS1123Subdomain(p.APIGroup)) != 0 {
			return spec, fmt.Errorf("deployment-target-spec parameters have an invalid apiGroup")
		}
	}
	return spec, nil
}

func loadManifests(path, input string) ([]unstructured.Unstructured, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s %q", input, path)
	}
	decoder := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	var objects []unstructured.Unstructured
	for document := 1; ; document++ {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("invalid %s %q document %d", input, path, document)
		}
		if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			continue
		}
		var object unstructured.Unstructured
		if err := json.Unmarshal(raw, &object); err != nil {
			return nil, fmt.Errorf("invalid %s %q document %d", input, path, document)
		}
		if object.GetAPIVersion() == "" || object.GetKind() == "" || object.GetName() == "" {
			return nil, fmt.Errorf("%s %q document %d requires apiVersion, kind and metadata.name", input, path, document)
		}
		if object.IsList() {
			return nil, fmt.Errorf("%s %q document %d must be an individual resource, not a List", input, path, document)
		}
		if object.GetNamespace() != "" {
			return nil, fmt.Errorf("%s %q document %d must omit metadata.namespace; the suite owns isolation", input, path, document)
		}
		objects = append(objects, object)
	}
	if len(objects) == 0 {
		return nil, fmt.Errorf("%s %q contains no resources", input, path)
	}
	return objects, nil
}
