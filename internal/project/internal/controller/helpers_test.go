//nolint:staticcheck // ST1001: allow dot-import for test utils using Gomega
package controller

import (
	"context"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	helperTimeout  = time.Second * 10
	helperInterval = time.Millisecond * 250
)

// NewProject builds a Project with the given name, applying any mutators.
func NewProject(name string, mutators ...func(*konfidence.Project)) *konfidence.Project {
	project := &konfidence.Project{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "konfidence.cloud/v1alpha1",
			Kind:       konfidence.ProjectKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	for _, mutate := range mutators {
		mutate(project)
	}
	return project
}

// CreateProject creates a Project with the given name and mutators.
func CreateProject(ctx context.Context, k8sClient client.Client, name string, mutators ...func(*konfidence.Project)) *konfidence.Project {
	project := NewProject(name, mutators...)
	Expect(k8sClient.Create(ctx, project)).To(Succeed())
	return project
}

// GetProject fetches a Project by name. With opt set, a missing Project
// returns nil instead of failing the test.
func GetProject(ctx context.Context, k8sClient client.Client, name string, opt bool) *konfidence.Project {
	project := &konfidence.Project{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, project)
	if opt && errors.IsNotFound(err) {
		return nil
	}
	Expect(err).NotTo(HaveOccurred())
	return project
}

// GetNamespace fetches a Namespace by name. With opt set, a missing Namespace
// returns nil instead of failing the test.
func GetNamespace(ctx context.Context, k8sClient client.Client, name string, opt bool) *corev1.Namespace {
	ns := &corev1.Namespace{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, ns)
	if opt && errors.IsNotFound(err) {
		return nil
	}
	Expect(err).NotTo(HaveOccurred())
	return ns
}

// CleanupProject deletes the Project and waits until it is gone. Envtest runs
// no namespace controller, so the project namespace never finishes
// terminating; once its deletion has been initiated, the controller finalizer
// is stripped so the Project itself can be released. Use unique project (and
// namespace) names per spec, since terminating namespaces linger.
func CleanupProject(ctx context.Context, k8sClient client.Client, name string) {
	project := GetProject(ctx, k8sClient, name, true)
	if project == nil {
		return
	}
	Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, project))).To(Succeed())

	if nsName := project.Status.Namespace; nsName != "" {
		Eventually(func(g Gomega) {
			ns := GetNamespace(ctx, k8sClient, nsName, true)
			if ns != nil {
				g.Expect(ns.DeletionTimestamp.IsZero()).To(BeFalse())
			}
		}, helperTimeout, helperInterval).Should(Succeed())
	}

	Eventually(func(g Gomega) {
		project := GetProject(ctx, k8sClient, name, true)
		if project == nil {
			return
		}
		if controllerutil.RemoveFinalizer(project, projectFinalizer) {
			g.Expect(client.IgnoreNotFound(k8sClient.Update(ctx, project))).To(Succeed())
		}
		g.Expect(GetProject(ctx, k8sClient, name, true)).To(BeNil())
	}, helperTimeout, helperInterval).Should(Succeed())
}
