package suite

import (
	"fmt"

	sharedcredentials "github.com/konfidence-project/konfidence/pkg/ocm/credentials"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// registerLifecycle gives Ginkgo ownership of run-scoped registry resources.
// Per-case resources belong in the scenarios' BeforeEach/DeferCleanup hooks.
func registerLifecycle(environment *Environment) {
	ginkgo.BeforeSuite(func(ctx ginkgo.SpecContext) {
		if len(environment.Inputs.RegistrySecrets) == 0 {
			return
		}
		initializeClient(environment)
		namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "kden-conformance-"}}
		if err := environment.Client.Create(ctx, namespace); err != nil {
			ginkgo.Fail("cannot create shared conformance namespace")
		}
		environment.SharedNamespace = namespace
		ginkgo.DeferCleanup(func(cleanupCtx ginkgo.SpecContext) {
			if err := environment.Client.Delete(cleanupCtx, namespace); err != nil && !apierrors.IsNotFound(err) {
				ginkgo.Fail(fmt.Sprintf("cannot delete shared conformance namespace %q", namespace.Name))
			}
		}, ginkgo.NodeTimeout(scenarioCleanupTimeout))

		refs := make([]sharedcredentials.Ref, 0, len(environment.Inputs.RegistrySecrets))
		for _, source := range environment.Inputs.RegistrySecrets {
			secret := source.DeepCopy()
			secret.Namespace = namespace.Name
			if err := environment.Client.Create(ctx, secret); err != nil {
				// Do not expose API response bodies that could include Secret data.
				ginkgo.Fail(fmt.Sprintf("cannot create OCM registry Secret %q", secret.Name))
			}
			refs = append(refs, sharedcredentials.Ref{Name: secret.Name})
		}
		resolver, err := sharedcredentials.ResolverFromRefs(ctx, environment.Client, namespace.Name, refs)
		if err != nil {
			ginkgo.Fail("cannot load OCM registry credentials; check the Secrets' .ocmconfig or .dockerconfigjson data")
		}
		environment.RegistryCredentials = resolver
	}, ginkgo.NodeTimeout(defaultScenarioTimeout))
}
