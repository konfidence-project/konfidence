package controller_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/vectordeployment"
	"github.com/konfidence-project/konfidence/pkg/ocm/credentials"
	cryptopkg "github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	testocm "github.com/konfidence-project/konfidence/pkg/testutil/ocm"
	"github.com/konfidence-project/konfidence/pkg/testutil/pki"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var _ = Describe("VectorDeployment pki enabled", Ordered, Serial, func() {
	const (
		pkiTestNamespace  = "pki"
		pkiTimeout        = 30 * time.Second
		pkiInterval       = 250 * time.Millisecond
		pkiNegativeWindow = 5 * time.Second

		pkiVectorSigName   = "pki-v-sig"
		pkiArtifactSigName = "pki-a-sig"
		pkiOCISecretName   = "pki-oci-creds"
		pkiSigningSecret   = "pki-signing-creds"
	)

	var (
		pkiCtx              context.Context
		pkiCancel           context.CancelFunc
		pkiRegistryEndpoint string
		pkiOCMClient        pkgocm.Client
		pkiVectorKey        pki.RSAKeyPair
		pkiArtifactKey      pki.RSAKeyPair
		pkiK8sClient        client.Client
	)

	BeforeAll(func() {
		pkiCtx, pkiCancel = context.WithCancel(ctx)
		DeferCleanup(pkiCancel)

		var err error
		pkiManager, err := manager.New(cfg, manager.Options{
			Metrics:    metricsserver.Options{BindAddress: "0"},
			Controller: config.Controller{SkipNameValidation: new(true)},
			Cache:      cache.Options{DefaultNamespaces: map[string]cache.Config{pkiTestNamespace: {}}},
		})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(konfidence.AddToScheme(pkiManager.GetScheme())).To(gomega.Succeed())

		go func() {
			defer GinkgoRecover()
			gomega.Expect(pkiManager.Start(pkiCtx)).To(gomega.Succeed())
		}()

		zotConfigDir, err := filepath.Abs(filepath.Join(".", "test", "zot-config"))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		zotImage := fmt.Sprintf("ghcr.io/project-zot/zot-linux-%s:latest", runtime.GOARCH)
		container, err := testcontainers.Run(
			pkiCtx, zotImage,
			testcontainers.WithExposedPorts("5100/tcp"),
			testcontainers.WithFiles(
				testcontainers.ContainerFile{HostFilePath: filepath.Join(zotConfigDir, "zot-config.json"), ContainerFilePath: "/etc/zot/config.json", FileMode: 0o644},
				testcontainers.ContainerFile{HostFilePath: filepath.Join(zotConfigDir, "htpasswd"), ContainerFilePath: "/etc/zot/htpasswd", FileMode: 0o644},
			),
			testcontainers.WithWaitStrategy(
				wait.ForHTTP("/v2/").WithPort("5100/tcp").WithStatusCodeMatcher(func(s int) bool {
					return s == http.StatusOK || s == http.StatusUnauthorized
				}),
			),
		)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		DeferCleanup(func() { gomega.Expect(testcontainers.TerminateContainer(container)).To(gomega.Succeed()) })

		pkiRegistryEndpoint, err = container.Endpoint(pkiCtx, "")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		GinkgoWriter.Printf("PKI OCI registry running at: http://%s\n", pkiRegistryEndpoint)

		pkiVectorKey = pki.GenerateRSAKeyPair("pki-vector-key")
		pkiArtifactKey = pki.GenerateRSAKeyPair("pki-artifact-key")

		pkiK8sClient, err = client.New(cfg, client.Options{Scheme: pkiManager.GetScheme()})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		gomega.Expect(pkiK8sClient.Create(pkiCtx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: pkiTestNamespace},
		})).To(gomega.Succeed())

		gomega.Expect(pkiK8sClient.Create(pkiCtx, testocm.OCMConfigSecret(pkiSigningSecret, pkiTestNamespace,
			testocm.Bind(pkiVectorSigName, pkiVectorKey),
			testocm.Bind(pkiArtifactSigName, pkiArtifactKey),
		))).To(gomega.Succeed())

		ociSecret := &corev1.Secret{}
		gomega.Expect(pkiK8sClient.Create(pkiCtx, testocm.DockerConfigSecret(pkiOCISecretName, pkiTestNamespace,
			"user", "password", pkiRegistryEndpoint,
		))).To(gomega.Succeed())
		gomega.Expect(pkiK8sClient.Get(pkiCtx, types.NamespacedName{Name: pkiOCISecretName, Namespace: pkiTestNamespace}, ociSecret)).To(gomega.Succeed())

		resolver, err := credentials.ResolverFromRefs(pkiCtx, pkiK8sClient, pkiTestNamespace, []credentials.Ref{{Name: pkiOCISecretName}})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		pkiOCMClient, err = pkgocm.NewOciClientBuilder().WithLogger(ctrl.Log.WithName("pki-ocm-client")).WithResolver(resolver).Build(pkiCtx)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		gomega.Expect(os.Setenv(vectordeployment.VectorSignaturesEnv, pkiVectorSigName)).To(gomega.Succeed())
		gomega.Expect(os.Setenv(vectordeployment.ArtifactSignaturesEnv, pkiArtifactSigName)).To(gomega.Succeed())
		gomega.Expect(os.Setenv(vectordeployment.CredentialsSecretNameEnv, pkiSigningSecret)).To(gomega.Succeed())
		gomega.Expect(os.Setenv(vectordeployment.CredentialsSecretNsEnv, pkiTestNamespace)).To(gomega.Succeed())
		DeferCleanup(func() {
			gomega.Expect(os.Unsetenv(vectordeployment.VectorSignaturesEnv)).To(gomega.Succeed())
			gomega.Expect(os.Unsetenv(vectordeployment.ArtifactSignaturesEnv)).To(gomega.Succeed())
			gomega.Expect(os.Unsetenv(vectordeployment.CredentialsSecretNameEnv)).To(gomega.Succeed())
			gomega.Expect(os.Unsetenv(vectordeployment.CredentialsSecretNsEnv)).To(gomega.Succeed())
		})

		gomega.Expect(vectordeployment.SetupControllers(pkiCtx, pkiManager, ctrl.Log, vectordeployment.Options{
			OCISecret: ociSecret,
			Limiter:   cryptopkg.NewLimiter(0),
		})).To(gomega.Succeed())
	})

	AfterEach(func() {
		cleanupVectorDeploymentNamespace(pkiCtx, pkiK8sClient, pkiTestNamespace, pkiTimeout, pkiInterval)
	})

	It("should set VectorDownloadedCondition=True when vector and artifacts are both signed", func() {
		artifactRef := testocm.ParseRef(pkiRegistryEndpoint, "github.com/konfidence-project/pki-test/artifact-signed:v1.0.0")
		vectorRef := testocm.ParseRef(pkiRegistryEndpoint, "github.com/konfidence-project/pki-test/vector-signed:v1.0.0")

		testocm.PushSignedComponent(pkiCtx, pkiOCMClient, artifactRef, nil,
			testocm.Bind(pkiArtifactSigName, pkiArtifactKey),
		)
		testocm.PushSignedVector(pkiCtx, pkiOCMClient, vectorRef, []compref.Ref{artifactRef}, "latest",
			testocm.SampleVectorConfig(),
			testocm.Bind(pkiVectorSigName, pkiVectorKey),
		)

		vectorURL := fmt.Sprintf("http://%s//github.com/konfidence-project/pki-test/vector-signed:v1.0.0", pkiRegistryEndpoint)
		vd := createVectorDeployment(pkiCtx, pkiK8sClient, "vd-pki-signed", pkiTestNamespace, vectorURL)

		actualVD := &konfidence.VectorDeployment{}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(pkiK8sClient.Get(pkiCtx, types.NamespacedName{Name: vd.Name, Namespace: pkiTestNamespace}, actualVD)).To(gomega.Succeed())
			g.Expect(meta.IsStatusConditionTrue(actualVD.Status.Conditions, konfidence.VectorDownloadedCondition)).To(gomega.BeTrue())
		}, pkiTimeout, pkiInterval).Should(gomega.Succeed())
	})

	It("should not set VectorDownloadedCondition when vector has no signature", func() {
		vd := pushUnsignedVectorAndCreateDeployment(pkiCtx, pkiOCMClient, pkiK8sClient,
			pkiRegistryEndpoint, "pki-test", "vector-unsigned", pkiTestNamespace)

		actualVD := &konfidence.VectorDeployment{}
		gomega.Consistently(func(g gomega.Gomega) {
			g.Expect(pkiK8sClient.Get(pkiCtx, types.NamespacedName{Name: vd.Name, Namespace: pkiTestNamespace}, actualVD)).To(gomega.Succeed())
			g.Expect(meta.IsStatusConditionTrue(actualVD.Status.Conditions, konfidence.VectorDownloadedCondition)).To(gomega.BeFalse())
		}, pkiNegativeWindow, pkiInterval).Should(gomega.Succeed())
	})
})

