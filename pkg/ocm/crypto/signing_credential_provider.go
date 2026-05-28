package crypto

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SecretSigningCredentialsProvider is an implementation of RSACredentialProvider.
// It provides and updates signing credentials (tls.crt and tls.key) from a specified Secret.
// It can be used with an RSASigner to provide credentials for signing OCM descriptors.
type SecretSigningCredentialsProvider struct {
	done    chan struct{}
	log     logr.Logger
	secret  types.NamespacedName
	data    atomic.Value
	started atomic.Bool
}

// SecretSigningCredentialsProviderOption configures a SecretSigningCredentialsProvider.
type SecretSigningCredentialsProviderOption func(*SecretSigningCredentialsProvider)

// WithSigningProviderLogger sets the logger for the provider.
func WithSigningProviderLogger(log logr.Logger) SecretSigningCredentialsProviderOption {
	return func(p *SecretSigningCredentialsProvider) {
		p.log = log
	}
}

// WithNamedSigningProviderLogger decorates the logger with the standard provider name "signing-credentials-provider".
func WithNamedSigningProviderLogger(log logr.Logger) SecretSigningCredentialsProviderOption {
	return func(p *SecretSigningCredentialsProvider) {
		p.log = log.WithName("signing-credentials-provider")
	}
}

func defaultSecretSigningCredentialsProviderOptions() *SecretSigningCredentialsProvider {
	return &SecretSigningCredentialsProvider{
		log: logr.Discard(),
	}
}

// Get returns the current credentials. Returns nil if SetupWithManager has not been called.
func (p *SecretSigningCredentialsProvider) Get(ctx context.Context) (map[string]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.done:
		return nil, fmt.Errorf("provider stopped")
	default:
		if v := p.data.Load(); v != nil {
			return v.(map[string]string), nil
		}
		return nil, nil
	}
}

// SetupWithManager initializes the provider by fetching the Secret and starting a background
// informer to watch for updates. If Get is called before SetupWithManager, it returns nil.
// If ctx is done, the provider stops updates and Get will return an error.
// Returns an error if called more than once or if initialization fails.
func (p *SecretSigningCredentialsProvider) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	if !p.started.CompareAndSwap(false, true) {
		return fmt.Errorf("provider already started")
	}
	p.done = make(chan struct{})
	data := &corev1.Secret{}
	// Use APIReader here as SetupWithManager might be called during controller startup
	// before the manager's cache is synced
	if err := mgr.GetAPIReader().Get(ctx, p.secret, data); err != nil {
		return fmt.Errorf("get secret %q: %w", p.secret, err)
	}
	if err := p.ingest(data); err != nil {
		return fmt.Errorf("ingest secret %q: %w", p.secret, err)
	}
	inf, err := mgr.GetCache().GetInformer(ctx, &corev1.Secret{})
	if err != nil {
		return fmt.Errorf("get informer for secret: %w", err)
	}
	if err := startCredentialInformer(ctx, inf, p.log, p.done, p.refresh); err != nil {
		return fmt.Errorf("start informer for secret %q: %w", p.secret, err)
	}
	return nil
}

func (p *SecretSigningCredentialsProvider) refresh(obj any) {
	sec, ok := obj.(*corev1.Secret)
	if !ok {
		p.log.Error(
			fmt.Errorf("unexpected object type: %T, expected *v1.Secret", obj),
			"refresh failed, skipping update")
		return
	}
	if sec.Namespace != p.secret.Namespace || sec.Name != p.secret.Name {
		return
	}
	if err := p.ingest(sec); err != nil {
		p.log.Error(err, "ingest failed, skipping update")
		return
	}
	p.log.Info("credentials refreshed")
}

func (p *SecretSigningCredentialsProvider) ingest(sec *corev1.Secret) error {
	if sec == nil {
		return fmt.Errorf("secret is nil")
	}
	cert, certOk := sec.Data["tls.crt"]
	key, keyOk := sec.Data["tls.key"]
	if !certOk || !keyOk {
		return fmt.Errorf("keys 'tls.crt' and/or 'tls.key' not found")
	}
	// unfortunately these constants are placed under signing/handler/internal so we have to duplicate them here
	update := map[string]string{
		"public_key_pem":  string(cert),
		"private_key_pem": string(key),
	}
	p.data.Store(update)
	return nil
}

// NewSecretSigningCredentialsProvider creates a new SecretSigningCredentialsProvider.
func NewSecretSigningCredentialsProvider(
	secret types.NamespacedName,
	opts ...SecretSigningCredentialsProviderOption,
) *SecretSigningCredentialsProvider {
	p := defaultSecretSigningCredentialsProviderOptions()
	for _, opt := range opts {
		opt(p)
	}
	p.secret = secret
	return p
}
