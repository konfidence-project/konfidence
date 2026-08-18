//nolint:staticcheck // ST1001: allow dot-import for test utils using Gomega
package controller

import (
	"context"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
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

// NewProjectNamespace builds a project namespace with appropriate labels.
func NewProjectNamespace(name, projectName string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				pkgctrl.ProjectTypeLabel: "project",
				pkgctrl.ProjectNameLabel: projectName,
			},
		},
	}
}

// CreateProjectNamespace creates a project namespace with the given name and project.
func CreateProjectNamespace(ctx context.Context, k8sClient client.Client, name, projectName string) *corev1.Namespace {
	ns := NewProjectNamespace(name, projectName)
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	return ns
}

// NewLandscape builds a Landscape with the given name and namespace, applying any mutators.
func NewLandscape(name, namespace string, mutators ...func(*konfidence.Landscape)) *konfidence.Landscape {
	landscape := &konfidence.Landscape{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "konfidence.cloud/v1alpha1",
			Kind:       konfidence.LandscapeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	for _, mutate := range mutators {
		mutate(landscape)
	}
	return landscape
}

// CreateLandscape creates a Landscape with the given name and namespace and mutators.
func CreateLandscape(ctx context.Context, k8sClient client.Client, name, namespace string, mutators ...func(*konfidence.Landscape)) *konfidence.Landscape {
	landscape := NewLandscape(name, namespace, mutators...)
	Expect(k8sClient.Create(ctx, landscape)).To(Succeed())
	return landscape
}

// GetLandscape fetches a Landscape by name and namespace. With opt set, a missing Landscape
// returns nil instead of failing the test.
func GetLandscape(ctx context.Context, k8sClient client.Client, name, namespace string, opt bool) *konfidence.Landscape {
	landscape := &konfidence.Landscape{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, landscape)
	if opt && errors.IsNotFound(err) {
		return nil
	}
	Expect(err).NotTo(HaveOccurred())
	return landscape
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

// CleanupLandscape deletes the Landscape and waits until it is gone. Envtest runs
// no namespace controller, so the landscape namespace never finishes
// terminating; we manually strip the finalizer so the Landscape can be released.
// Use unique landscape (and namespace) names per spec, since terminating
// namespaces linger.
func CleanupLandscape(ctx context.Context, k8sClient client.Client, name, namespace string) {
	landscape := GetLandscape(ctx, k8sClient, name, namespace, true)
	if landscape == nil {
		return
	}
	Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, landscape))).To(Succeed())

	// In envtest, namespace termination doesn't complete automatically,
	// so we skip waiting for the namespace deletion timestamp and go
	// straight to removing the finalizer so the test can clean up.
	Eventually(func(g Gomega) {
		landscape := GetLandscape(ctx, k8sClient, name, namespace, true)
		if landscape == nil {
			return
		}
		if controllerutil.RemoveFinalizer(landscape, landscapeFinalizer) {
			g.Expect(client.IgnoreNotFound(k8sClient.Update(ctx, landscape))).To(Succeed())
		}
		g.Expect(GetLandscape(ctx, k8sClient, name, namespace, true)).To(BeNil())
	}, helperTimeout, helperInterval).Should(Succeed())
}

// CleanupProjectNamespace deletes the project namespace.
func CleanupProjectNamespace(ctx context.Context, k8sClient client.Client, name string) {
	ns := GetNamespace(ctx, k8sClient, name, true)
	if ns != nil {
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, ns))).To(Succeed())
	}
}
