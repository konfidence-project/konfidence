package controller_test

import (
	"context"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/landscape/internal/controller"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Landscape Controller", Ordered, func() {
	var (
		k8sClient client.Client
		cancel    context.CancelFunc
	)

	BeforeAll(func() {
		k8sClient, cancel = StartTestManagerWithReconciler(func(mgr ctrl.Manager) error {
			return controller.NewLandscapeReconciler(mgr).SetupWithManager(mgr)
		})
	})

	AfterAll(func() {
		cancel()
	})

	const (
		ControllerName = "landscape-controller"
		timeout        = time.Second * 10
		interval       = time.Millisecond * 250
	)

	ctx := context.Background()

	expectCondition := func(g Gomega, landscape *konfidence.Landscape, conditionType string, status metav1.ConditionStatus, reason string) {
		condition := meta.FindStatusCondition(landscape.Status.Conditions, conditionType)
		g.Expect(condition).NotTo(BeNil(), "condition %s missing", conditionType)
		g.Expect(condition.Status).To(Equal(status))
		g.Expect(condition.Reason).To(Equal(reason))
	}

	expectManagedNamespace := func(g Gomega, ns *corev1.Namespace, projectName, landscapeName string) {
		g.Expect(ns).NotTo(BeNil())
		g.Expect(ns.Labels).To(HaveKeyWithValue(pkgctrl.ManagedByLabel, ControllerName))
		g.Expect(ns.Labels).To(HaveKeyWithValue(pkgctrl.ProjectTypeLabel, "landscape"))
		g.Expect(ns.Labels).To(HaveKeyWithValue(pkgctrl.ProjectNameLabel, projectName))
		g.Expect(ns.Labels).To(HaveKeyWithValue(pkgctrl.LandscapeNameLabel, landscapeName))
		// Note: Landscape is namespace-scoped, so it cannot set an owner reference
		// on the cluster-scoped Namespace. We rely on labels and finalizers instead.
	}

	It("creates the namespace with fallback name", func() {
		const projectName = "proj1"
		const projectNsName = "kden-p-proj1"
		const landscapeName = "land-default"

		controller.CreateProjectNamespace(ctx, k8sClient, projectNsName, projectName)
		DeferCleanup(func() { controller.CleanupProjectNamespace(ctx, k8sClient, projectNsName) })

		controller.CreateLandscape(ctx, k8sClient, landscapeName, projectNsName)
		DeferCleanup(func() { controller.CleanupLandscape(ctx, k8sClient, landscapeName, projectNsName) })

		Eventually(func(g Gomega) {
			landscape := controller.GetLandscape(ctx, k8sClient, landscapeName, projectNsName, false)
			g.Expect(landscape.Status.Namespace).NotTo(BeEmpty())
			g.Expect(landscape.Status.ProjectName).To(Equal(projectName))
			expectCondition(g, landscape, konfidence.LandscapeReadyCondition, metav1.ConditionTrue, konfidence.LandscapeReconciledReason)
			expectCondition(g, landscape, konfidence.LandscapeNamespaceReadyCondition, metav1.ConditionTrue, konfidence.LandscapeNamespaceReconciledReason)

			expectedNsName := landscape.Status.Namespace
			ns := controller.GetNamespace(ctx, k8sClient, expectedNsName, true)
			expectManagedNamespace(g, ns, projectName, landscapeName)
		}, timeout, interval).Should(Succeed())
	})

	It("creates the namespace from spec.namespace when overridden", func() {
		const projectName = "proj2"
		const projectNsName = "kden-p-proj2"
		const landscapeName = "land-override"
		const customNsName = "land-override-custom-ns"

		controller.CreateProjectNamespace(ctx, k8sClient, projectNsName, projectName)
		DeferCleanup(func() { controller.CleanupProjectNamespace(ctx, k8sClient, projectNsName) })

		controller.CreateLandscape(ctx, k8sClient, landscapeName, projectNsName, func(l *konfidence.Landscape) {
			l.Spec.Namespace = customNsName
		})
		DeferCleanup(func() { controller.CleanupLandscape(ctx, k8sClient, landscapeName, projectNsName) })

		Eventually(func(g Gomega) {
			ns := controller.GetNamespace(ctx, k8sClient, customNsName, true)
			expectManagedNamespace(g, ns, projectName, landscapeName)
		}, timeout, interval).Should(Succeed())

		// Ensure the default namespace was not created
		const defaultNsName = "kden-l-land-override-xsf5ze"
		Expect(controller.GetNamespace(ctx, k8sClient, defaultNsName, true)).To(BeNil())
	})

	It("sets InvalidNamespace status when created in non-project namespace", func() {
		const regularNsName = "regular-namespace"
		const landscapeName = "land-invalid"

		// Create a regular namespace (not a project namespace)
		regularNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: regularNsName,
			},
		}
		Expect(k8sClient.Create(ctx, regularNs)).To(Succeed())
		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, regularNs))).To(Succeed())
		})

		controller.CreateLandscape(ctx, k8sClient, landscapeName, regularNsName)
		DeferCleanup(func() { controller.CleanupLandscape(ctx, k8sClient, landscapeName, regularNsName) })

		Eventually(func(g Gomega) {
			landscape := controller.GetLandscape(ctx, k8sClient, landscapeName, regularNsName, false)
			expectCondition(g, landscape, konfidence.LandscapeReadyCondition, metav1.ConditionFalse, konfidence.LandscapeInvalidNamespaceReason)
			expectCondition(g, landscape, konfidence.LandscapeNamespaceReadyCondition, metav1.ConditionFalse, konfidence.LandscapeInvalidNamespaceReason)

			g.Expect(landscape.Status.ProjectName).To(BeEmpty())
			g.Expect(landscape.Status.Namespace).To(BeEmpty())
		}, timeout, interval).Should(Succeed())
	})

	It("sets InvalidNamespace status when project namespace is missing project label", func() {
		const incompleteNsName = "incomplete-project-ns"
		const landscapeName = "land-incomplete"

		// Create a namespace with project type but no project name label
		incompleteNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: incompleteNsName,
				Labels: map[string]string{
					pkgctrl.ProjectTypeLabel: "project",
					// Missing pkgctrl.ProjectNameLabel
				},
			},
		}
		Expect(k8sClient.Create(ctx, incompleteNs)).To(Succeed())
		DeferCleanup(func() {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, incompleteNs))).To(Succeed())
		})

		controller.CreateLandscape(ctx, k8sClient, landscapeName, incompleteNsName)
		DeferCleanup(func() { controller.CleanupLandscape(ctx, k8sClient, landscapeName, incompleteNsName) })

		Eventually(func(g Gomega) {
			landscape := controller.GetLandscape(ctx, k8sClient, landscapeName, incompleteNsName, false)
			expectCondition(g, landscape, konfidence.LandscapeReadyCondition, metav1.ConditionFalse, konfidence.LandscapeInvalidNamespaceReason)
		}, timeout, interval).Should(Succeed())
	})

	DescribeTable("rejects invalid Landscapes at admission",
		func(projectNsName string, mutate func(*konfidence.Landscape), expectedError string) {
			// Create project namespace if needed
			if projectNsName != "" {
				projectName := "test-proj-" + projectNsName // Make unique per test
				controller.CreateProjectNamespace(ctx, k8sClient, projectNsName, projectName)
				DeferCleanup(func() { controller.CleanupProjectNamespace(ctx, k8sClient, projectNsName) })
			}

			landscape := controller.NewLandscape("land-invalid", projectNsName, mutate)
			err := k8sClient.Create(ctx, landscape)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expectedError))
		},
		Entry("with a landscape name longer than 46 characters", "kden-p-test1", func(l *konfidence.Landscape) {
			l.Name = "landscape-with-a-very-long-name-that-exceeds-limit-x"
		}, "landscape name must be at most 46 characters"),
		Entry("with an invalid namespace name", "kden-p-test2", func(l *konfidence.Landscape) {
			l.Spec.Namespace = "Invalid_Namespace"
		}, "should match"),
	)

	// expectNamespaceUpdateRejected asserts that setting spec.namespace to the
	// given value is rejected as immutable. It refetches and retries via
	// Eventually, since the controller concurrently updates the Landscape
	// (finalizer, status) and a stale write would fail with a conflict instead
	// of the expected CEL rejection.
	expectNamespaceUpdateRejected := func(landscapeName, projectNsName, namespace string) {
		Eventually(func(g Gomega) {
			landscape := controller.GetLandscape(ctx, k8sClient, landscapeName, projectNsName, false)
			landscape.Spec.Namespace = namespace
			err := k8sClient.Update(ctx, landscape)
			g.Expect(err).To(HaveOccurred())
			g.Expect(err.Error()).To(ContainSubstring("namespace is immutable"))
		}, timeout, interval).Should(Succeed())
	}

	It("rejects spec.namespace mutations", func() {
		const projectName = "proj4"
		const projectNsName = "kden-p-proj4"
		const landscapeName = "land-immutable"

		controller.CreateProjectNamespace(ctx, k8sClient, projectNsName, projectName)
		DeferCleanup(func() { controller.CleanupProjectNamespace(ctx, k8sClient, projectNsName) })

		controller.CreateLandscape(ctx, k8sClient, landscapeName, projectNsName)
		DeferCleanup(func() { controller.CleanupLandscape(ctx, k8sClient, landscapeName, projectNsName) })

		By("rejecting setting an unset spec.namespace")
		expectNamespaceUpdateRejected(landscapeName, projectNsName, "land-immutable-custom")

		By("rejecting unsetting a set spec.namespace")
		const landscapeName2 = "land-immutable2"
		controller.CreateLandscape(ctx, k8sClient, landscapeName2, projectNsName, func(l *konfidence.Landscape) {
			l.Spec.Namespace = "land-immutable2-custom"
		})
		DeferCleanup(func() { controller.CleanupLandscape(ctx, k8sClient, landscapeName2, projectNsName) })
		expectNamespaceUpdateRejected(landscapeName2, projectNsName, "")

		By("rejecting changing a set spec.namespace")
		expectNamespaceUpdateRejected(landscapeName2, projectNsName, "land-immutable2-changed")
	})

	It("generates unique namespace names for landscapes with same name in different projects", func() {
		const landscapeName = "dev"
		const projectName1 = "proj5a"
		const projectName2 = "proj5b"
		const projectNsName1 = "kden-p-proj5a"
		const projectNsName2 = "kden-p-proj5b"

		controller.CreateProjectNamespace(ctx, k8sClient, projectNsName1, projectName1)
		controller.CreateProjectNamespace(ctx, k8sClient, projectNsName2, projectName2)
		DeferCleanup(func() {
			controller.CleanupProjectNamespace(ctx, k8sClient, projectNsName1)
			controller.CleanupProjectNamespace(ctx, k8sClient, projectNsName2)
		})

		controller.CreateLandscape(ctx, k8sClient, landscapeName, projectNsName1)
		controller.CreateLandscape(ctx, k8sClient, landscapeName, projectNsName2)
		DeferCleanup(func() {
			controller.CleanupLandscape(ctx, k8sClient, landscapeName, projectNsName1)
			controller.CleanupLandscape(ctx, k8sClient, landscapeName, projectNsName2)
		})

		var ns1Name, ns2Name string
		Eventually(func(g Gomega) {
			landscape1 := controller.GetLandscape(ctx, k8sClient, landscapeName, projectNsName1, false)
			landscape2 := controller.GetLandscape(ctx, k8sClient, landscapeName, projectNsName2, false)
			ns1Name = landscape1.Status.Namespace
			ns2Name = landscape2.Status.Namespace
			g.Expect(ns1Name).NotTo(BeEmpty())
			g.Expect(ns2Name).NotTo(BeEmpty())
			g.Expect(ns1Name).NotTo(Equal(ns2Name), "landscape namespaces should be unique across projects")
		}, timeout, interval).Should(Succeed())

		// Verify both namespaces exist
		Expect(controller.GetNamespace(ctx, k8sClient, ns1Name, true)).NotTo(BeNil())
		Expect(controller.GetNamespace(ctx, k8sClient, ns2Name, true)).NotTo(BeNil())
	})
})
