package controller_test

import (
	"context"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/project/internal/controller"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// validJWKSSubject returns a fully valid JWKS subject that admission accepts,
// so rejection specs can invalidate exactly one field in isolation.
func validJWKSSubject() *konfidence.JWKSSubject {
	return &konfidence.JWKSSubject{
		Endpoint: "https://token.actions.githubusercontent.com/.well-known/openid-configuration",
		Audience: "https://konfidence.example/api",
		Claims:   map[string]konfidence.GlobMatch{"sub": "repo:konfidence-project/konfidence:*"},
	}
}

var _ = Describe("Project Controller", Ordered, func() {
	var (
		k8sClient client.Client
		cancel    context.CancelFunc
	)

	BeforeAll(func() {
		k8sClient, cancel = StartTestManagerWithReconciler(func(mgr ctrl.Manager) error {
			return controller.NewProjectReconciler(mgr).SetupWithManager(mgr)
		})
	})

	AfterAll(func() {
		cancel()
	})

	const (
		ControllerName = "project-controller"
		timeout        = time.Second * 10
		interval       = time.Millisecond * 250
	)

	ctx := context.Background()

	expectCondition := func(g Gomega, project *konfidence.Project, conditionType string, status metav1.ConditionStatus, reason string) {
		condition := meta.FindStatusCondition(project.Status.Conditions, conditionType)
		g.Expect(condition).NotTo(BeNil(), "condition %s missing", conditionType)
		g.Expect(condition.Status).To(Equal(status))
		g.Expect(condition.Reason).To(Equal(reason))
	}

	expectManagedNamespace := func(g Gomega, ns *corev1.Namespace, projectName string) {
		g.Expect(ns).NotTo(BeNil())
		g.Expect(ns.Labels).To(HaveKeyWithValue(pkgctrl.ManagedByLabel, ControllerName))
		g.Expect(ns.Labels).To(HaveKeyWithValue(pkgctrl.ProjectTypeLabel, "project"))
		g.Expect(ns.Labels).To(HaveKeyWithValue(pkgctrl.ProjectNameLabel, projectName))
		ownerRef := metav1.GetControllerOf(ns)
		g.Expect(ownerRef).NotTo(BeNil())
		g.Expect(ownerRef.Kind).To(Equal(konfidence.ProjectKind))
		g.Expect(ownerRef.Name).To(Equal(projectName))
	}

	It("creates the default project namespace with labels and owner reference", func() {
		const projectName = "proj-default"
		controller.CreateProject(ctx, k8sClient, projectName)
		DeferCleanup(func() { controller.CleanupProject(ctx, k8sClient, projectName) })

		nsName := konfidence.ProjectNamespacePrefix + projectName
		Eventually(func(g Gomega) {
			ns := controller.GetNamespace(ctx, k8sClient, nsName, true)
			expectManagedNamespace(g, ns, projectName)
		}, timeout, interval).Should(Succeed())

		Eventually(func(g Gomega) {
			project := controller.GetProject(ctx, k8sClient, projectName, false)
			g.Expect(project.Status.Namespace).To(Equal(nsName))
			expectCondition(g, project, konfidence.ProjectReadyCondition, metav1.ConditionTrue, konfidence.ProjectReconciledReason)
			expectCondition(g, project, konfidence.ProjectNamespaceReadyCondition, metav1.ConditionTrue, konfidence.ProjectNamespaceReconciledReason)
		}, timeout, interval).Should(Succeed())
	})

	It("creates the namespace from spec.namespace when overridden", func() {
		const projectName = "proj-override"
		const nsName = "proj-override-custom-ns"
		controller.CreateProject(ctx, k8sClient, projectName, func(p *konfidence.Project) {
			p.Spec.Namespace = nsName
		})
		DeferCleanup(func() { controller.CleanupProject(ctx, k8sClient, projectName) })

		Eventually(func(g Gomega) {
			ns := controller.GetNamespace(ctx, k8sClient, nsName, true)
			expectManagedNamespace(g, ns, projectName)
		}, timeout, interval).Should(Succeed())

		Expect(controller.GetNamespace(ctx, k8sClient, konfidence.ProjectNamespacePrefix+projectName, true)).To(BeNil())
	})

	It("round-trips the roleBindings schema", func() {
		const projectName = "proj-roles"
		roleBindings := map[string]konfidence.Subjects{
			"admin": {
				{Session: &konfidence.SessionSubject{MemberOf: []string{"group1", "group2"}}},
				{JWKS: &konfidence.JWKSSubject{
					Endpoint: "https://token.actions.githubusercontent.com/.well-known/openid-configuration",
					Audience: "https://konfidence.example/api",
					Claims:   map[string]konfidence.GlobMatch{"sub": "repo:konfidence-project/konfidence:*"},
				}},
			},
			"dev": {
				{Session: &konfidence.SessionSubject{MemberOf: []string{"devs"}}},
			},
		}
		controller.CreateProject(ctx, k8sClient, projectName, func(p *konfidence.Project) {
			p.Spec.DisplayName = "Roles Project"
			p.Spec.RoleBindings = roleBindings
		})
		DeferCleanup(func() { controller.CleanupProject(ctx, k8sClient, projectName) })

		project := controller.GetProject(ctx, k8sClient, projectName, false)
		Expect(project.Spec.DisplayName).To(Equal("Roles Project"))
		Expect(project.Spec.RoleBindings).To(Equal(roleBindings))
	})

	DescribeTable("rejects invalid Projects at admission",
		func(mutate func(*konfidence.Project), expectedError string) {
			project := controller.NewProject("proj-invalid", mutate)
			err := k8sClient.Create(ctx, project)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expectedError))
		},
		Entry("with both session and jwks subjects", func(p *konfidence.Project) {
			p.Spec.RoleBindings = map[string]konfidence.Subjects{"admin": {{
				Session: &konfidence.SessionSubject{MemberOf: []string{"group1"}},
				JWKS:    validJWKSSubject(),
			}}}
		}, "exactly one of session or jwks must be set"),
		Entry("with neither session nor jwks subject", func(p *konfidence.Project) {
			p.Spec.RoleBindings = map[string]konfidence.Subjects{"admin": {{}}}
		}, "exactly one of session or jwks must be set"),
		Entry("with an empty memberOf list", func(p *konfidence.Project) {
			p.Spec.RoleBindings = map[string]konfidence.Subjects{"admin": {{
				Session: &konfidence.SessionSubject{MemberOf: []string{}},
			}}}
		}, "should have at least 1 items"),
		Entry("with a non-https jwks endpoint", func(p *konfidence.Project) {
			jwks := validJWKSSubject()
			jwks.Endpoint = "http://github.example/.well-known/openid-configuration"
			p.Spec.RoleBindings = map[string]konfidence.Subjects{"admin": {{JWKS: jwks}}}
		}, "should match"),
		Entry("with a jwks subject missing an audience", func(p *konfidence.Project) {
			jwks := validJWKSSubject()
			jwks.Audience = ""
			p.Spec.RoleBindings = map[string]konfidence.Subjects{"admin": {{JWKS: jwks}}}
		}, "audience"),
		Entry("with a jwks subject with no claims", func(p *konfidence.Project) {
			jwks := validJWKSSubject()
			jwks.Claims = map[string]konfidence.GlobMatch{}
			p.Spec.RoleBindings = map[string]konfidence.Subjects{"admin": {{JWKS: jwks}}}
		}, "should have at least 1 properties"),
		Entry("with a project name longer than 50 characters", func(p *konfidence.Project) {
			p.Name = "proj-with-a-very-long-name-that-exceeds-the-fifty-character-limit"
		}, "project name must be at most 50 characters"),
		Entry("with an invalid namespace name", func(p *konfidence.Project) {
			p.Spec.Namespace = "Invalid_Namespace"
		}, "should match"),
	)

	// expectNamespaceUpdateRejected asserts that setting spec.namespace to the
	// given value is rejected as immutable. It refetches and retries via
	// Eventually, since the controller concurrently updates the Project
	// (finalizer, status) and a stale write would fail with a conflict instead
	// of the expected CEL rejection.
	expectNamespaceUpdateRejected := func(projectName, namespace string) {
		Eventually(func(g Gomega) {
			project := controller.GetProject(ctx, k8sClient, projectName, false)
			project.Spec.Namespace = namespace
			err := k8sClient.Update(ctx, project)
			g.Expect(err).To(HaveOccurred())
			g.Expect(err.Error()).To(ContainSubstring("namespace is immutable"))
		}, timeout, interval).Should(Succeed())
	}

	It("rejects spec.namespace mutations", func() {
		const projectName = "proj-immutable"
		controller.CreateProject(ctx, k8sClient, projectName)
		DeferCleanup(func() { controller.CleanupProject(ctx, k8sClient, projectName) })

		By("rejecting setting an unset spec.namespace")
		expectNamespaceUpdateRejected(projectName, "proj-immutable-custom-ns")

		const pinnedName = "proj-immutable-pinned"
		controller.CreateProject(ctx, k8sClient, pinnedName, func(p *konfidence.Project) {
			p.Spec.Namespace = "proj-immutable-pinned-ns"
		})
		DeferCleanup(func() { controller.CleanupProject(ctx, k8sClient, pinnedName) })

		By("rejecting changing a set spec.namespace")
		expectNamespaceUpdateRejected(pinnedName, "proj-immutable-other-ns")
	})

	It("refuses to adopt an existing unmanaged namespace", func() {
		const projectName = "proj-conflict"
		const nsName = "proj-conflict-taken-ns"

		unmanaged := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
		Expect(k8sClient.Create(ctx, unmanaged)).To(Succeed())

		controller.CreateProject(ctx, k8sClient, projectName, func(p *konfidence.Project) {
			p.Spec.Namespace = nsName
		})
		DeferCleanup(func() { controller.CleanupProject(ctx, k8sClient, projectName) })

		Eventually(func(g Gomega) {
			project := controller.GetProject(ctx, k8sClient, projectName, false)
			expectCondition(g, project, konfidence.ProjectNamespaceReadyCondition, metav1.ConditionFalse, konfidence.ProjectNamespaceConflictReason)
			expectCondition(g, project, konfidence.ProjectReadyCondition, metav1.ConditionFalse, konfidence.ProjectNamespaceConflictReason)
		}, timeout, interval).Should(Succeed())

		By("leaving the unmanaged namespace untouched")
		Consistently(func(g Gomega) {
			ns := controller.GetNamespace(ctx, k8sClient, nsName, false)
			g.Expect(ns.Labels).NotTo(HaveKey(pkgctrl.ManagedByLabel))
			g.Expect(ns.OwnerReferences).To(BeEmpty())
		}, time.Second*2, interval).Should(Succeed())

		By("emitting a warning event")
		Eventually(func(g Gomega) {
			eventList := &eventsv1.EventList{}
			g.Expect(k8sClient.List(ctx, eventList)).To(Succeed())
			g.Expect(eventList.Items).To(ContainElement(SatisfyAll(
				HaveField("Reason", konfidence.ProjectNamespaceConflictReason),
				HaveField("Type", corev1.EventTypeWarning),
				HaveField("Regarding.Name", projectName),
			)))
		}, timeout, interval).Should(Succeed())
	})

	It("repairs label drift on the managed namespace", func() {
		const projectName = "proj-drift"
		controller.CreateProject(ctx, k8sClient, projectName)
		DeferCleanup(func() { controller.CleanupProject(ctx, k8sClient, projectName) })

		nsName := konfidence.ProjectNamespacePrefix + projectName
		Eventually(func(g Gomega) {
			expectManagedNamespace(g, controller.GetNamespace(ctx, k8sClient, nsName, true), projectName)
		}, timeout, interval).Should(Succeed())

		By("overwriting a managed label")
		ns := controller.GetNamespace(ctx, k8sClient, nsName, false)
		ns.Labels[pkgctrl.ProjectNameLabel] = "drifted"
		Expect(k8sClient.Update(ctx, ns)).To(Succeed())

		By("waiting for the label to be restored")
		Eventually(func(g Gomega) {
			ns := controller.GetNamespace(ctx, k8sClient, nsName, false)
			g.Expect(ns.Labels).To(HaveKeyWithValue(pkgctrl.ProjectNameLabel, projectName))
		}, timeout, interval).Should(Succeed())
	})

	It("deletes the project namespace and waits for its termination", func() {
		const projectName = "proj-delete"
		project := controller.CreateProject(ctx, k8sClient, projectName)

		nsName := konfidence.ProjectNamespacePrefix + projectName
		Eventually(func(g Gomega) {
			expectManagedNamespace(g, controller.GetNamespace(ctx, k8sClient, nsName, true), projectName)
		}, timeout, interval).Should(Succeed())

		By("deleting the Project")
		Expect(k8sClient.Delete(ctx, project)).To(Succeed())

		// Envtest runs no namespace controller, so the namespace never finishes
		// terminating: assert that its deletion was initiated and that the
		// controller correctly holds the finalizer until the namespace is gone.
		Eventually(func(g Gomega) {
			ns := controller.GetNamespace(ctx, k8sClient, nsName, true)
			g.Expect(ns).NotTo(BeNil())
			g.Expect(ns.DeletionTimestamp.IsZero()).To(BeFalse(), "namespace deletion should have been initiated")
		}, timeout, interval).Should(Succeed())

		remaining := controller.GetProject(ctx, k8sClient, projectName, false)
		Expect(remaining.Finalizers).To(ContainElement("konfidence.cloud/project-finalizer"))

		By("surfacing the wait as a status condition")
		Eventually(func(g Gomega) {
			project := controller.GetProject(ctx, k8sClient, projectName, false)
			expectCondition(g, project, konfidence.ProjectReadyCondition, metav1.ConditionFalse, konfidence.ProjectTerminatingReason)
			expectCondition(g, project, konfidence.ProjectNamespaceReadyCondition, metav1.ConditionFalse, konfidence.ProjectNamespaceTerminatingReason)
		}, timeout, interval).Should(Succeed())

		controller.CleanupProject(ctx, k8sClient, projectName)
	})
})
