package ocm

import (
	"context"
	"fmt"
	"log/slog"
	sysRuntime "runtime"
	"slices"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	ocmDescriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	rsahandler "ocm.software/open-component-model/bindings/go/rsa/signing/handler"
	"ocm.software/open-component-model/bindings/go/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	_ Verifier = (*RSAVerifier)(nil)
)

// Verifier is an interface for verifying OCM descriptors.
type Verifier interface {
	// Verify verifies the signatures of the given OCM descriptors.
	// If any verification fails, an error is returned.
	Verify(ctx context.Context, descs ...*ocmDescriptor.Descriptor) error
}

// RSAVerifier is the default implementation of the Verifier interface for verifying OCM descriptors.
// Optionally an additional trust anchor can be used for verify ops - otherwise only the system trust store is used.
// The RSAVerifier relies on the signatures transporting the signers cert chain (v1alpha1.MediaTypePEM)
type RSAVerifier struct {
	rsaHandler       *rsahandler.Handler
	targetSignatures []string // ocm will provide a feature in the future that makes it possible to transport the target signatures via ocm configuration
	credStore        atomic.Value
}

func (o *RSAVerifier) Verify(ctx context.Context, descs ...*ocmDescriptor.Descriptor) error {
	creds := o.credStore.Load().(map[string]string) // same credentials for 1 Verify call - for consistency
	if len(descs) == 1 {
		return o.verify(ctx, creds, descs[0])
	}
	verifierPool, ctx2 := errgroup.WithContext(ctx)
	verifierPool.SetLimit(sysRuntime.GOMAXPROCS(0)) // no oversubscription on CPU bound verification tasks
	for _, t := range descs {                       // no loop var capture needed because we use Go 1.22+
		verifierPool.Go(func() error { return o.verify(ctx2, creds, t) })
	}
	return verifierPool.Wait()
}

func (o *RSAVerifier) verify(ctx context.Context, creds map[string]string, desc *ocmDescriptor.Descriptor) error {
	if err := signing.IsSafelyDigestible(&desc.Component); err != nil {
		return fmt.Errorf("ocm descriptor verification failed: descriptor is not safely digestible: %w", err)
	}
	toVerify := make([]ocmDescriptor.Signature, 0, len(o.targetSignatures))
	for _, targetSignature := range o.targetSignatures {
		if idx := slices.IndexFunc(desc.Signatures, func(sig ocmDescriptor.Signature) bool { return sig.Name == targetSignature }); idx == -1 {
			return fmt.Errorf("ocm descriptor verification failed: signature with name %s not found in descriptor", targetSignature)
		} else {
			toVerify = append(toVerify, desc.Signatures[idx])
		}
	}
	for _, sig := range toVerify {
		if err := signing.VerifyDigestMatchesDescriptor(ctx, desc, sig, slog.Default()); err != nil {
			return fmt.Errorf(
				"ocm descriptor verification failed: digest verification failed for signature with name %s: %w",
				sig.Name, err)
		}
		if err := o.rsaHandler.Verify(ctx, sig, nil, creds); err != nil {
			return fmt.Errorf(
				"ocm descriptor verification failed: signature verification failed for signature with name %s: %w",
				sig.Name, err)
		}
	}
	return nil
}

func (o *RSAVerifier) onAnchorUpdate(obj interface{}, ns, name string) {
	cfg := obj.(*v1.ConfigMap) // might panic but that's okay - since this would be a design time error anyway
	if cfg.Namespace != ns || cfg.Name != name {
		return
	}
	anchor, err := trustAnchorFromConfigMap(cfg)
	if err != nil { // future alerting needed probably
		ctrl.Log.Error(fmt.Errorf("unable to update ocm verifier trust anchor from configmap update: %w", err),
			"configmap_namespace", ns, "configmap_name", name)
		return
	}
	o.credStore.Store(anchor)
	ctrl.Log.Info("successfully updated ocm verifier trust anchor from configmap update",
		"configmap_namespace", ns, "configmap_name", name)
}

type RSAVerifierTrustAnchorConfig struct {
	configMapName      string
	configMapNamespace string
}

// NewRSAVerifier creates a new RSAVerifier instance.
// If anchorConfig is not nil, the RSAVerifier will load the trust anchor from the specified ConfigMap
// and set up an informer to watch for updates to the ConfigMap.
// If anchorConfig is nil, the RSAVerifier will only use the system trust store for verification.
func NewRSAVerifier(
	mgr ctrl.Manager,
	anchorConfig *RSAVerifierTrustAnchorConfig,
	signatureName string,
	additionalSignatureNames ...string) (Verifier, error) {
	rsaHandler, err := rsahandler.New(runtime.NewScheme(), true) // load system roots
	if err != nil {
		return nil, fmt.Errorf("unable to create rsa handler for ocm verifier: %w", err)
	}
	verifier := &RSAVerifier{
		rsaHandler:       rsaHandler,
		targetSignatures: append([]string{signatureName}, additionalSignatureNames...),
		credStore:        atomic.Value{},
	}
	if anchorConfig == nil {
		return verifier, nil
	}
	if err := loadTrustAnchorIntoVerifier(mgr.GetAPIReader(), verifier, anchorConfig.configMapNamespace, anchorConfig.configMapName); err != nil {
		return nil, err
	}
	if err := setupInformerForTrustAnchor(mgr, verifier, anchorConfig.configMapNamespace, anchorConfig.configMapName); err != nil {
		return nil, err
	}
	return verifier, nil
}

func loadTrustAnchorIntoVerifier(reader client.Reader, verifier *RSAVerifier, ns, name string) error {
	cfg := &v1.ConfigMap{}
	if err := reader.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: name}, cfg); err != nil {
		return fmt.Errorf("unable to load configmap for ocm verification trust anchor %s/%s: %w", ns, name, err)
	}
	creds, err := trustAnchorFromConfigMap(cfg)
	if err != nil {
		return fmt.Errorf("unable to load trust anchor for ocm verifier from configmap %s/%s: %w", ns, name, err)
	}
	verifier.credStore.Store(creds)
	return nil
}

func trustAnchorFromConfigMap(cfg *v1.ConfigMap) (map[string]string, error) {
	cert, ok := cfg.Data["tls.crt"]
	if !ok {
		return nil, fmt.Errorf(
			"configmap %s/%s: key tls.crt not found in configmap data",
			cfg.Namespace, cfg.Name)
	}
	return map[string]string{ // unfortunately this constant is placed under signing/handler/internal so we have to duplicate it here
		"public_key_pem": cert,
	}, nil
}

func setupInformerForTrustAnchor(mgr ctrl.Manager, verifier *RSAVerifier, ns, name string) error {
	inf, err := mgr.GetCache().GetInformer(context.Background(), &v1.ConfigMap{})
	if err != nil {
		return fmt.Errorf("unable to set up informer for ocm verification trust anchor configmap: %w", err)
	}
	if _, err := inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { verifier.onAnchorUpdate(obj, ns, name) },
		UpdateFunc: func(oldObj, newObj interface{}) { verifier.onAnchorUpdate(newObj, ns, name) },
		DeleteFunc: func(obj interface{}) {},
	}); err != nil {
		return fmt.Errorf("unable to add event handler for ocm verification trust anchor configmap informer: %w", err)
	}
	return nil
}

// NoopVerifier is a Verifier implementation that does not perform any verification and returns nil for all operations.
// It's the goto way to disable verification.
type NoopVerifier struct{}

func (n NoopVerifier) Verify(ctx context.Context, descs ...*ocmDescriptor.Descriptor) error {
	return nil
}
