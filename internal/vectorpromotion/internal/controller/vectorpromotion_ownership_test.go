package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func lifecyclePromotion(name string, sequence int64, cleared bool) *konfidence.VectorPromotion {
	p := &konfidence.VectorPromotion{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "project", UID: types.UID(name)},
		Spec: konfidence.VectorPromotionSpec{
			VectorPromotionConfigName: "config", Sequence: sequence, Vector: testVector,
			Source: templateSource("source"), Target: stageTargetInLandscape("target", "landscape"),
			Gates: approvalGate(),
		},
	}
	state := konfidence.PromotionGateStatePending
	status := metav1.ConditionFalse
	if cleared {
		state = konfidence.PromotionGateStateSatisfied
		status = metav1.ConditionTrue
	}
	p.Status.Gates = []konfidence.PromotionGateStatus{{Type: konfidence.GateTypeExplicitApproval, State: state}}
	setCondition(konfidence.ConditionTypeGatesSatisfied, status, "Test", "test")(p)
	setCondition(konfidence.ConditionTypeSucceeded, metav1.ConditionUnknown, "Test", "test")(p)
	return p
}

func lifecycleClient(t *testing.T, objects ...client.Object) client.WithWatch {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := konfidence.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).
		WithStatusSubresource(&konfidence.VectorPromotion{}).
		WithIndex(&konfidence.VectorPromotion{}, promotionConfigNameField, func(obj client.Object) []string {
			return []string{obj.(*konfidence.VectorPromotion).Spec.VectorPromotionConfigName}
		}).Build()
}

func TestPromotionOwnsSupersession(t *testing.T) {
	for _, test := range []struct {
		name                                                                      string
		currentCleared, newerCleared, newerStarted, newerFinished, wantSuperseded bool
		newerFailed, newerBypassed                                                bool
	}{
		{name: "waiting yields to started", newerCleared: true, newerStarted: true, wantSuperseded: true},
		{name: "waiting yields to completed", newerCleared: true, newerFinished: true, wantSuperseded: true},
		{name: "waiting yields to ready before execution starts", newerCleared: true, wantSuperseded: true},
		{name: "waiting ignores newer unapproved"},
		{name: "waiting yields to bypassed", newerBypassed: true, wantSuperseded: true},
		{name: "waiting yields to failed without cleared gates", newerFailed: true, wantSuperseded: true},
		{name: "cleared yields to cleared", currentCleared: true, newerCleared: true, wantSuperseded: true},
		{name: "cleared yields to completed", currentCleared: true, newerCleared: true, newerFinished: true, wantSuperseded: true},
		{name: "cleared yields to bypassed", currentCleared: true, newerBypassed: true, wantSuperseded: true},
		{name: "cleared yields to failed without cleared gates", currentCleared: true, newerFailed: true, wantSuperseded: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			ctx := context.Background()
			current := lifecyclePromotion("current", 1, test.currentCleared)
			newer := lifecyclePromotion("newer", 2, test.newerCleared)
			if test.newerStarted {
				now := metav1.Now()
				newer.Status.StartTime = &now
			}
			if test.newerFinished {
				finish(metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, "done")(newer)
			}
			if test.newerFailed {
				finish(metav1.ConditionFalse, "ExecutionFailed", "failed")(newer)
			}
			if test.newerBypassed {
				newer.Status.Override = &konfidence.PromotionOverride{Type: konfidence.PromotionOverrideTypeBypassGates}
			}
			landscape, stage := cachedExecutionTarget()
			base := lifecycleClient(t, current, newer, landscape, stage)
			before := &konfidence.VectorPromotion{}
			g.Expect(base.Get(ctx, client.ObjectKeyFromObject(newer), before)).To(gomega.Succeed())
			key := client.ObjectKeyFromObject(current)
			c := interceptor.NewClient(base, interceptor.Funcs{
				SubResourcePatch: func(ctx context.Context, c client.Client, subresource string,
					obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
					g.Expect(client.ObjectKeyFromObject(obj)).To(gomega.Equal(key), "must never patch sibling status")
					return c.SubResource(subresource).Patch(ctx, obj, patch, opts...)
				},
			})
			r := &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
			result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(base.Get(ctx, key, current)).To(gomega.Succeed())
			g.Expect(meta.IsStatusConditionFalse(current.Status.Conditions, konfidence.ConditionTypeSucceeded)).To(gomega.Equal(test.wantSuperseded))
			if test.wantSuperseded {
				g.Expect(
					meta.FindStatusCondition(current.Status.Conditions, konfidence.ConditionTypeSucceeded).Reason).To(gomega.Equal(konfidence.ReasonPromotionSuperseded))
				finished := current.DeepCopy()
				_, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(base.Get(ctx, key, current)).To(gomega.Succeed())
				g.Expect(current).To(gomega.Equal(finished), "terminal reconciliation must be a no-op")
			} else {
				g.Expect(result.RequeueAfter).To(gomega.Equal(promotionResyncInterval))
			}
			g.Expect(base.Get(ctx, client.ObjectKeyFromObject(newer), newer)).To(gomega.Succeed())
			g.Expect(newer).To(gomega.Equal(before))
			g.Expect(base.Get(ctx, client.ObjectKeyFromObject(stage), stage)).To(gomega.Succeed())
			g.Expect(stage.Spec.Vector).To(gomega.BeEmpty(), "a superseded or waiting promotion must not execute")
		})
	}
}

