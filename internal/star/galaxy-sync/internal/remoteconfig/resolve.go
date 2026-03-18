/*
Copyright 2026.

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

package remoteconfig

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	SecretName = "gcp-sync-kubeconfig"
	SecretKey  = "kubeconfig"
)

// FromSecret reads the kubeconfig bytes stored under SecretKey in the Secret
// named SecretName inside the given namespace and returns the corresponding
// *rest.Config.
//
// It returns (nil, nil) when the Secret does not exist so that the caller can
// apply the single-cluster fallback.
func FromSecret(c client.Client, namespace string) (*rest.Config, error) {
	logger := log.Log.WithName("remoteconfig")

	secretKey := types.NamespacedName{
		Namespace: namespace,
		Name:      SecretName,
	}

	logger.Info("Looking up remote kubeconfig Secret", "secret", secretKey.String())

	secret := &corev1.Secret{}
	if err := c.Get(context.Background(), secretKey, secret); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Remote kubeconfig Secret not found", "secret", secretKey.String())
			return nil, nil
		}
		return nil, fmt.Errorf("unable to fetch Secret %s: %w", secretKey.String(), err)
	}

	kubeconfigBytes, ok := secret.Data[SecretKey]
	if !ok {
		return nil, fmt.Errorf("secret %s does not contain key %q", secretKey.String(), SecretKey)
	}

	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse kubeconfig from Secret %s: %w", secretKey.String(), err)
	}

	logger.Info("Successfully loaded remote kubeconfig from Secret", "secret", secretKey.String())
	return cfg, nil
}