var _ = Describe("VectorDeployment pki disabled", Ordered, Serial, func() {
	const (
		noVerifyNamespace = "no-verify"
		noVerifyTimeout   = 30 * time.Second
		noVerifyInterval  = 250 * time.Millisecond

		noVerifyOCISecretName = "no-verify-oci-creds"
	)

	var (
		noVerifyCtx              context.Context
		noVerifyCancel           context.CancelFunc
		noVerifyRegistryEndpoint string
		noVerifyOCMClient        pkgocm.Client
		noVerifyK8sClient        client.Client
	)

	BeforeAll(func() {
		noVerifyCtx, noVerifyCancel = context.WithCancel(ctx)
		DeferCleanup(noVerifyCancel)

		noVerifyManager, err := manager.New(cfg, manager.Options{
			Metrics:    metricsserver.Options{BindAddress: "0"},
			Controller: config.Controller{SkipNameValidation: new(true)},
			Cache:      cache.Options{DefaultNamespaces: map[string]cache.Config{noVerifyNamespace: {}}},
		})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(konfidence.AddToScheme(noVerifyManager.GetScheme())).To(gomega.Succeed())

		go func() {
			defer GinkgoRecover()
			gomega.Expect(noVerifyManager.Start(noVerifyCtx)).To(gomega.Succeed())
		}()

		zotConfigDir, err := filepath.Abs(filepath.Join(".", "test", "zot-config"))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		zotImage := fmt.Sprintf("ghcr.io/project-zot/zot-linux-%s:latest", runtime.GOARCH)
		container, err := testcontainers.Run(
			noVerifyCtx, zotImage,
			testcontainers.WithExposedPorts("5100/tcp"),
			testcontainers.WithFiles(
				testcontainers.ContainerFile{HostFilePath: filepath.Join(zotConfigDir, "zot-config.json"), ContainerFilePath: "/etc/zot/config.json", FileMode: 0o644},
				testcontainers.ContainerFile{HostFilePath: filepath.Join(zotConfigDir, "htpasswd"), ContainerFilePath: "/etc/zot/htpasswd", FileMode: 0o644},
			),
			testcontainers.WithWaitStrategy(
				wait.ForHTTP("/v2/").WithPort("5100/tcp").WithStatusCodeMatcher(func(s int) bool {
					return s == http.StatusOK || s == http.StatusUnauthorized
				}),
			),
		)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		DeferCleanup(func() { gomega.Expect(testcontainers.TerminateContainer(container)).To(gomega.Succeed()) })

		noVerifyRegistryEndpoint, err = container.Endpoint(noVerifyCtx, "")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		noVerifyK8sClient, err = client.New(cfg, client.Options{Scheme: noVerifyManager.GetScheme()})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		gomega.Expect(noVerifyK8sClient.Create(noVerifyCtx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: noVerifyNamespace},
		})).To(gomega.Succeed())

		ociSecret := &corev1.Secret{}
		gomega.Expect(noVerifyK8sClient.Create(noVerifyCtx, testocm.DockerConfigSecret(noVerifyOCISecretName, noVerifyNamespace,
			"user", "password", noVerifyRegistryEndpoint,
		))).To(gomega.Succeed())
		gomega.Expect(noVerifyK8sClient.Get(noVerifyCtx,
			types.NamespacedName{Name: noVerifyOCISecretName, Namespace: noVerifyNamespace}, ociSecret)).To(gomega.Succeed())

		resolver, err := credentials.ResolverFromRefs(noVerifyCtx, noVerifyK8sClient, noVerifyNamespace, []credentials.Ref{{Name: noVerifyOCISecretName}})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		noVerifyOCMClient, err = pkgocm.NewOciClientBuilder().WithLogger(ctrl.Log.WithName("no-verify-ocm-client")).WithResolver(resolver).Build(noVerifyCtx)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// No signature env vars set — SetupControllers will use NoopVerifiers.
		gomega.Expect(vectordeployment.SetupControllers(noVerifyCtx, noVerifyManager, ctrl.Log, vectordeployment.Options{
			OCISecret: ociSecret,
			Limiter:   cryptopkg.NewLimiter(0),
		})).To(gomega.Succeed())
	})

	AfterEach(func() {
		cleanupVectorDeploymentNamespace(noVerifyCtx, noVerifyK8sClient, noVerifyNamespace, noVerifyTimeout, noVerifyInterval)
	})

	It("should set VectorDownloadedCondition=True for an unsigned vector when verification is disabled", func() {
		vd := pushUnsignedVectorAndCreateDeployment(noVerifyCtx, noVerifyOCMClient, noVerifyK8sClient,
			noVerifyRegistryEndpoint, "no-verify-test", "vector", noVerifyNamespace)

		actualVD := &konfidence.VectorDeployment{}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(noVerifyK8sClient.Get(noVerifyCtx, types.NamespacedName{Name: vd.Name, Namespace: noVerifyNamespace}, actualVD)).To(gomega.Succeed())
			g.Expect(meta.IsStatusConditionTrue(actualVD.Status.Conditions, konfidence.VectorDownloadedCondition)).To(gomega.BeTrue())
		}, noVerifyTimeout, noVerifyInterval).Should(gomega.Succeed())
	})
})

