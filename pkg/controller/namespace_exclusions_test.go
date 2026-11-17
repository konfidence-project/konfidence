package utils

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestParseNamespaceExclusions(t *testing.T) {
	for _, spec := range []string{"", " , ", "fixtures", " fixtures,other, fixtures,, "} {
		policy, err := ParseNamespaceExclusions(spec)
		if err != nil {
			t.Fatalf("parse %q: %v", spec, err)
		}
		if policy.Contains("") || policy.Contains("allowed") {
			t.Fatalf("unexpected exclusion for %q", spec)
		}
		if strings.Contains(spec, "fixtures") && !policy.Contains("fixtures") {
			t.Fatalf("missing fixtures for %q", spec)
		}
	}
	for _, spec := range []string{"UPPERCASE", "bad_name", "-bad", strings.Repeat("a", 64)} {
		if _, err := ParseNamespaceExclusions(spec); err == nil {
			t.Fatalf("accepted invalid namespace %q", spec)
		}
	}
}

func TestNamespaceWrapper(t *testing.T) {
	policy, err := ParseNamespaceExclusions("fixtures")
	if err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("delegate failure")
	expected := reconcile.Result{RequeueAfter: 17 * time.Second}
	for _, namespace := range []string{"fixtures", "allowed", ""} {
		t.Run(namespace, func(t *testing.T) {
			calls := 0
			ctx := context.WithValue(context.Background(), namespaceContextKey{}, namespace)
			request := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: namespace, Name: "stage"}}
			delegate := reconcile.Func(func(gotCtx context.Context, gotReq reconcile.Request) (reconcile.Result, error) {
				calls++
				if gotCtx != ctx || gotReq != request {
					t.Fatal("changed delegate arguments")
				}
				return expected, sentinel
			})
			result, gotErr := WrapExcludedNamespaces(delegate, policy).Reconcile(ctx, request)
			if namespace == "fixtures" {
				if calls != 0 || result != (reconcile.Result{}) || gotErr != nil {
					t.Fatal("excluded request invoked delegate or requeued")
				}
			} else if calls != 1 || result != expected || gotErr != sentinel {
				t.Fatal("allowed request did not preserve delegate behavior")
			}
		})
	}
}

type namespaceContextKey struct{}

func TestEmptyNamespacePolicyPreservesDelegate(t *testing.T) {
	calls := 0
	delegate := reconcile.Func(func(context.Context, reconcile.Request) (reconcile.Result, error) {
		calls++
		return reconcile.Result{}, nil
	})
	for _, wrapper := range []reconcile.Reconciler{WrapExcludedNamespaces(delegate), (NamespaceExclusions{}).Wrap(delegate)} {
		if _, err := wrapper.Reconcile(context.Background(), reconcile.Request{}); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Fatal("an empty exclusion policy skipped the delegate")
	}
}
