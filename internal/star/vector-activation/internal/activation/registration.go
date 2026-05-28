package activation

import (
	"context"
	"fmt"

	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func GetRegistrations(ctx context.Context, c client.Client, namespace string) (*star.ActivationTaskRegistrationList, error) {
	log := logf.FromContext(ctx)
	registrationList := &star.ActivationTaskRegistrationList{}
	if err := c.List(ctx, registrationList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list activation task registrations: %w", err)
	}
	log.Info("read in activation task registrations", "count", len(registrationList.Items))
	return registrationList, nil
}
