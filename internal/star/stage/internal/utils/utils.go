package utils

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/big"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func VerifyOwnerReference(ctx context.Context, writer client.Writer, owner client.Object, child client.Object, scheme *runtime.Scheme, controlled bool) error {
	hasRef, err := controllerutil.HasOwnerReference(child.GetOwnerReferences(), owner, scheme)
	if err != nil {
		return fmt.Errorf("unable to check owner reference: %w", err)
	}

	if !hasRef {
		if controlled {
			if err := controllerutil.SetControllerReference(owner, child, scheme); err != nil {
				return fmt.Errorf("unable to set controller reference: %w", err)
			}
		} else {
			if err := controllerutil.SetOwnerReference(owner, child, scheme); err != nil {
				return fmt.Errorf("unable to set owner reference: %w", err)
			}
		}

		if err := writer.Update(ctx, child); err != nil {
			return fmt.Errorf("failed to update owner references: %w", err)
		}
	}

	return nil
}

func SetOwnerReference(owner client.Object, child client.Object, scheme *runtime.Scheme, controlled bool) error {
	hasRef, err := controllerutil.HasOwnerReference(child.GetOwnerReferences(), owner, scheme)
	if err != nil {
		return fmt.Errorf("unable to check owner reference: %w", err)
	}

	if !hasRef {
		if controlled {
			if err := controllerutil.SetControllerReference(owner, child, scheme); err != nil {
				return fmt.Errorf("unable to set controller reference: %w", err)
			}
		} else {
			if err := controllerutil.SetOwnerReference(owner, child, scheme); err != nil {
				return fmt.Errorf("unable to set owner reference: %w", err)
			}
		}
	}

	return nil
}

func ComputeFnv64Digest(content string) (string, error) {
	if len(content) == 0 {
		return "", fmt.Errorf("content is empty")
	}

	digest := fnv.New64a()
	_, err := digest.Write([]byte(content))
	if err != nil {
		return "", fmt.Errorf("unable to compute digest: %w", err)
	}

	// encode to base36
	return strconv.FormatUint(digest.Sum64(), 36), nil
}

func ComputeFnv128Digest(content string) (string, error) {
	if len(content) == 0 {
		return "", fmt.Errorf("content is empty")
	}

	digest := fnv.New128a()
	_, err := digest.Write([]byte(content))
	if err != nil {
		return "", fmt.Errorf("unable to compute digest: %w", err)
	}

	hashBytes := digest.Sum(nil)
	n := new(big.Int).SetBytes(hashBytes)

	// encode to base36
	return n.Text(36), nil
}
