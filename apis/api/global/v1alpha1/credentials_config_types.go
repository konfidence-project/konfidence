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

package v1alpha1

// CredentialsConfig defines a credential reference to a secret or configMap used to access an OCI registry.
type CredentialsConfig struct {
	// Kind of the configuration resource. Allowed values are Secret or ConfigMap.
	Kind string `json:"kind"`

	// APIVersion is the api version of the of configuration resource, e.g. v1.
	APIVersion string `json:"apiVersion"`

	// Name is the name	 of the of configuration resource.
	Name string `json:"name"`
}
