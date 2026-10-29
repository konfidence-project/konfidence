package util

import (
	"context"
	"fmt"
	"io"

	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

// KonfidenceResourceTypeVectorConfig is the OCM resource name used during vector assembly
// to mark the optional, singleton vector-scoped configuration resource on a vector
// ComponentVersion. Note that this value is matched against `Resource.Name`, not `Resource.Type` -- the Type field
// is left to the vector author. The character set is constrained to lowercase letters, digits and dashes because
// dots are not permitted in OCM resource names per the schema. The constant is exported so that the assembly and
// deployment sides can reuse the exact same identifier.
const KonfidenceResourceNameVectorConfig = "cloud-konfidence-vector-config"

// GetLocalResourceBlob retrieves the raw bytes of a locally-stored OCM resource blob.
func GetLocalResourceBlob(ctx context.Context, client pkgocm.Client, ref compref.Ref, res *descruntime.Resource) ([]byte, error) {
	b, _, err := client.GetLocalResource(ctx, ref, res.ToIdentity())
	if err != nil {
		return nil, fmt.Errorf("getting local resource for %s identity %v: %w", ref, res.ToIdentity(), err)
	}
	rc, err := b.ReadCloser()
	if err != nil {
		return nil, fmt.Errorf("opening blob reader: %w", err)
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("reading blob content: %w", err)
	}
	return data, nil
}

// ExtractVectorConfigResource scans the given resource slice for the optional, singleton vector-scoped configuration
// resource. The match is performed on the resource Name (KonfidenceResourceTypeVectorConfig), mirroring the producer
// side during vector assembly which sets that field to the same constant. When present, the resource's blob is
// fetched and the resource is removed from the slice that is returned to the caller. The function returns the blob
// bytes (or nil), the pruned resource slice, and any error encountered. Multiple matches are an authoring mistake and
// produce an error rather than a silently-chosen winner.
func ExtractVectorConfigResource(
	ctx context.Context,
	client pkgocm.Client,
	ref compref.Ref,
	resources []descruntime.Resource,
) ([]byte, []descruntime.Resource, error) {
	var (
		blob   []byte
		picked *descruntime.Resource
		pruned = make([]descruntime.Resource, 0, len(resources))
	)
	for i := range resources {
		resource := resources[i]
		if resource.Name != KonfidenceResourceNameVectorConfig {
			pruned = append(pruned, resource)
			continue
		}
		if picked != nil {
			return nil, nil, fmt.Errorf(
				"vector component %s declares more than one resource named %q (versions %q and %q); at most one is permitted",
				ref.String(), KonfidenceResourceNameVectorConfig, picked.Version, resource.Version,
			)
		}
		picked = &resources[i]
	}
	if picked != nil {
		data, err := GetLocalResourceBlob(ctx, client, ref, picked)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"failed to read vector config blob for resource %s in vector %s: %w",
				picked.Name, ref.String(), err,
			)
		}
		blob = data
	}
	return blob, pruned, nil
}