func TestRejectedPromotionNeverSupersedes(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		for _, cleared := range []bool{false, true} {
			t.Run(fmt.Sprintf("terminal=%t/current-cleared=%t", terminal, cleared), func(t *testing.T) {
				g := gomega.NewWithT(t)
				ctx := context.Background()
				current := lifecyclePromotion("current", 1, cleared)
				newer := lifecyclePromotion("rejected", 2, true)
				newer.Status.Override = &konfidence.PromotionOverride{Type: konfidence.PromotionOverrideTypeReject}
				if terminal {
					finish(metav1.ConditionFalse, konfidence.ReasonPromotionRejected, "rejected")(newer)
				}
				landscape, stage := cachedExecutionTarget()
				c := lifecycleClient(t, current, newer, landscape, stage)
				g.Expect(c.Get(ctx, client.ObjectKeyFromObject(newer), newer)).To(gomega.Succeed())
				r := &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
				key := client.ObjectKeyFromObject(current)
				_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(c.Get(ctx, key, current)).To(gomega.Succeed())
				condition := meta.FindStatusCondition(current.Status.Conditions, konfidence.ConditionTypeSucceeded)
				g.Expect(condition).NotTo(gomega.BeNil())
				if cleared {
					g.Expect(condition.Status).To(gomega.Equal(metav1.ConditionTrue))
				} else {
					g.Expect(condition.Status).To(gomega.Equal(metav1.ConditionUnknown))
				}
				g.Expect(neededByOlderPromotion(newer, []konfidence.VectorPromotion{*current})).To(gomega.BeFalse())
				unchanged := &konfidence.VectorPromotion{}
				g.Expect(c.Get(ctx, client.ObjectKeyFromObject(newer), unchanged)).To(gomega.Succeed())
				g.Expect(unchanged.Status).To(gomega.Equal(newer.Status))
			})
		}
	}
}

func TestPromotionWatchMapping(t *testing.T) {
	g := gomega.NewWithT(t)
	current := lifecyclePromotion("current", 3, true)
	older := lifecyclePromotion("older", 1, false)
	newer := lifecyclePromotion("newer", 4, true)
	finished := lifecyclePromotion("finished", 2, true)
	finish(metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, "done")(finished)
	otherConfig := lifecyclePromotion("other-config", 1, false)
	otherConfig.Spec.VectorPromotionConfigName = "different"
	otherNamespace := lifecyclePromotion("other-namespace", 1, false)
	otherNamespace.Namespace = "different"
	c := lifecycleClient(t, current, older, newer, finished, otherConfig, otherNamespace)
	r := &VectorPromotionReconciler{Client: c}
	request := func(p *konfidence.VectorPromotion) reconcile.Request {
		return reconcile.Request{NamespacedName: client.ObjectKeyFromObject(p)}
	}
	g.Expect(r.mapToActivePromotions(context.Background(), current)).To(gomega.ConsistOf(request(older), request(newer)))
	config := &konfidence.VectorPromotionConfig{ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: "project"}}
	g.Expect(r.mapToActivePromotions(context.Background(), config)).To(gomega.ConsistOf(request(older), request(current), request(newer)))
}

func TestPromotionSchedulingPredicate(t *testing.T) {
	g := gomega.NewWithT(t)
	p := lifecyclePromotion("current", 1, false)
	predicate := promotionSchedulingChanged()
	g.Expect(predicate.Create(event.CreateEvent{Object: p})).To(gomega.BeTrue())
	g.Expect(predicate.Delete(event.DeleteEvent{Object: p})).To(gomega.BeTrue())
	g.Expect(predicate.Generic(event.GenericEvent{Object: p})).To(gomega.BeFalse())

	for _, test := range []struct {
		name   string
		change func(*konfidence.VectorPromotion)
		want   bool
	}{
		{name: "cleared", want: true, change: func(p *konfidence.VectorPromotion) {
			setCondition(konfidence.ConditionTypeGatesSatisfied, metav1.ConditionTrue, "Test", "test")(p)
		}},
		{name: "bypassed", want: true, change: func(p *konfidence.VectorPromotion) {
			p.Status.Override = &konfidence.PromotionOverride{Type: konfidence.PromotionOverrideTypeBypassGates}
		}},
		{name: "started", want: true, change: func(p *konfidence.VectorPromotion) { now := metav1.Now(); p.Status.StartTime = &now }},
		{name: "finished", want: true, change: func(p *konfidence.VectorPromotion) { finish(metav1.ConditionTrue, "Test", "test")(p) }},
		{name: "condition message only", change: func(p *konfidence.VectorPromotion) { p.Status.Conditions[0].Message = "changed" }},
		{name: "gate provenance only", change: func(p *konfidence.VectorPromotion) { p.Status.Gates[0].SetBy = "alice" }},
		{name: "metadata only", change: func(p *konfidence.VectorPromotion) { p.Labels = map[string]string{"test": "test"} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			old := lifecyclePromotion("current", 1, false)
			current := old.DeepCopy()
			test.change(current)
			g.Expect(promotionSchedulingChanged().Update(event.UpdateEvent{ObjectOld: old, ObjectNew: current})).To(gomega.Equal(test.want))
		})
	}
}

