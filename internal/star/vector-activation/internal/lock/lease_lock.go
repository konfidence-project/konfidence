package lock

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"

	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DefaultLeaseTTL = 30 * time.Second

func AcquireResourceLease(ctx context.Context, client kubernetes.Interface, resourceId string, namespace string, controllerID string, resourceType string, stageName string, ownerRef metav1.OwnerReference) (bool, error) {
	leaseName := getLeaseName(resourceType, stageName)
	lease, err := client.CoordinationV1().Leases(namespace).Get(ctx, leaseName, metav1.GetOptions{})
	holderIdentity := getHolderIdentity(controllerID, resourceId)
	now := time.Now()

	if err != nil {
		if apierrors.IsNotFound(err) {
			// If not found, create it
			lease = &coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{
					Name:            leaseName,
					Namespace:       namespace,
					OwnerReferences: []metav1.OwnerReference{ownerRef},
				},
				Spec: coordinationv1.LeaseSpec{
					HolderIdentity:       &holderIdentity,
					LeaseDurationSeconds: pointer(int32(DefaultLeaseTTL.Seconds())),
					AcquireTime:          &metav1.MicroTime{Time: now},
					RenewTime:            &metav1.MicroTime{Time: now},
				},
			}
			_, err := client.CoordinationV1().Leases(namespace).Create(ctx, lease, metav1.CreateOptions{})
			if err != nil {
				return false, fmt.Errorf("failed to create lease: %w", err)
			}
			return true, nil
		}
		return false, err
	}

	// If lease is expired or held by this controller, take it
	if lease.Spec.RenewTime == nil || now.Sub(lease.Spec.RenewTime.Time) > DefaultLeaseTTL || *lease.Spec.HolderIdentity == holderIdentity {
		lease.Spec.HolderIdentity = &holderIdentity
		lease.Spec.RenewTime = &metav1.MicroTime{Time: now}
		lease.Spec.AcquireTime = &metav1.MicroTime{Time: now}
		_, err := client.CoordinationV1().Leases(namespace).Update(ctx, lease, metav1.UpdateOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to update lease: %w", err)
		}
		return true, nil
	}
	return false, nil // Lease is held by another controller

}

func ReleaseResourceLease(ctx context.Context, client kubernetes.Interface, resourceName string, namespace string, controllerID string, resourceType string, stageName string) error {
	leaseName := getLeaseName(resourceType, stageName)
	holderIdentity := getHolderIdentity(controllerID, resourceName)
	lease, err := client.CoordinationV1().Leases(namespace).Get(ctx, leaseName, metav1.GetOptions{})
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
		_, err := client.CoordinationV1().Leases(namespace).Update(ctx, lease, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to release lease: %w", err)
		}
	}
	return nil
}

func getLeaseName(resourceType string, stageName string) string {
	return fmt.Sprintf("%s-%s-lock", strings.ToLower(resourceType), strings.ToLower(stageName))
}

func getHolderIdentity(controllerID string, resourceId string) string {
	return fmt.Sprintf("%s-%s", controllerID, resourceId)
}

func pointer(i int32) *int32 {
	return &i
}