func cleanupVectorDeploymentNamespace(
	ctx context.Context, k8sClient client.Client, namespace string,
	timeout, interval time.Duration,
) {
	gomega.Expect(k8sClient.DeleteAllOf(ctx, &konfidence.VectorDeployment{}, client.InNamespace(namespace))).To(gomega.Succeed())
	gomega.Expect(k8sClient.DeleteAllOf(ctx, &konfidence.ArtifactDeployment{}, client.InNamespace(namespace))).To(gomega.Succeed())
	gomega.Expect(k8sClient.DeleteAllOf(ctx, &konfidence.VectorAssignment{}, client.InNamespace(namespace))).To(gomega.Succeed())
	gomega.Eventually(func(g gomega.Gomega) {
		list := &konfidence.VectorDeploymentList{}
		g.Expect(k8sClient.List(ctx, list, client.InNamespace(namespace))).To(gomega.Succeed())
		g.Expect(list.Items).To(gomega.BeEmpty())
	}, timeout, interval).Should(gomega.Succeed())
}

func createVectorDeployment(ctx context.Context, k8sClient client.Client, name, namespace, vectorURL string) *konfidence.VectorDeployment {
	vd := &konfidence.VectorDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       konfidence.VectorDeploymentSpec{Vector: vectorURL},
	}
	gomega.Expect(k8sClient.Create(ctx, vd)).To(gomega.Succeed())
	return vd
}

func pushUnsignedVectorAndCreateDeployment(
	ctx context.Context, ocmClient pkgocm.Client, k8sClient client.Client,
	registryEndpoint, project, vectorName, namespace string,
) *konfidence.VectorDeployment {
	artifactRef := testocm.ParseRef(registryEndpoint, fmt.Sprintf("github.com/konfidence-project/%s/artifact:v1.0.0", project))
	vectorRef := testocm.ParseRef(registryEndpoint, fmt.Sprintf("github.com/konfidence-project/%s/%s:v1.0.0", project, vectorName))
	testocm.PushComponent(ctx, ocmClient, artifactRef, nil)
	testocm.PushVector(ctx, ocmClient, vectorRef, []compref.Ref{artifactRef}, "latest", testocm.SampleVectorConfig())
	vectorURL := fmt.Sprintf("http://%s//github.com/konfidence-project/%s/%s:v1.0.0", registryEndpoint, project, vectorName)
	return createVectorDeployment(ctx, k8sClient, fmt.Sprintf("vd-%s-%s", project, vectorName), namespace, vectorURL)
}
