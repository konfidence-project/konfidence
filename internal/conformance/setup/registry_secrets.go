package setup

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

// loadRegistrySecrets loads manifests only. Credential interpretation happens
// through ResolverFromRefs after the suite creates these Secrets in the cluster.
func loadRegistrySecrets(path string) ([]*corev1.Secret, error) {
	if path == "" {
		return nil, nil
	}
	objects, err := loadManifests(path, "ocm-registry-config")
	if err != nil {
		return nil, err
	}
	secrets := make([]*corev1.Secret, 0, len(objects))
	names := make(map[string]bool, len(objects))
	for _, object := range objects {
		if object.GetAPIVersion() != "v1" || object.GetKind() != "Secret" {
			return nil, fmt.Errorf("ocm-registry-config must contain only v1 Secret resources")
		}
		if len(validation.IsDNS1123Subdomain(object.GetName())) != 0 || names[object.GetName()] {
			return nil, fmt.Errorf("ocm-registry-config Secret names must be valid and unique")
		}
		data, err := json.Marshal(object.Object)
		if err != nil {
			return nil, fmt.Errorf("cannot decode ocm-registry-config Secret")
		}
		var secret corev1.Secret
		if err := json.Unmarshal(data, &secret); err != nil {
			return nil, fmt.Errorf("cannot decode ocm-registry-config Secret")
		}
		names[secret.Name] = true
		secrets = append(secrets, &secret)
	}
	return secrets, nil
}
