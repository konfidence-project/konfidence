package controller

import (
	"context"
	"errors"
	"sync/atomic"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var _ = Describe("VectorTemplate result persistence", func() {
	It("reuses a completed assembly result when the status patch is retried", func() {
		testCtx := context.Background()
		key := types.NamespacedName{
			Namespace: "result-persistence",
			Name:      "template",
		}

		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(konfidence.AddToScheme(scheme)).To(Succeed())

		namespace := &corev1.Namespace{
			Name: key.Namespace,
			Labels: map[string]string{
				pkgctrl.ProjectTypeLabel: "project",
			},
		}
		vectorTemplate := &konfidence.VectorTemplate{
			Namespace:  key.Namespace,
			Name:       key.Name,
			UID:        types.UID("template-uid"),
			Generation: 1,
		}

		patchFailure := errors.New("injected status patch failure")
		var patchAttempts atomic.Int32

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(namespace, vectorTemplate).
			WithStatusSubresource(&konfidence.VectorTemplate{}).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, c client.Client, subResource string, obj client.Object,
					patch client.Patch, opts ...client.SubResourcePatchOption,
				) error {
					if patchAttempts.Add(1) == 1 {
						return patchFailure
					}

					return c.Status().Patch(ctx, obj, patch, opts...)
				},
			}).
			Build()

		registry := newJobRegistry()
		DeferCleanup(registry.shutdown)

		reconciler := &VectorTemplateReconciler{
			Client:   k8sClient,
			Recorder: events.NewFakeRecorder(10),
			jobs:     registry,
		}

		By("registering a successfully completed assembly job")
		Expect(registry.launch(key, vectorTemplate.Generation, vectorTemplate.UID, func(context.Context) assemblyResult {
			return assemblyResult{
				latestVector:  "registry.example/vector:1.0.0",
				vectorVersion: "1.0.0",
				componentName: "vector",
			}
		},
		)).To(BeTrue())

		job, exists := registry.get(key)
		Expect(exists).To(BeTrue())

		Eventually(func() bool {
			_, completed := registry.completedResult(key, job)
			return completed
		}).Should(BeTrue())

		By("failing the first status patch")
		_, err := reconciler.Reconcile(testCtx, ctrl.Request{NamespacedName: key})
		Expect(err).To(MatchError(patchFailure))

		By("retaining the completed result")
		retainedJob, exists := registry.get(key)
		Expect(exists).To(BeTrue())
		Expect(retainedJob).To(BeIdenticalTo(job))

		retainedResult, completed := registry.completedResult(key, retainedJob)
		Expect(completed).To(BeTrue())
		Expect(retainedResult.latestVector).To(Equal("registry.example/vector:1.0.0"))

		stored := &konfidence.VectorTemplate{}
		Expect(k8sClient.Get(testCtx, key, stored)).To(Succeed())
		Expect(stored.Status.LatestVector).To(BeEmpty())

		By("reapplying the cached result on the next reconciliation")
		_, err = reconciler.Reconcile(testCtx, ctrl.Request{NamespacedName: key})
		Expect(err).NotTo(HaveOccurred())
		Expect(patchAttempts.Load()).To(Equal(int32(2)))

		Expect(k8sClient.Get(testCtx, key, stored)).To(Succeed())
		Expect(stored.Status.LatestVector).To(Equal("registry.example/vector:1.0.0"))

		By("acknowledging the job after the successful patch")
		_, exists = registry.get(key)
		Expect(exists).To(BeFalse())
	})
})
