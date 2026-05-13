package crypto

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ConfigMapTrustAnchorProvider is an implementation of RSACredentialProvider.
// It provides and updates a trust anchor in the form of a tls.crt entry in a specified ConfigMap.
// It can be used with an RSAVerifier to provide an additional trust anchor
// for signature verification in addition to the system trust store.
type ConfigMapTrustAnchorProvider struct {
	done    chan struct{}
	log     logr.Logger
	cfg     types.NamespacedName
	data    atomic.Value
	started atomic.Bool
}

// ConfigMapTrustAnchorProviderOption configures a ConfigMapTrustAnchorProvider.
type ConfigMapTrustAnchorProviderOption func(*ConfigMapTrustAnchorProvider)

// WithTrustAnchorProviderLogger sets the logger for the provider.
func WithTrustAnchorProviderLogger(log logr.Logger) ConfigMapTrustAnchorProviderOption {
	return func(p *ConfigMapTrustAnchorProvider) {
		p.log = log
	}
}

// WithNamedTrustAnchorProviderLogger decorates the logger with the standard provider name "trust-anchor-provider".
func WithNamedTrustAnchorProviderLogger(log logr.Logger) ConfigMapTrustAnchorProviderOption {
	return func(p *ConfigMapTrustAnchorProvider) {
		p.log = log.WithName("trust-anchor-provider")
	}
}

func defaultConfigMapTrustAnchorProviderOptions() *ConfigMapTrustAnchorProvider {
	return &ConfigMapTrustAnchorProvider{
		log: logr.Discard(),
	}
}

// Get returns the current credentials. Returns nil if SetupWithManager has not been called.
func (p *ConfigMapTrustAnchorProvider) Get(ctx context.Context) (map[string]string, error) {
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

// SetupWithManager initializes the provider by fetching the ConfigMap and starting a background
// informer to watch for updates. If Get is called before SetupWithManager, it returns nil.
// If ctx is done, the provider stops updates and Get will return an error.
// Returns an error if called more than once or if initialization fails.
func (p *ConfigMapTrustAnchorProvider) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	if !p.started.CompareAndSwap(false, true) {
		return fmt.Errorf("provider already started")
	}
	p.done = make(chan struct{})
	data := &v1.ConfigMap{}
	// Use APIReader here as SetupWithManager might be called during controller startup
	// before the manager's cache is synced
	if err := mgr.GetAPIReader().Get(ctx, p.cfg, data); err != nil {
		return fmt.Errorf("get configmap %q: %w", p.cfg, err)
	}
	if err := p.ingest(data); err != nil {
		return fmt.Errorf("ingest configmap %q: %w", p.cfg, err)
	}
	inf, err := mgr.GetCache().GetInformer(ctx, &v1.ConfigMap{})
	if err != nil {
		return fmt.Errorf("get informer for configmap %q: %w", p.cfg, err)
	}
	if err := startCredentialInformer(ctx, inf, p.log, p.done, p.refresh); err != nil {
		return fmt.Errorf("start informer for configmap %q: %w", p.cfg, err)
	}
	return nil
}

func (p *ConfigMapTrustAnchorProvider) refresh(obj any) {
	cfg, ok := obj.(*v1.ConfigMap)
	if !ok {
		p.log.Error(
			fmt.Errorf("unexpected object type: %T, expected *v1.ConfigMap", obj),
			"refresh failed, skipping update")
		return
	}
	if cfg.Namespace != p.cfg.Namespace || cfg.Name != p.cfg.Name {
		return
	}
	if err := p.ingest(cfg); err != nil {
		p.log.Error(err, "ingest failed, skipping update")
		return
	}
	p.log.Info("credentials refreshed")
}

func (p *ConfigMapTrustAnchorProvider) ingest(cfg *v1.ConfigMap) error {
	if cfg == nil {
		return fmt.Errorf("configmap is nil")
	}
	cert, ok := cfg.Data["tls.crt"]
	if !ok {
		return fmt.Errorf("key 'tls.crt' not found")
	}
	// unfortunately this constant is placed under signing/handler/internal so we have to duplicate it here
	update := map[string]string{
		"public_key_pem": cert,
	}
	p.data.Store(update)
	return nil
}

// NewConfigMapTrustAnchorProvider creates a new ConfigMapTrustAnchorProvider.
func NewConfigMapTrustAnchorProvider(
	cfg types.NamespacedName,
	opts ...ConfigMapTrustAnchorProviderOption,
) *ConfigMapTrustAnchorProvider {
	p := defaultConfigMapTrustAnchorProviderOptions()
	for _, opt := range opts {
		opt(p)
	}
	p.cfg = cfg
	p.log = p.log.WithValues("configmap", cfg)
	return p
}
