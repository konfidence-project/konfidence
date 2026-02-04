package ocm

import (
	"context"
	"fmt"
	"os"
	sysRuntime "runtime"
	"slices"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	ocmDescriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	rsahandler "ocm.software/open-component-model/bindings/go/rsa/signing/handler"
	rsav1alpha1 "ocm.software/open-component-model/bindings/go/rsa/signing/v1alpha1"
	"ocm.software/open-component-model/bindings/go/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CredentialSecretNameEnv      = "RSA_SIGNING_KEY_SECRET_NAME"
	CredentialSecretNamespaceEnv = "RSA_SIGNING_KEY_SECRET_NAMESPACE"
	signingAlgorithm             = rsav1alpha1.AlgorithmRSASSAPSS
)

var (
	_ Signer = (*RSASigner)(nil)
)

// Signer is an interface for signing OCM descriptors.
type Signer interface {
	// Sign signs the given OCM descriptor and adds the signatures to the descriptor's signatures.
	// If signing fails or the signature already exists, a non-nil error is returned.
	Sign(ctx context.Context, desc *ocmDescriptor.Descriptor) error
}

// RSASigner is the default implementation of the Signer interface for signing OCM descriptors.
type RSASigner struct {
	targetSignatures []string
	credStore        atomic.Value
	rsaHandler       *rsahandler.Handler
	rsaConfig        *rsav1alpha1.Config
	digester         Digester
}

func (s *RSASigner) Sign(ctx context.Context, desc *ocmDescriptor.Descriptor) error {
	dig, err := s.digester.GenerateDigest(ctx, desc)
	if err != nil {
		return fmt.Errorf("unable to sign ocm descriptor, generating digest failed: %w", err)
	}
	mux := &sync.RWMutex{}                          // sync access to the descriptors signatures
	creds := s.credStore.Load().(map[string]string) // load a snapshot of the credentials once for a single signing process
	if len(s.targetSignatures) == 1 {
		return s.sign(ctx, mux, creds, desc, dig, s.targetSignatures[0])
	}
	signerPool, ctx2 := errgroup.WithContext(ctx)
	signerPool.SetLimit(sysRuntime.GOMAXPROCS(0)) // no oversubscription on CPU bound signing tasks
	for _, sig := range s.targetSignatures {      // no loop var capture needed because we use Go 1.22+
		signerPool.Go(func() error { return s.sign(ctx2, mux, creds, desc, dig, sig) })
	}
	return signerPool.Wait()
}

func (s *RSASigner) sign(
	ctx context.Context,
	mux *sync.RWMutex,
	creds map[string]string,
	desc *ocmDescriptor.Descriptor,
	dig *ocmDescriptor.Digest,
	signatureName string) error {
	mux.RLock()
	if slices.ContainsFunc(desc.Signatures, func(sig ocmDescriptor.Signature) bool { return sig.Name == signatureName }) {
		mux.RUnlock()
		return fmt.Errorf("unable to sign ocm descriptor, signature with name %s already exists", signatureName)
	}
	mux.RUnlock()
	signatureInfo, err := s.rsaHandler.Sign(ctx, *dig, s.rsaConfig, creds)
	if err != nil {
		return fmt.Errorf(
			"unable to sign ocm descriptor, signing failed for signature with name %s: %w",
			signatureName, err)
	}
	mux.Lock()
	defer mux.Unlock()
	desc.Signatures = append(desc.Signatures, ocmDescriptor.Signature{
		Name:      signatureName,
		Digest:    *dig,
		Signature: signatureInfo,
	})
	return nil
}

func (s *RSASigner) onSecretUpdate(obj interface{}, ns, name string) {
	sec := obj.(*v1.Secret)                      // might panic but that's okay - since this would be a design time error anyway
	if sec.Namespace != ns || sec.Name != name { // short circuit if this is not the secret we care about
		return
	}
	creds, err := ocmCredsFromSecret(sec)
	if err != nil { // this might need alerting at some point - otherwise it will lead to corrupted ocm components
		ctrl.Log.
			Error(err,
				"unable to load ocm signing credentials from secret during informer event handling, secret refresh will be ignored",
				"secretNamespace", ns, "secretName", name)
	}
	s.credStore.Store(creds)
	ctrl.Log.Info("successfully updated ocm signing credentials from secret update")
}

