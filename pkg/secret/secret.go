package secret

//go:generate go run go.uber.org/mock/mockgen -destination=internal/mocks/mock_client.go -package=mocks sigs.k8s.io/controller-runtime/pkg/client Client

import (
	"context"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	AuthConfigMapKey          = "authenticationSecretRefs"
	KonfidenceSystemNamespace = "konfidence-system"
)

// GetSecretByConfigMap tries to map a domain name to a secret name using a pre-defined configMap
func GetSecretByConfigMap(ctx context.Context, reader client.Reader, configMapName string,
	domainName string) (string, error) {
	log := logf.FromContext(ctx)
	configMap := &corev1.ConfigMap{}
	// config map must be in konfidence system namespace
	err := reader.Get(ctx, types.NamespacedName{
		Namespace: KonfidenceSystemNamespace,
		Name:      configMapName,
	}, configMap)

	if err != nil && !errors.IsNotFound(err) {
		return "", err
	}
	if err != nil && errors.IsNotFound(err) {
		return "", nil
	}

	authConfig, ok := configMap.Data[AuthConfigMapKey]
	if !ok {
		log.Info("Could not find any data in ConfigMap with AuthConfigMapKey", "key", AuthConfigMapKey)
		return "", nil
	}

	authMap := make(map[string]string)
	err = yaml.Unmarshal([]byte(authConfig), authMap)
	if err != nil {
		log.Info("Error unmarshalling authConfig")
		return "", nil
	}

	secret, ok := authMap[domainName]
	if !ok {
		log.Info("Could not find a map entry for domain", "domainName", domainName)
		return "", nil
	}

	return secret, nil
}
