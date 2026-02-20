package crypto

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	_ RSACredentialProvider = (*ConfigMapTrustAnchorProvider)(nil)
	_ RSACredentialProvider = (*SecretSigningCredentialsProvider)(nil)
)

// RSACredentialProvider provides RSA credentials for OCM signing and verification.
// The returned map[string]string is compatible with ocm's signing.Signer and signing.Verifier.
type RSACredentialProvider interface {
	// Get returns the credentials or an error if retrieval fails.
	Get(ctx context.Context) (map[string]string, error)
}

// startCredentialInformer sets up a Kubernetes informer for credential providers.
// It watches for changes to the specified resource type and calls refresh on updates.
// When the context is cancelled, it stops the informer and signals done.
func startCredentialInformer[T client.Object](
	ctx context.Context,
	mgr ctrl.Manager,
	log logr.Logger,
	done chan struct{},
	refresh func(obj any),
) error {
	var zero T
	inf, err := mgr.GetCache().GetInformer(ctx, zero)
	if err != nil {
		return fmt.Errorf("get %T informer: %w", zero, err)
	}
	registration, err := inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { refresh(obj) },
		UpdateFunc: func(oldObj, newObj interface{}) { refresh(newObj) },
		DeleteFunc: func(obj interface{}) {},
	})
	if err != nil {
		return fmt.Errorf("add event handler: %w", err)
	}
	go func() {
		<-ctx.Done()
		close(done)
		log.Info("provider stopping")
		if err := inf.RemoveEventHandler(registration); err != nil {
			log.Error(err, "remove event handler failed")
		}
	}()
	return nil
}
