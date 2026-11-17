package gc

import (
	"context"
	"testing"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGarbageCollectorSkipsExcludedNamespaces(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := konfidence.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	policy, err := pkgctrl.ParseNamespaceExclusions("fixtures")
	if err != nil {
		t.Fatal(err)
	}
	retained := &konfidence.StageVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "fixed", Namespace: "fixtures"},
		Spec:       konfidence.StageVersionSpec{StageRef: &konfidence.StageReference{Name: "stage"}},
	}
	unused := &konfidence.StageVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "unused", Namespace: "allowed"},
		Spec:       konfidence.StageVersionSpec{StageRef: &konfidence.StageReference{Name: "stage"}},
	}
	// A fixture usage must not interfere with collection in an allowed namespace.
	fixtureUsage := &konfidence.StageVersionUsage{
		ObjectMeta: metav1.ObjectMeta{Name: "usage", Namespace: "fixtures"},
		Status:     konfidence.StageVersionUsageStatus{ResolvedStageVersions: []string{"unused"}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(retained, unused, fixtureUsage,
		&konfidence.Stage{ObjectMeta: metav1.ObjectMeta{Name: "stage", Namespace: "allowed"}},
	).Build()
	collector := &StageVersionGarbageCollector{
		Client: c, Interval: time.Millisecond,
		ExcludedNamespaces: []pkgctrl.NamespaceExclusions{policy}, Recorder: events.NewFakeRecorder(10),
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- collector.Start(ctx) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	deadline := time.Now().Add(3 * time.Second)
	for {
		err := c.Get(ctx, client.ObjectKeyFromObject(unused), &konfidence.StageVersion{})
		if apierrors.IsNotFound(err) {
			break
		}
		if err != nil || time.Now().After(deadline) {
			t.Fatalf("allowed unused version was not collected: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(retained), &konfidence.StageVersion{}); err != nil {
		t.Fatalf("fixture version changed: %v", err)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(fixtureUsage), &konfidence.StageVersionUsage{}); err != nil {
		t.Fatalf("fixture usage changed: %v", err)
	}
}
