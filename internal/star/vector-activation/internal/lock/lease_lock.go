package lock

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DefaultLeaseTTL = 30 * time.Second

// AcquireResourceLease tries to acquire a lease for a given resource.
// It returns true if the lease was successfully acquired, false otherwise.
// This function creates the lease if it does not exist.
// The lease is associated with the provided owner object.
// The lease is identified by a combination of resourceType and owner name. This ensures that there is a unique lease per resource type and owner.
// The holder identity is a combination of controllerID and resourceId to uniquely identify the controller holding the lease for the specific resource.
func AcquireResourceLease(
	ctx context.Context, c client.Client, resourceId string, namespace string,
	controllerID string, resourceType string, owner metav1.Object,
) (bool, error) {
	leaseName := getLeaseName(resourceType, owner.GetName())
	holderIdentity := getHolderIdentity(controllerID, resourceId)
	lease, err := getLease(ctx, c, leaseName, namespace)

	now := time.Now()

	if err != nil {
		if apierrors.IsNotFound(err) {
			// If not found, create it
			lease = &coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Name:      leaseName,
					Namespace: namespace,
				},
				Spec: coordinationv1.LeaseSpec{
					HolderIdentity:       &holderIdentity,
					LeaseDurationSeconds: pointer(int32(DefaultLeaseTTL.Seconds())),
					AcquireTime:          &metav1.MicroTime{Time: now},
					RenewTime:            &metav1.MicroTime{Time: now},
					LeaseTransitions:     pointer(0),
				},
			}
			if err := controllerutil.SetOwnerReference(owner, lease, c.Scheme()); err != nil {
				return false, fmt.Errorf("failed to set owner reference on lease: %w", err)
			}
			if err := c.Create(ctx, lease); err != nil {
				return false, fmt.Errorf("failed to create lease: %w", err)
			}
			return true, nil
		}
		return false, err
	}

	// If lease is expired or held by this controller, take it
	if lease.Spec.RenewTime == nil || now.Sub(lease.Spec.RenewTime.Time) > DefaultLeaseTTL || *lease.Spec.HolderIdentity == holderIdentity {
		if lease.Spec.HolderIdentity == nil || *lease.Spec.HolderIdentity != holderIdentity {
			lease.Spec.LeaseTransitions = pointer(*lease.Spec.LeaseTransitions + 1)
		}
		lease.Spec.HolderIdentity = &holderIdentity
		lease.Spec.RenewTime = &metav1.MicroTime{Time: now}
		lease.Spec.AcquireTime = &metav1.MicroTime{Time: now}
		if err := c.Update(ctx, lease); err != nil {
			return false, fmt.Errorf("failed to update lease: %w", err)
		}
		return true, nil
	}
	return false, nil // Lease is held by another controller

}

// ReleaseResourceLease releases the lease for a given resource if it is held by the specified controller.
// It sets the HolderIdentity, RenewTime, and AcquireTime to nil to indicate that the lease is no longer held.
func ReleaseResourceLease(
	ctx context.Context, c client.Client, resourceId string, namespace string,
	controllerID string, resourceType string, owner metav1.Object,
) error {
	leaseName := getLeaseName(resourceType, owner.GetName())
	holderIdentity := getHolderIdentity(controllerID, resourceId)
	lease, err := getLease(ctx, c, leaseName, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get lease: %w", err)
	}

	if lease.Spec.HolderIdentity != nil && *lease.Spec.HolderIdentity == holderIdentity {
		lease.Spec.HolderIdentity = nil
		lease.Spec.RenewTime = nil
		lease.Spec.AcquireTime = nil
		if err := c.Update(ctx, lease); err != nil {
			return fmt.Errorf("failed to release lease: %w", err)
		}
	}
	return nil
}

func getLease(ctx context.Context, c client.Client, leaseName string, namespace string) (*coordinationv1.Lease, error) {
	lease := &coordinationv1.Lease{}
	err := c.Get(ctx, types.NamespacedName{Name: leaseName, Namespace: namespace}, lease)
	return lease, err
}

func getLeaseName(resourceType string, ownerName string) string {
	return fmt.Sprintf("%s-%s-lock", strings.ToLower(resourceType), strings.ToLower(ownerName))
}

func getHolderIdentity(controllerID string, resourceId string) string {
	return fmt.Sprintf("%s-%s", controllerID, resourceId)
}

func pointer(i int32) *int32 {
	return &i
}
