package stage

import (
	"context"
	"fmt"
	"sort"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResolvedStage is a stage together with the landscape it lives in and its resolved stage versions.
type ResolvedStage struct {
	Stage       konfidence.Stage
	LandscapeID string
	Target      *konfidence.StageVersion // nil if not yet created
	Active      *konfidence.StageVersion // nil if none active / dangling ref
}

type Repository interface {
	// ListForScope lists stages with resolved versions across the given
	// landscape scope (from landscapedomain.ResolveScope). Skips scope
	// entries whose Namespace is still "".
	ListForScope(ctx context.Context, scope []landscapedomain.ScopedLandscape) ([]ResolvedStage, error)
}

type k8sRepository struct{ reader client.Reader }

func NewRepository(reader client.Reader) Repository {
	return &k8sRepository{reader: reader}
}

func (r *k8sRepository) ListForScope(
	ctx context.Context,
	scope []landscapedomain.ScopedLandscape,
) ([]ResolvedStage, error) {
	resolved := make([]ResolvedStage, 0, len(scope))
	for _, scoped := range scope {
		if scoped.Namespace == "" {
			continue
		}

		var stages konfidence.StageList
		if err := r.reader.List(ctx, &stages, client.InNamespace(scoped.Namespace)); err != nil {
			return nil, fmt.Errorf("listing stages in namespace %q: %w", scoped.Namespace, err)
		}

		var versions konfidence.StageVersionList
		if err := r.reader.List(ctx, &versions, client.InNamespace(scoped.Namespace)); err != nil {
			return nil, fmt.Errorf("listing stage versions in namespace %q: %w", scoped.Namespace, err)
		}
		versionsByStage := groupVersionsByStage(versions.Items)

		for _, s := range stages.Items {
			resolved = append(resolved, resolveStage(s, scoped.Landscape.Name, versionsByStage[s.Name]))
		}
	}

	sort.Slice(resolved, func(i, j int) bool {
		if resolved[i].LandscapeID != resolved[j].LandscapeID {
			return resolved[i].LandscapeID < resolved[j].LandscapeID
		}
		return resolved[i].Stage.Name < resolved[j].Stage.Name
	})

	return resolved, nil
}

// groupVersionsByStage groups stage versions by the name of the stage they belong to.
// Versions without a stage reference cannot be attributed to a stage and are ignored.
func groupVersionsByStage(versions []konfidence.StageVersion) map[string][]*konfidence.StageVersion {
	grouped := make(map[string][]*konfidence.StageVersion, len(versions))
	for i := range versions {
		version := &versions[i]
		if version.Spec.StageRef == nil {
			continue
		}
		grouped[version.Spec.StageRef.Name] = append(grouped[version.Spec.StageRef.Name], version)
	}
	return grouped
}

// resolveStage picks the target and active stage version out of the stage's own versions.
// The target is the version created for the stage's current spec, matched by generation and
// vector instead of by name, so the version's name hash does not have to be re-derived.
func resolveStage(s konfidence.Stage, landscapeID string, versions []*konfidence.StageVersion) ResolvedStage {
	resolved := ResolvedStage{Stage: s, LandscapeID: landscapeID}

	activeName := ""
	if s.Status.ActiveStageVersion != nil {
		activeName = s.Status.ActiveStageVersion.Name
	}

	for _, version := range versions {
		if version.Spec.StageGeneration == s.Generation && version.Spec.Vector == s.Spec.Vector {
			resolved.Target = version
		}
		if activeName != "" && version.Name == activeName {
			resolved.Active = version
		}
	}

	return resolved
}
