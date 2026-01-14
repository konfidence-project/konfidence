/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package secret

import (
	"context"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
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
	configMap := &v1.ConfigMap{}
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