func TestRetentionPreservesSupersessionEvidence(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()
	older := lifecyclePromotion("older", 1, false)
	newer := lifecyclePromotion("newer", 2, true)
	finish(metav1.ConditionTrue, konfidence.ReasonPromotionSucceeded, "done")(newer)
	expired := metav1.NewTime(time.Now().Add(-time.Hour))
	newer.Status.CompletionTime = &expired
	newer.Spec.TTLAfterFinished = &metav1.Duration{Duration: time.Second}
	keep := int32(0)
	config := &konfidence.VectorPromotionConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: "project"},
		Spec:       konfidence.VectorPromotionConfigSpec{KeepLastPromotions: &keep},
	}
	c := lifecycleClient(t, older, newer, config)
	ttl := &VectorPromotionTTLReconciler{Client: c}
	key := client.ObjectKeyFromObject(newer)
	result, err := ttl.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(result.RequeueAfter).To(gomega.Equal(siblingBlockedRequeueInterval))
	g.Expect(c.Get(ctx, key, newer)).To(gomega.Succeed(), "neither expired TTL nor a zero count bound may erase evidence yet")

	// A fresh reconciler, as after a restart, discovers supersession from state
	// alone. It does not depend on receiving the original completion event.
	r := &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
	_, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(older)})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(c.Get(ctx, client.ObjectKeyFromObject(older), older)).To(gomega.Succeed())
	g.Expect(meta.IsStatusConditionFalse(older.Status.Conditions, konfidence.ConditionTypeSucceeded)).To(gomega.BeTrue())
	_, err = ttl.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(apierrors.IsNotFound(c.Get(ctx, key, newer))).To(gomega.BeTrue())
}

func TestOutstandingRetentionOnlyTransitionsCurrentPromotion(t *testing.T) {
	g := gomega.NewWithT(t)
	ctx := context.Background()
	oldest := lifecyclePromotion("oldest", 1, false)
	middle := lifecyclePromotion("middle", 2, false)
	newest := lifecyclePromotion("newest", 3, false)
	keep := int32(1)
	config := &konfidence.VectorPromotionConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: "project"},
		Spec:       konfidence.VectorPromotionConfigSpec{KeepOutstandingPromotions: &keep},
	}
	c := lifecycleClient(t, oldest, middle, newest, config)
	r := &VectorPromotionReconciler{Client: c, Recorder: events.NewFakeRecorder(20)}
	// Newest-first execution used to finish all older objects from this request.
	for _, p := range []*konfidence.VectorPromotion{newest, middle, oldest} {
		before := &konfidence.VectorPromotionList{}
		g.Expect(c.List(ctx, before)).To(gomega.Succeed())
		_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(p)})
		g.Expect(err).NotTo(gomega.HaveOccurred())
		for i := range before.Items {
			sibling := &before.Items[i]
			if sibling.Name == p.Name {
				continue
			}
			after := &konfidence.VectorPromotion{}
			g.Expect(c.Get(ctx, client.ObjectKeyFromObject(sibling), after)).To(gomega.Succeed())
			g.Expect(after).To(gomega.Equal(sibling))
		}
	}
	for _, p := range []*konfidence.VectorPromotion{oldest, middle} {
		g.Expect(c.Get(ctx, client.ObjectKeyFromObject(p), p)).To(gomega.Succeed())
		reason := konfidence.ReasonPromotionOutstandingLimitReached
		if p.Name == oldest.Name {
			// The middle promotion already terminated with Succeeded=False,
			// so the oldest supersedes itself before applying its own limit.
			reason = konfidence.ReasonPromotionSuperseded
		}
		g.Expect(meta.FindStatusCondition(p.Status.Conditions, konfidence.ConditionTypeSucceeded).Reason).
			To(gomega.Equal(reason))
	}
	// Terminal siblings do not count toward the outstanding bound.
	_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(newest)})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(c.Get(ctx, client.ObjectKeyFromObject(newest), newest)).To(gomega.Succeed())
	g.Expect(meta.FindStatusCondition(newest.Status.Conditions, konfidence.ConditionTypeSucceeded).Status).To(gomega.Equal(metav1.ConditionUnknown))
}
