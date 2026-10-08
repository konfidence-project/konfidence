package controller

import (
	"context"
	"encoding/json"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type noopRecorder struct{}

func (noopRecorder) Eventf(_ runtime.Object, _ runtime.Object, _, _, _, _ string, _ ...interface{}) {}

var _ events.EventRecorder = noopRecorder{}

func newReconciler(objects ...client.Object) (*VectorDeploymentReconciler, client.Client) {
	GinkgoHelper()

	scheme := runtime.NewScheme()
	Expect(konfidence.AddToScheme(scheme)).To(Succeed())
	Expect(corev1.AddToScheme(scheme)).To(Succeed())

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(&konfidence.VectorDeployment{}, &konfidence.VectorData{}).
		Build()
	return &VectorDeploymentReconciler{Client: c, Scheme: scheme, Recorder: noopRecorder{}}, c
}

func newVD(name string, results map[string]konfidence.ComponentDeploymentResults) *konfidence.VectorDeployment {
	return &konfidence.VectorDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "landscape-a", UID: types.UID("uid-" + name), Generation: 1},
		Spec:       konfidence.VectorDeploymentSpec{Vector: "https://example/vector:1.0.0"},
		Status:     konfidence.VectorDeploymentStatus{DeploymentResults: results},
	}
}

var _ = Describe("VectorData internals", func() {
	It("creates VectorData with the split envelope", func() {
		envelope := []byte(`{"features":{"darkMode":true},"authored":{"db":{"host":"mysql"}}}`)
		results := map[string]konfidence.ComponentDeploymentResults{
			"github.com/acme/svc-a": {{Name: "result-1", Type: "test", Spec: runtime.RawExtension{Raw: []byte(`{"endpoint":"http://a"}`)}}},
		}
		vd := newVD("vd-1", results)
		r, c := newReconciler(vd)

		Expect(r.handleVectorData(context.Background(), vd, envelope, logf.Log)).To(Succeed())

		got := &konfidence.VectorData{}
		Expect(c.Get(context.Background(), types.NamespacedName{Name: "vd-1", Namespace: "landscape-a"}, got)).To(Succeed())
		Expect(got.Spec.Features).NotTo(BeNil())
		Expect(string(got.Spec.Features.Raw)).To(Equal(`{"darkMode":true}`))
		Expect(got.Spec.Authored).NotTo(BeNil())
		Expect(string(got.Spec.Authored.Raw)).To(Equal(`{"db":{"host":"mysql"}}`))
		Expect(got.Spec.DeploymentResults).To(HaveLen(1))
		Expect(got.OwnerReferences).NotTo(BeEmpty())
		Expect(meta.IsStatusConditionTrue(vd.Status.Conditions, konfidence.VectorDataCreatedCondition)).To(BeTrue())
	})

	It("preserves multiple results per component", func() {
		results := map[string]konfidence.ComponentDeploymentResults{
			"github.com/acme/shop/storefront": {
				{Name: "storefront", Type: "http-k8s-service", Spec: runtime.RawExtension{Raw: []byte(`{"K8sName":"storefront-a1b2"}`)}},
				{Name: "storefront-admin", Type: "http-k8s-service", Spec: runtime.RawExtension{Raw: []byte(`{"K8sName":"storefront-admin-a1b2"}`)}},
			},
		}
		vd := newVD("vd-multi", results)
		r, c := newReconciler(vd)

		Expect(r.handleVectorData(context.Background(), vd, []byte(`{}`), logf.Log)).To(Succeed())

		got := &konfidence.VectorData{}
		Expect(c.Get(context.Background(), types.NamespacedName{Name: "vd-multi", Namespace: "landscape-a"}, got)).To(Succeed())
		entry := got.Spec.DeploymentResults["github.com/acme/shop/storefront"]
		Expect(entry).To(HaveLen(2))
		Expect(entry[0].Name).To(Equal("storefront"))
		Expect(entry[1].Name).To(Equal("storefront-admin"))
	})

	It("does not change an existing VectorData", func() {
		vd := newVD("vd-existing", nil)
		preExisting := &konfidence.VectorData{
			ObjectMeta: metav1.ObjectMeta{Name: "vd-existing", Namespace: "landscape-a"},
			Spec:       konfidence.VectorDataSpec{Features: &runtime.RawExtension{Raw: []byte(`{"prior":true}`)}},
		}
		r, c := newReconciler(vd, preExisting)

		Expect(r.handleVectorData(context.Background(), vd, []byte(`{"features":{"new":true}}`), logf.Log)).To(Succeed())

		got := &konfidence.VectorData{}
		Expect(c.Get(context.Background(), types.NamespacedName{Name: "vd-existing", Namespace: "landscape-a"}, got)).To(Succeed())
		Expect(string(got.Spec.Features.Raw)).To(Equal(`{"prior":true}`))
	})

	It("rejects an invalid envelope", func() {
		vd := newVD("vd-bad", nil)
		r, _ := newReconciler(vd)

		Expect(r.handleVectorData(context.Background(), vd, []byte("not json"), logf.Log)).To(HaveOccurred())
		cond := meta.FindStatusCondition(vd.Status.Conditions, konfidence.VectorDataCreatedCondition)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		Expect(cond.Reason).To(Equal("InvalidConfigPayload"))
	})

	It("tracks the VectorData implementor readiness state", func() {
		vd := newVD("vd-r", nil)
		vd.Status.ResultingVectorData = &konfidence.LocalObjectReference{Name: "vd-r"}
		cr := &konfidence.VectorData{ObjectMeta: metav1.ObjectMeta{Name: "vd-r", Namespace: "landscape-a"}}
		r, c := newReconciler(vd, cr)
		ctx := context.Background()

		ready, err := r.vectorDataIsReady(ctx, vd)
		Expect(err).NotTo(HaveOccurred())
		Expect(ready).To(BeFalse())

		meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
			Type: konfidence.VectorDataReadyCondition, Status: metav1.ConditionTrue, Reason: konfidence.VectorDataReasonMaterialized,
		})
		Expect(c.Status().Update(ctx, cr)).To(Succeed())

		ready, err = r.vectorDataIsReady(ctx, vd)
		Expect(err).NotTo(HaveOccurred())
		Expect(ready).To(BeTrue())
	})

	Describe("splitEnvelope", func() {
		It("handles an empty envelope", func() {
			features, authored, err := splitEnvelope(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(features).To(BeNil())
			Expect(authored).To(BeNil())
		})

		It("extracts features without authored configuration", func() {
			features, authored, err := splitEnvelope([]byte(`{"features":{"a":1}}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(features).NotTo(BeNil())
			Expect(authored).To(BeNil())
		})

		It("rejects invalid JSON", func() {
			_, _, err := splitEnvelope([]byte(`{`))
			Expect(err).To(HaveOccurred())
		})

		It("returns valid JSON in the RawExtension", func() {
			features, _, err := splitEnvelope([]byte(`{"features":{"x":1}}`))
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Valid(features.Raw)).To(BeTrue())
		})
	})
})
