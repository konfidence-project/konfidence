package controller

import (
	"context"

	"github.com/go-logr/logr"
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	"github.com/konfidence-project/konfidence/pkg/observability"
	metrictest "github.com/konfidence-project/konfidence/pkg/testutil/metrics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("ArtifactDeployment metrics", func() {
	const namespace = "default"

	var (
		scheme       *runtime.Scheme
		promRegistry *prometheus.Registry
		metrics      *observability.Metrics
		vd           *konfidence.VectorDeployment
		artifact     resolvedArtifactManifest
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(konfidence.AddToScheme(scheme)).To(Succeed())

		promRegistry = prometheus.NewPedanticRegistry()

		var err error
		metrics, err = observability.New(promRegistry)
		Expect(err).NotTo(HaveOccurred())

		vd = &konfidence.VectorDeployment{
			Name:      "vector",
			Namespace: namespace,
			UID:       types.UID("vector-uid"),
		}

		artifact = resolvedArtifactManifest{
			Reference: compref.Ref{
				Component: "example.com/artifact",
				Version:   "1.0.0",
			},
			Manifest: ArtifactManifest{
				Type:       konfidence.MustArtifactType("artifact.example.test"),
				AllowReuse: true,
			},
		}
	})

	newReconciler := func(
		objects ...runtime.Object,
	) *VectorDeploymentReconciler {
		allObjects := make([]runtime.Object, 0, 1+len(objects))
		allObjects = append(allObjects, &corev1.Namespace{
			Name: namespace,
			Labels: map[string]string{
				pkgctrl.LandscapeNameLabel: "test",
			},
		})
		allObjects = append(allObjects, objects...)

		return &VectorDeploymentReconciler{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(allObjects...).
				Build(),
			Scheme:   scheme,
			Recorder: events.NewFakeRecorder(20),
			Metrics:  metrics,
		}
	}

	metricValue := func(policy, result string) float64 {
		value, err := metrictest.Value(promRegistry, "konfidence_vectordeployment_artifact_deployments_total",
			prometheus.Labels{
				"reuse_policy": policy,
				"result":       result,
			},
		)
		Expect(err).NotTo(HaveOccurred())
		return value
	}

	It("records creation without also recording reuse", func() {
		r := newReconciler()

		ready, err := r.handleArtifactDeployments(context.Background(), []resolvedArtifactManifest{artifact}, vd, logr.Discard())
		Expect(err).NotTo(HaveOccurred())
		Expect(ready).To(BeFalse())

		Expect(metricValue(observability.ReusePolicyEnabled, observability.ArtifactDeploymentResultCreated)).To(Equal(float64(1)))
		Expect(metricValue(observability.ReusePolicyEnabled, observability.ArtifactDeploymentResultReused)).To(BeZero())
	})

	It("records an existing unowned deployment as reused", func() {
		name, hash, err := ConstructArtifactDeploymentName(
			artifact.Reference.Component,
			artifact.Reference.Version,
			nil,
			0,
		)
		Expect(err).NotTo(HaveOccurred())

		existing := (&VectorDeploymentReconciler{}).
			constructArtifactDeployment(
				artifact.Reference,
				artifact.Manifest,
				vd,
				name,
				hash,
				nil,
				"test",
			)

		r := newReconciler(existing)

		_, err = r.handleArtifactDeployments(context.Background(), []resolvedArtifactManifest{artifact}, vd, logr.Discard())
		Expect(err).NotTo(HaveOccurred())

		Expect(metricValue(observability.ReusePolicyEnabled, observability.ArtifactDeploymentResultReused)).To(Equal(float64(1)))
		Expect(metricValue(observability.ReusePolicyEnabled, observability.ArtifactDeploymentResultCreated)).To(BeZero())
	})

	It("records a deterministic-name collision", func() {
		name, _, err := ConstructArtifactDeploymentName(
			artifact.Reference.Component,
			artifact.Reference.Version,
			nil,
			0,
		)
		Expect(err).NotTo(HaveOccurred())

		existing := &konfidence.ArtifactDeployment{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				pkgctrl.ArtifactComponentAnnotation: "another/component",
				pkgctrl.ArtifactVersionAnnotation:   "9.0.0",
			},
		}

		r := newReconciler(existing)

		_, err = r.handleArtifactDeployments(context.Background(), []resolvedArtifactManifest{artifact}, vd, logr.Discard())
		Expect(err).To(MatchError(errArtifactDeploymentCollision))

		Expect(metricValue(observability.ReusePolicyEnabled, observability.ArtifactDeploymentResultCollision)).To(Equal(float64(1)))
	})
})