func NewRSASigner(mgr ctrl.Manager, signatures ...string) (Signer, error) {
	if len(signatures) == 0 {
		return nil, fmt.Errorf("unable to create ocm signer: at least one target signature name must be provided")
	}
	if mgr == nil {
		return nil, fmt.Errorf("unable to create ocm signer: controller manager is nil")
	}
	rsaHandler, err := rsahandler.New(runtime.NewScheme(), false) // loading the systems trust store is only useful during verify
	if err != nil {
		return nil, fmt.Errorf("unable to create rsa handler for ocm signer: %w", err)
	}
	secretName := os.Getenv(CredentialSecretNameEnv)
	if secretName == "" {
		return nil, fmt.Errorf("unable to create ocm signer: environment variable %s is not set or empty", CredentialSecretNameEnv)
	}
	secretNamespace := os.Getenv(CredentialSecretNamespaceEnv)
	if secretNamespace == "" {
		return nil, fmt.Errorf("unable to create ocm signer: environment variable %s is not set or empty", CredentialSecretNamespaceEnv)
	}
	signer := &RSASigner{
		targetSignatures: signatures,
		credStore:        atomic.Value{},
		digester:         newOcmDigester(),
		rsaHandler:       rsaHandler,
		rsaConfig: &rsav1alpha1.Config{
			Type:                    runtime.NewVersionedType(rsav1alpha1.ConfigType, rsav1alpha1.Version),
			SignatureAlgorithm:      signingAlgorithm,
			SignatureEncodingPolicy: rsav1alpha1.SignatureEncodingPolicyPEM,
		},
	}
	if err := loadSecretIntoSigner(mgr.GetAPIReader(), signer, secretNamespace, secretName); err != nil {
		return nil, err
	}
	if err := setupInformerForSecret(mgr, signer, secretNamespace, secretName); err != nil {
		return nil, err
	}
	return signer, nil
}

// loadSecretIntoSigner loads the signing credentials from the specified secret and stores them in the RSASigner -
// used for initial loading before the informer picks up any changes to the secret
func loadSecretIntoSigner(reader client.Reader, signer *RSASigner, ns, name string) error {
	sec := &v1.Secret{}
	if err := reader.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: name}, sec); err != nil {
		return fmt.Errorf("unable to load secret for ocm signing credentials %s/%s: %w", ns, name, err)
	}
	creds, err := ocmCredsFromSecret(sec)
	if err != nil {
		return fmt.Errorf("unable to load ocm signing credentials from secret %s/%s: %w", ns, name, err)
	}
	signer.credStore.Store(creds)
	return nil
}

func setupInformerForSecret(mgr ctrl.Manager, signer *RSASigner, ns, name string) error {
	inf, err := mgr.GetCache().GetInformer(context.Background(), &v1.Secret{})
	if err != nil {
		return fmt.Errorf("unable to set up informer for signing credentials secret: %w", err)
	}
	if _, err := inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { signer.onSecretUpdate(obj, ns, name) },
		UpdateFunc: func(oldObj, newObj interface{}) { signer.onSecretUpdate(newObj, ns, name) },
		DeleteFunc: func(obj interface{}) {},
	}); err != nil {
		return fmt.Errorf("unable to add event handler for signing credentials secret informer: %w", err)
	}
	return nil
}

func ocmCredsFromSecret(sec *v1.Secret) (map[string]string, error) {
	cert, certOk := sec.Data["tls.crt"]
	key, keyOk := sec.Data["tls.key"]
	if !certOk || !keyOk {
		return nil, fmt.Errorf("signing secret %s/%s does not contain required keys 'tls.crt' and 'tls.key'", sec.Namespace, sec.Name)
	}
	return map[string]string{ // unfortunately these constants are placed under signing/handler/internal so we have to duplicate them here
		"public_key_pem":  string(cert),
		"private_key_pem": string(key),
	}, nil
}

// NoopSigner is a Signer implementation that does not perform any signing and returns nil for all operations.
// It's the goto way to disable signing.
type NoopSigner struct{}

func (n NoopSigner) Sign(ctx context.Context, desc *ocmDescriptor.Descriptor) error {
	return nil
}
