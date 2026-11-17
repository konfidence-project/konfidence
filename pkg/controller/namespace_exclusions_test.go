package utils_test

import (
	"context"
	"errors"
	"strings"
	"time"

	utils "github.com/konfidence-project/konfidence/pkg/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Namespace exclusions", func() {
	Describe("Parsing", func() {
		DescribeTable("parses the comma-separated namespace list", func(spec string, namespaces []string) {
			policy, err := utils.ParseNamespaceExclusions(spec)
			Expect(err).NotTo(HaveOccurred())
			Expect(policy.Contains("")).To(BeFalse())
			Expect(policy.Contains("allowed")).To(BeFalse())
			for _, namespace := range namespaces {
				Expect(policy.Contains(namespace)).To(BeTrue())
			}
		},
			Entry("empty list", "", []string{}),
			Entry("blank entries", " , ", []string{}),
			Entry("one namespace", "fixtures", []string{"fixtures"}),
			Entry("whitespace, duplicates and empty entries", " fixtures,other, fixtures,, ", []string{"fixtures", "other"}),
		)

		DescribeTable("rejects invalid namespace names", func(spec string) {
			_, err := utils.ParseNamespaceExclusions(spec)
			Expect(err).To(HaveOccurred())
		},
			Entry("uppercase", "UPPERCASE"),
			Entry("underscore", "bad_name"),
			Entry("leading hyphen", "-bad"),
			Entry("too long", strings.Repeat("a", 64)),
		)
	})

	Describe("Reconciliation wrapper", func() {
		var (
			policy   utils.NamespaceExclusions
			ctx      context.Context
			calls    int
			sentinel error
			expected reconcile.Result
		)

		BeforeEach(func() {
			var err error
			policy, err = utils.ParseNamespaceExclusions("fixtures")
			Expect(err).NotTo(HaveOccurred())
			ctx = context.WithValue(context.Background(), namespaceContextKey{}, "value")
			calls = 0
			sentinel = errors.New("delegate failure")
			expected = reconcile.Result{RequeueAfter: 17 * time.Second}
		})

		It("skips excluded requests without requeuing", func() {
			delegate := reconcile.Func(func(context.Context, reconcile.Request) (reconcile.Result, error) {
				calls++
				return expected, sentinel
			})
			request := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "fixtures", Name: "stage"}}

			result, err := policy.Wrap(delegate).Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
			Expect(calls).To(BeZero())
		})

		DescribeTable("preserves delegate arguments, results and errors", func(namespace string) {
			request := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: namespace, Name: "stage"}}
			delegate := reconcile.Func(func(gotCtx context.Context, gotReq reconcile.Request) (reconcile.Result, error) {
				calls++
				Expect(gotCtx).To(BeIdenticalTo(ctx))
				Expect(gotReq).To(Equal(request))
				return expected, sentinel
			})

			result, err := policy.Wrap(delegate).Reconcile(ctx, request)
			Expect(err).To(BeIdenticalTo(sentinel))
			Expect(result).To(Equal(expected))
			Expect(calls).To(Equal(1))
		},
			Entry("allowed namespace", "allowed"),
			Entry("cluster-scoped request", ""),
		)

		It("calls the delegate for the zero-value policy", func() {
			delegate := reconcile.Func(func(context.Context, reconcile.Request) (reconcile.Result, error) {
				calls++
				return reconcile.Result{}, nil
			})

			_, err := (utils.NamespaceExclusions{}).Wrap(delegate).Reconcile(ctx, reconcile.Request{})
			Expect(err).NotTo(HaveOccurred())
			Expect(calls).To(Equal(1))
		})
	})
})

type namespaceContextKey struct{}
