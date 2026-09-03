package vectordeployment

import (
	"context"
	"fmt"
	"sort"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type State string

const (
	StateDeployingVector  State = "DeployingVector"
	StateDeploymentReady  State = "DeploymentReady"
	StateDeploymentFailed State = "DeploymentFailed"
)

func StateFromConditions(conditions []metav1.Condition) State {
	if meta.IsStatusConditionTrue(conditions, konfidence.VectorReadyCondition) {
		return StateDeploymentReady
	}

	// TODO: mapping to StateDeploymentFailed https://github.com/konfidence-project/konfidence/issues/167

	return StateDeployingVector
}

type ResolvedVectorDeployment struct {
	VectorDeployment konfidence.VectorDeployment
	LandscapeId      string
	StageId          string
}

type Repository interface {
	// ListForScope lists vector deployments with their landscape and stage IDs across
	// the given landscape scope. It skips entries whose namespace is still empty.
	ListForScope(ctx context.Context, scope []landscapedomain.ScopedLandscape) ([]ResolvedVectorDeployment, error)
}

type k8sRepository struct{ reader client.Reader }

func NewRepository(reader client.Reader) Repository {
	return &k8sRepository{reader: reader}
}

func (r *k8sRepository) ListForScope(
	ctx context.Context,
	scope []landscapedomain.ScopedLandscape,
) ([]ResolvedVectorDeployment, error) {
	resolved := make([]ResolvedVectorDeployment, 0, len(scope))
	for _, scoped := range scope {
		if scoped.Namespace == "" {
			continue
		}

		var deployments konfidence.VectorDeploymentList
		if err := r.reader.List(ctx, &deployments, client.InNamespace(scoped.Namespace)); err != nil {
			return nil, fmt.Errorf("listing vector deployments in namespace %q: %w", scoped.Namespace, err)
		}

		var stageVersions konfidence.StageVersionList
		if err := r.reader.List(ctx, &stageVersions, client.InNamespace(scoped.Namespace)); err != nil {
			return nil, fmt.Errorf("listing stage versions in namespace %q: %w", scoped.Namespace, err)
		}
		stageVersionByName := make(map[string]*konfidence.StageVersion, len(stageVersions.Items))
		for i := range stageVersions.Items {
			stageVersion := &stageVersions.Items[i]
			stageVersionByName[stageVersion.Name] = stageVersion
		}

		for i := range deployments.Items {
			deployment := &deployments.Items[i]
			stageVersionName := stageVersionOwnerName(deployment)
			stageVersion := stageVersionByName[stageVersionName]
			if stageVersionName == "" || stageVersion == nil || stageVersion.Spec.StageRef == nil || stageVersion.Spec.StageRef.Name == "" {
				return nil, fmt.Errorf("resolving stage for vector deployment %q in namespace %q", deployment.Name, deployment.Namespace)
			}

			resolved = append(resolved, ResolvedVectorDeployment{
				VectorDeployment: *deployment,
				LandscapeId:      scoped.Landscape.Name,
				StageId:          stageVersion.Spec.StageRef.Name,
			})
		}
	}

	sort.Slice(resolved, func(i, j int) bool {
		if resolved[i].LandscapeId != resolved[j].LandscapeId {
			return resolved[i].LandscapeId < resolved[j].LandscapeId
		}
		return resolved[i].VectorDeployment.Name < resolved[j].VectorDeployment.Name
	})

	return resolved, nil
}

func stageVersionOwnerName(deployment *konfidence.VectorDeployment) string {
	for _, owner := range deployment.OwnerReferences {
		if owner.Kind == konfidence.StageVersionKind {
			return owner.Name
		}
	}
	return ""
}
