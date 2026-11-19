// Package suite owns conformance execution, not CLI parsing or credential loading.
package suite

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/konfidence-project/konfidence/internal/conformance/setup"
	sharedcredentials "github.com/konfidence-project/konfidence/pkg/ocm/credentials"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"ocm.software/open-component-model/bindings/go/credentials"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Result summarizes conformance scenario execution.
type Result struct {
	Executed int
}

// Run prepares shared registry Secrets when supplied, then executes scenarios.
// Per-case target setup resources are separate from these run-scoped credentials.
func Run(ctx context.Context, inputs *setup.Inputs) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if len(inputs.RegistrySecrets) == 0 {
		return runScenarios(ctx, inputs, nil)
	}
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return Result{}, err
	}
	k8sClient, err := client.New(inputs.Cluster, client.Options{Scheme: scheme})
	if err != nil {
		return Result{}, fmt.Errorf("cannot initialize Kubernetes client for registry credentials")
	}
	return runWithClient(ctx, inputs, k8sClient)
}

func runWithClient(ctx context.Context, inputs *setup.Inputs, k8sClient client.Client) (result Result, runErr error) {
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "kden-conformance-"}}
	setupCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	if err := k8sClient.Create(setupCtx, namespace); err != nil {
		return Result{}, fmt.Errorf("cannot create shared conformance namespace")
	}
	defer func() {
		// Cleanup must still run when execution is canceled. Deleting the owned
		// namespace asks Kubernetes to remove all resources created within it.
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := k8sClient.Delete(cleanupCtx, namespace); err != nil && !apierrors.IsNotFound(err) {
			runErr = errors.Join(runErr, fmt.Errorf("cannot delete shared conformance namespace %q", namespace.Name))
		}
	}()
	refs := make([]sharedcredentials.Ref, 0, len(inputs.RegistrySecrets))
	for _, source := range inputs.RegistrySecrets {
		secret := source.DeepCopy()
		secret.Namespace = namespace.Name
		if err := k8sClient.Create(setupCtx, secret); err != nil {
			// API error messages can contain rejected Secret values.
			return Result{}, fmt.Errorf("cannot create OCM registry Secret %q", secret.Name)
		}
		refs = append(refs, sharedcredentials.Ref{Name: secret.Name})
	}
	resolver, err := sharedcredentials.ResolverFromRefs(setupCtx, k8sClient, namespace.Name, refs)
	if err != nil {
		return Result{}, fmt.Errorf("cannot load OCM registry credentials; check the Secrets' .ocmconfig or .dockerconfigjson data")
	}
	return runScenarios(ctx, inputs, resolver)
}

// runScenarios is the execution boundary for cases using prepared inputs and
// OCM registry credentials.
func runScenarios(ctx context.Context, _ *setup.Inputs, _ credentials.Resolver) (Result, error) {
	return Result{}, ctx.Err()
}
