package test_utils

import (
	"context"
	"reflect"

	"github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewFunc constructs a new zero-value object of the concrete K8s type.
type NewFunc[T client.Object] func() T

// Create creates the object and asserts success.
func Create[T client.Object](ctx context.Context, c client.Client, obj T) {
	gomega.Expect(c.Create(ctx, obj)).To(gomega.Succeed())
}

// Get fetches an object; if optional and NotFound returns nil (for pointer types).
// For non-optional calls it asserts no error.
func Get[T client.Object](ctx context.Context, c client.Client, name string, namespace string, obj T, allowNotFound bool) T {

	err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, obj)
	if allowNotFound && apierrors.IsNotFound(err) {
		var zero T
		return zero
	}
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "failed to get object %s", name)
	return obj
}

// Update updates the object and asserts success.
func Update(ctx context.Context, c client.Client, obj client.Object) {
	gomega.Expect(c.Update(ctx, obj)).To(gomega.Succeed())
}

// Delete deletes the object and asserts success.
func Delete[T client.Object](ctx context.Context, c client.Client, name string, namespace string) {
	obj := newOf[T]()
	obj.SetName(name)
	obj.SetNamespace(namespace)
	DeleteObj(ctx, c, obj)
}

func DeleteObj(ctx context.Context, c client.Client, obj client.Object) {
	err := c.Delete(ctx, obj)
	if apierrors.IsNotFound(err) {
		return
	}
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "failed to delete object %s", obj.GetName())
}

func DeleteAll[T client.Object, L client.ObjectList](ctx context.Context, c client.Client, opts ...client.ListOption) {
	list := newListOf[L]()
	List[L](ctx, c, list, opts...)

	items, err := meta.ExtractList(list)
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "failed to extract list items")

	for _, item := range items {
		obj, ok := item.(T)
		gomega.Expect(ok).To(gomega.BeTrue(), "list item is not a client.Object")

		DeleteObj(ctx, c, obj)
	}

}

// List lists objects into the provided list object.
func List[T client.ObjectList](ctx context.Context, c client.Client, list T, opts ...client.ListOption) T {
	gomega.Expect(c.List(ctx, list, opts...)).To(gomega.Succeed())
	return list
}

// newOf creates a new instance of T using reflection.
// T should be a concrete pointer type that implements client.Object (e.g., *v1.Pod).
func newOf[T client.Object]() T {
	t := reflect.TypeOf((*T)(nil)).Elem() // T's concrete type (usually a pointer)
	v := reflect.New(t.Elem())            // create pointer to the struct
	return v.Interface().(T)
}

func newListOf[L client.ObjectList]() L {
	t := reflect.TypeOf((*L)(nil)).Elem() // resolve the concrete type of L
	if t.Kind() != reflect.Ptr {
		panic("newListOf: L must be a pointer type implementing client.ObjectList")
	}
	v := reflect.New(t.Elem()) // allocate the underlying struct and return its pointer
	return v.Interface().(L)
}
