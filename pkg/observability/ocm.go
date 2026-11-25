package observability

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	ConsumerVectorDeployment = "vector_deployment"
	ConsumerVectorAssembly   = "vector_assembly"
	CacheResultHit           = "hit"
	CacheResultMiss          = "miss"

	ManifestInvalidationDeleted           = "deleted"
	ManifestInvalidationEmptyReferences   = "empty_references"
	ManifestInvalidationUIDChanged        = "uid_changed"
	ManifestInvalidationGenerationChanged = "generation_changed"
	ManifestInvalidationReferencesChanged = "references_changed"

	ReusePolicyEnabled                = "enabled"
	ReusePolicyDisabled               = "disabled"
	ArtifactDeploymentResultCreated   = "created"
	ArtifactDeploymentResultReused    = "reused"
	ArtifactDeploymentResultCollision = "collision"
	ArtifactDeploymentResultError     = "error"

	ResultSuccess = "success"
	ResultFailure = "failure"
)

type Metrics struct {
	descriptorRequests      *prometheus.CounterVec
	descriptorDuration      *prometheus.HistogramVec
	localResourceReads      *prometheus.CounterVec
	localResourceBytes      *prometheus.CounterVec
	signatureRequests       *prometheus.CounterVec
	signatureCacheLookups   *prometheus.CounterVec
	signatureCrypto         *prometheus.CounterVec
	manifestCacheLookups    *prometheus.CounterVec
	manifestInvalidations   *prometheus.CounterVec
	manifestRetries         prometheus.Counter
	manifestDownloadsActive prometheus.Gauge
	artifactDeployments     *prometheus.CounterVec
	vectorCacheLookups      *prometheus.CounterVec
}

func New(registerer prometheus.Registerer) (*Metrics, error) {
	m := &Metrics{
		descriptorRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "konfidence",
				Subsystem: "ocm",
				Name:      "descriptor_requests_total",
				Help:      "Logical OCM component descriptor requests.",
			},
			[]string{"consumer", "descriptor_kind", "result"},
		),
		descriptorDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "konfidence",
				Subsystem: "ocm",
				Name:      "descriptor_request_duration_seconds",
				Help:      "Duration of logical OCM component descriptor requests.",
				Buckets:   prometheus.ExponentialBuckets(0.01, 2.5, 10),
			},
			[]string{"consumer", "descriptor_kind", "result"},
		),
		localResourceReads: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "konfidence",
				Subsystem: "ocm",
				Name:      "local_resource_reads_total",
				Help:      "Complete reads of OCM local resources.",
			},
			[]string{"consumer", "resource_class", "result"},
		),
		localResourceBytes: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "konfidence",
				Subsystem: "ocm",
				Name:      "local_resource_read_bytes_total",
				Help:      "Bytes read from OCM local resources.",
			},
			[]string{"consumer", "resource_class"},
		),
		signatureRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "konfidence",
				Subsystem: "ocm",
				Name:      "signature_requests_total",
				Help:      "Logical OCM signature verification requests.",
			},
			[]string{"consumer", "descriptor_kind", "result"},
		),
		signatureCacheLookups: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "konfidence",
				Subsystem: "ocm",
				Name:      "signature_cache_lookups_total",
				Help:      "OCM signature verification cache lookups.",
			},
			[]string{"result"},
		),
		signatureCrypto: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "konfidence",
				Subsystem: "ocm",
				Name:      "signature_crypto_operations_total",
				Help:      "Actual cryptographic signature operations.",
			},
			[]string{"result"},
		),
		manifestCacheLookups: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "konfidence",
				Subsystem: "vectordeployment",
				Name:      "manifest_cache_lookups_total",
				Help: "Artifact manifest result-cache lookups. A miss means no completed cached result was available; " +
					"repeated polls of one job are each misses.",
			},
			[]string{"result"},
		),
		manifestInvalidations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "konfidence",
				Subsystem: "vectordeployment",
				Name:      "manifest_cache_invalidations_total",
				Help:      "Artifact manifest job-cache invalidations.",
			},
			[]string{"reason"},
		),
		manifestRetries: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "konfidence",
				Subsystem: "vectordeployment",
				Name:      "manifest_job_retries_total",
				Help:      "Transient artifact manifest job retries.",
			},
		),
		manifestDownloadsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "konfidence",
				Subsystem: "vectordeployment",
				Name:      "manifest_downloads_active",
				Help:      "Currently active artifact manifest downloads.",
			},
		),
		artifactDeployments: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "konfidence",
				Subsystem: "vectordeployment",
				Name:      "artifact_deployments_total",
				Help:      "ArtifactDeployment handling results.",
			},
			[]string{"reuse_policy", "result"},
		),
		vectorCacheLookups: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "konfidence",
				Subsystem: "vectorassembly",
				Name:      "vector_cache_lookups_total",
				Help:      "Vector assembly LRU cache lookups.",
			},
			[]string{"result"},
		),
	}

	collectors := []prometheus.Collector{
		m.descriptorRequests,
		m.descriptorDuration,
		m.localResourceReads,
		m.localResourceBytes,
		m.signatureRequests,
		m.signatureCacheLookups,
		m.signatureCrypto,
		m.manifestCacheLookups,
		m.manifestInvalidations,
		m.manifestRetries,
		m.manifestDownloadsActive,
		m.artifactDeployments,
		m.vectorCacheLookups,
	}
	for _, collector := range collectors {
		if err := registerer.Register(collector); err != nil {
			return nil, fmt.Errorf("register metrics collector: %w", err)
		}
	}

	return m, nil
}

