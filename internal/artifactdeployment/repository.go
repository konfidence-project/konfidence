package artifactdeployment

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrArtifactDeploymentNotFound = errors.New("artifact deployment not found")

type ResolvedArtifactDeployment struct {
	ArtifactDeployment  konfidence.ArtifactDeployment
	LandscapeId         string
	StageIds            []string
	VectorDeploymentIds []string
}

type Repository interface {
	GetForScope(ctx context.Context, scoped landscapedomain.ScopedLandscape, name string) (*ResolvedArtifactDeployment, error)
	ListForScope(ctx context.Context, scope []landscapedomain.ScopedLandscape, opts ...ListOption) ([]ResolvedArtifactDeployment, error)
}

type listOptions struct {
	vectorDeploymentID *string
}

type ListOption func(*listOptions)

func WithVectorDeploymentId(id string) ListOption {
	return func(options *listOptions) {
		options.vectorDeploymentID = &id
	}
}

type k8sRepository struct{ reader client.Reader }

func NewRepository(reader client.Reader) Repository {
	return &k8sRepository{reader: reader}
}

func (r *k8sRepository) GetForScope(ctx context.Context, scoped landscapedomain.ScopedLandscape, name string) (*ResolvedArtifactDeployment, error) {
	var ad konfidence.ArtifactDeployment
	if err := r.reader.Get(ctx, types.NamespacedName{
		Namespace: scoped.Namespace,
		Name:      name,
	}, &ad); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, ErrArtifactDeploymentNotFound
		}
		return nil, fmt.Errorf("getting artifact deployment %q in namespace %q: %w", name, scoped.Namespace, err)
	}

	var stageVersions konfidence.StageVersionList
	if err := r.reader.List(ctx, &stageVersions, client.InNamespace(scoped.Namespace)); err != nil {
		return nil, fmt.Errorf("listing stage versions in namespace %q: %w", scoped.Namespace, err)
	}

	var vectorDeployments konfidence.VectorDeploymentList
	if err := r.reader.List(ctx, &vectorDeployments, client.InNamespace(scoped.Namespace)); err != nil {
		return nil, fmt.Errorf("listing vector deployments in namespace %q: %w", scoped.Namespace, err)
	}

	stageVersionByName := make(map[string]*konfidence.StageVersion, len(stageVersions.Items))
	for i := range stageVersions.Items {
		sv := &stageVersions.Items[i]
		stageVersionByName[sv.Name] = sv
	}

	vectorDeploymentByName := make(map[string]*konfidence.VectorDeployment, len(vectorDeployments.Items))
	for i := range vectorDeployments.Items {
		vd := &vectorDeployments.Items[i]
		vectorDeploymentByName[vd.Name] = vd
	}

	return &ResolvedArtifactDeployment{
		ArtifactDeployment:  ad,
		LandscapeId:         scoped.Landscape.Name,
		StageIds:            extractStageIds(&ad, vectorDeploymentByName, stageVersionByName),
		VectorDeploymentIds: extractVectorDeploymentIds(&ad),
	}, nil
}

func (r *k8sRepository) ListForScope(ctx context.Context, scope []landscapedomain.ScopedLandscape, opts ...ListOption) ([]ResolvedArtifactDeployment, error) {
	options := listOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	resolved := make([]ResolvedArtifactDeployment, 0)
	for _, scoped := range scope {
		if scoped.Namespace == "" {
			continue
		}

		var artifactDeployments konfidence.ArtifactDeploymentList
		if err := r.reader.List(ctx, &artifactDeployments, client.InNamespace(scoped.Namespace)); err != nil {
			return nil, fmt.Errorf("listing artifact deployments in namespace %q: %w", scoped.Namespace, err)
		}

		var stageVersions konfidence.StageVersionList
		if err := r.reader.List(ctx, &stageVersions, client.InNamespace(scoped.Namespace)); err != nil {
			return nil, fmt.Errorf("listing stage versions in namespace %q: %w", scoped.Namespace, err)
		}

		var vectorDeployments konfidence.VectorDeploymentList
		if err := r.reader.List(ctx, &vectorDeployments, client.InNamespace(scoped.Namespace)); err != nil {
			return nil, fmt.Errorf("listing vector deployments in namespace %q: %w", scoped.Namespace, err)
		}

		stageVersionByName := make(map[string]*konfidence.StageVersion, len(stageVersions.Items))
		for i := range stageVersions.Items {
			sv := &stageVersions.Items[i]
			stageVersionByName[sv.Name] = sv
		}

		vectorDeploymentByName := make(map[string]*konfidence.VectorDeployment, len(vectorDeployments.Items))
		for i := range vectorDeployments.Items {
			vd := &vectorDeployments.Items[i]
			vectorDeploymentByName[vd.Name] = vd
		}

		for i := range artifactDeployments.Items {
			artifactDeployment := &artifactDeployments.Items[i]
			vectorDeploymentIDs := extractVectorDeploymentIds(artifactDeployment)
			if options.vectorDeploymentID != nil && !slices.Contains(vectorDeploymentIDs, *options.vectorDeploymentID) {
				continue
			}

			stageIDs := extractStageIds(artifactDeployment, vectorDeploymentByName, stageVersionByName)
			resolved = append(resolved, ResolvedArtifactDeployment{
				ArtifactDeployment:  *artifactDeployment,
				LandscapeId:         scoped.Landscape.Name,
				StageIds:            stageIDs,
				VectorDeploymentIds: vectorDeploymentIDs,
			})
		}
	}

	sort.Slice(resolved, func(i, j int) bool {
		if resolved[i].LandscapeId != resolved[j].LandscapeId {
			return resolved[i].LandscapeId < resolved[j].LandscapeId
		}
		return resolved[i].ArtifactDeployment.Name < resolved[j].ArtifactDeployment.Name
	})

	return resolved, nil
}

// extractVectorDeploymentIds returns every owning VectorDeployment. An
// ArtifactDeployment reused across stages has one owner reference per
// VectorDeployment, so owner refs (not the single-valued label) are the source.
func extractVectorDeploymentIds(ad *konfidence.ArtifactDeployment) []string {
	var vectorDeploymentIds []string
	for _, ownerRef := range ad.OwnerReferences {
		if ownerRef.Kind == konfidence.VectorDeploymentKind {
			vectorDeploymentIds = append(vectorDeploymentIds, ownerRef.Name)
		}
	}
	return vectorDeploymentIds
}

func extractStageIds(
	ad *konfidence.ArtifactDeployment,
	vectorDeploymentByName map[string]*konfidence.VectorDeployment,
	stageVersionByName map[string]*konfidence.StageVersion,
) []string {
	var stageIds []string
	for _, artifactOwnerRef := range ad.OwnerReferences {
		if artifactOwnerRef.Kind != konfidence.VectorDeploymentKind {
			continue
		}
		vectorDeployment := vectorDeploymentByName[artifactOwnerRef.Name]
		if vectorDeployment == nil {
			continue
		}
		for _, vectorOwnerRef := range vectorDeployment.OwnerReferences {
			if vectorOwnerRef.Kind != konfidence.StageVersionKind {
				continue
			}
			stageVersion := stageVersionByName[vectorOwnerRef.Name]
			if stageVersion != nil && stageVersion.Spec.StageRef != nil && stageVersion.Spec.StageRef.Name != "" {
				stageIds = append(stageIds, stageVersion.Spec.StageRef.Name)
			}
		}
	}
	return stageIds
}
