package observability

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
)

var _ = Describe("OCM metrics", func() {
	var (
		registry *prometheus.Registry
		metrics  *Metrics
	)

	BeforeEach(func() {
		registry = prometheus.NewPedanticRegistry()

		var err error
		metrics, err = New(registry)
		Expect(err).NotTo(HaveOccurred())
	})

	It("registers all collectors", func() {
		metrics.ObserveDescriptorRequest(ConsumerVectorDeployment, "vector", ResultSuccess, time.Second)
		metrics.ObserveLocalResourceRead(ConsumerVectorDeployment, "vector_config", ResultSuccess, 1)
		metrics.ObserveSignatureRequest(ConsumerVectorDeployment, "vector", ResultSuccess)
		metrics.ObserveSignatureCacheLookup(CacheResultHit)
		metrics.ObserveSignatureCrypto(ResultSuccess)
		metrics.ObserveManifestCacheLookup(CacheResultMiss)
		metrics.ObserveManifestInvalidation(ManifestInvalidationDeleted)
		metrics.ObserveManifestRetry()
		metrics.IncManifestDownloadsActive()
		metrics.DecManifestDownloadsActive()
		metrics.ObserveArtifactDeployment(ReusePolicyEnabled, ArtifactDeploymentResultCreated)
		metrics.ObserveVectorCacheLookup(CacheResultHit)
		families, err := registry.Gather()
		Expect(err).NotTo(HaveOccurred())

		names := make([]string, 0, len(families))
		for _, family := range families {
			names = append(names, family.GetName())
		}

		Expect(names).To(ConsistOf(
			"konfidence_ocm_descriptor_requests_total",
			"konfidence_ocm_descriptor_request_duration_seconds",
			"konfidence_ocm_local_resource_reads_total",
			"konfidence_ocm_local_resource_read_bytes_total",
			"konfidence_ocm_signature_requests_total",
			"konfidence_ocm_signature_cache_lookups_total",
			"konfidence_ocm_signature_crypto_operations_total",
			"konfidence_vectordeployment_manifest_cache_lookups_total",
			"konfidence_vectordeployment_manifest_cache_invalidations_total",
			"konfidence_vectordeployment_manifest_job_retries_total",
			"konfidence_vectordeployment_manifest_downloads_active",
			"konfidence_vectordeployment_artifact_deployments_total",
			"konfidence_vectorassembly_vector_cache_lookups_total",
		))
	})

	It("returns an error for duplicate registration", func() {
		_, err := New(registry)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("register metrics collector"))
	})

	It("allows every method on a nil Metrics pointer", func() {
		var nilMetrics *Metrics

		Expect(func() {
			nilMetrics.ObserveDescriptorRequest(ConsumerVectorDeployment, "vector", ResultSuccess, time.Second)
			nilMetrics.ObserveLocalResourceRead(ConsumerVectorDeployment, "vector_config", ResultSuccess, 10)
			nilMetrics.ObserveSignatureRequest(ConsumerVectorDeployment, "vector", ResultSuccess)
			nilMetrics.ObserveSignatureCacheLookup(CacheResultHit)
			nilMetrics.ObserveSignatureCrypto(ResultSuccess)
			nilMetrics.ObserveManifestCacheLookup(CacheResultMiss)
			nilMetrics.ObserveManifestInvalidation(ManifestInvalidationDeleted)
			nilMetrics.ObserveManifestRetry()
			nilMetrics.IncManifestDownloadsActive()
			nilMetrics.DecManifestDownloadsActive()
			nilMetrics.ObserveArtifactDeployment(ReusePolicyEnabled, ArtifactDeploymentResultCreated)
			nilMetrics.ObserveVectorCacheLookup(CacheResultHit)
		}).NotTo(Panic())
	})

	It("records descriptor count and duration together", func() {
		metrics.ObserveDescriptorRequest(ConsumerVectorDeployment, "vector", ResultSuccess, 250*time.Millisecond)

		counter := metrics.descriptorRequests.WithLabelValues(ConsumerVectorDeployment, "vector", ResultSuccess)
		Expect(testutil.ToFloat64(counter)).To(Equal(float64(1)))

		observer, err := metrics.descriptorDuration.GetMetricWithLabelValues(ConsumerVectorDeployment, "vector", ResultSuccess)
		Expect(err).NotTo(HaveOccurred())

		prometheusMetric, ok := observer.(prometheus.Metric)
		Expect(ok).To(BeTrue())

		message := &dto.Metric{}
		Expect(prometheusMetric.Write(message)).To(Succeed())
		Expect(message.GetHistogram().GetSampleCount()).To(Equal(uint64(1)))
		Expect(message.GetHistogram().GetSampleSum()).To(BeNumerically("~", 0.25, 0.001))
	})

	It("records bytes only for successful reads", func() {
		metrics.ObserveLocalResourceRead(ConsumerVectorDeployment, "vector_config", ResultFailure, 100)
		metrics.ObserveLocalResourceRead(ConsumerVectorDeployment, "vector_config", ResultSuccess, 42)

		Expect(testutil.ToFloat64(metrics.localResourceReads.WithLabelValues(ConsumerVectorDeployment, "vector_config", ResultFailure))).To(Equal(float64(1)))
		Expect(testutil.ToFloat64(metrics.localResourceBytes.WithLabelValues(ConsumerVectorDeployment, "vector_config"))).To(Equal(float64(42)))
	})

	It("balances the active download gauge", func() {
		metrics.IncManifestDownloadsActive()
		metrics.IncManifestDownloadsActive()
		metrics.DecManifestDownloadsActive()
		metrics.DecManifestDownloadsActive()
		Expect(testutil.ToFloat64(metrics.manifestDownloadsActive)).To(BeZero())
	})
})