func (m *Metrics) ObserveDescriptorRequest(consumer, kind, result string, duration time.Duration) {
	if m == nil {
		return
	}
	m.descriptorRequests.WithLabelValues(consumer, kind, result).Inc()
	m.descriptorDuration.WithLabelValues(consumer, kind, result).Observe(duration.Seconds())
}

func (m *Metrics) ObserveLocalResourceRead(consumer, class, result string, bytes int) {
	if m == nil {
		return
	}
	m.localResourceReads.WithLabelValues(consumer, class, result).Inc()
	if result == "success" {
		m.localResourceBytes.WithLabelValues(consumer, class).Add(float64(bytes))
	}
}

func (m *Metrics) ObserveSignatureRequest(consumer, kind, result string) {
	if m != nil {
		m.signatureRequests.WithLabelValues(consumer, kind, result).Inc()
	}
}

func (m *Metrics) ObserveSignatureCacheLookup(result string) {
	if m != nil {
		m.signatureCacheLookups.WithLabelValues(result).Inc()
	}
}

func (m *Metrics) ObserveSignatureCrypto(result string) {
	if m != nil {
		m.signatureCrypto.WithLabelValues(result).Inc()
	}
}

func (m *Metrics) ObserveManifestCacheLookup(result string) {
	if m != nil {
		m.manifestCacheLookups.WithLabelValues(result).Inc()
	}
}

func (m *Metrics) ObserveManifestInvalidation(reason string) {
	if m != nil {
		m.manifestInvalidations.WithLabelValues(reason).Inc()
	}
}

func (m *Metrics) ObserveManifestRetry() {
	if m != nil {
		m.manifestRetries.Inc()
	}
}

func (m *Metrics) IncManifestDownloadsActive() {
	if m != nil {
		m.manifestDownloadsActive.Inc()
	}
}

func (m *Metrics) DecManifestDownloadsActive() {
	if m != nil {
		m.manifestDownloadsActive.Dec()
	}
}

func (m *Metrics) ObserveArtifactDeployment(policy, result string) {
	if m != nil {
		m.artifactDeployments.WithLabelValues(policy, result).Inc()
	}
}

func (m *Metrics) ObserveVectorCacheLookup(result string) {
	if m != nil {
		m.vectorCacheLookups.WithLabelValues(result).Inc()
	}
}
