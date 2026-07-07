package ocm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/konfidence-project/konfidence/pkg/testutil/pki"
	. "github.com/onsi/gomega" //nolint:staticcheck
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ocmv1alpha1 "ocm.software/open-component-model/kubernetes/controller/api/v1alpha1"
)

// SignatureBinding associates an RSA key pair with an OCM signature name.
// Used by OCMConfigSecret to register credentials for a named signature.
type SignatureBinding struct {
	SignatureName string
	Pair          pki.RSAKeyPair
}

// Bind is a convenience constructor for SignatureBinding.
func Bind(signatureName string, pair pki.RSAKeyPair) SignatureBinding {
	return SignatureBinding{SignatureName: signatureName, Pair: pair}
}

// OCMConfigSecret builds a corev1.Secret with a .ocmconfig entry that registers
// RSACredentials/v1 for each (signatureName, RSAKeyPair) binding.
// Multiple bindings produce multiple consumers in one secret.
func OCMConfigSecret(name, namespace string, bindings ...SignatureBinding) *corev1.Secret {
	consumers := make([]map[string]any, 0, len(bindings))
	for _, b := range bindings {
		consumers = append(consumers, map[string]any{
			"identities": []map[string]any{
				{
					"type":      "RSA/v1alpha1",
					"signature": b.SignatureName,
					"algorithm": "RSASSA-PSS",
				},
			},
			"credentials": []map[string]any{
				{
					"type":          "RSACredentials/v1",
					"privateKeyPEM": string(b.Pair.PrivateKeyPEM),
					"publicKeyPEM":  string(b.Pair.CertificatePEM),
				},
			},
		})
	}

	credConfig := map[string]any{
		"type":      "credentials.config.ocm.software/v1",
		"consumers": consumers,
	}
	ocmConfig := map[string]any{
		"type":           "generic.config.ocm.software/v1",
		"configurations": []any{credConfig},
	}

	data, err := json.Marshal(ocmConfig)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "marshal .ocmconfig for secret %s", name)

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       map[string][]byte{ocmv1alpha1.OCMConfigKey: data},
	}
}

// DockerConfigSecret builds a corev1.Secret with a .dockerconfigjson for
// a single user:pass credential applied to every listed registry endpoint.
func DockerConfigSecret(name, namespace, user, pass string, registryEndpoints ...string) *corev1.Secret {
	auths := make(map[string]any, len(registryEndpoints))
	for _, ep := range registryEndpoints {
		auths[ep] = map[string]any{
			"username": user,
			"password": pass,
			"auth":     base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, pass))),
		}
	}
	data, err := json.Marshal(map[string]any{"auths": auths})
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "marshal .dockerconfigjson for secret %s", name)

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Type:       corev1.SecretTypeDockerConfigJson,
		Data:       map[string][]byte{corev1.DockerConfigJsonKey: data},
	}
}
