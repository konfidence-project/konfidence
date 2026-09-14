package artifactdeployment

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	utils "github.com/konfidence-project/konfidence/pkg/controller"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrNotFound = fmt.Errorf("artifact deployment not found")

type ResolvedArtifactDeployment struct {
	ArtifactDeployment  konfidence.ArtifactDeployment
	LandscapeId         string
	StageIds            []string
	VectorDeploymentIds []string
}

type Repository interface {
	Get(ctx context.Context, name string) (*konfidence.ArtifactDeployment, error)
	ListForScope(ctx context.Context, namespace string, opts ...ListOption) ([]ResolvedArtifactDeployment, error)
}

type ListOption func(client.MatchingLabels)

func WithLandscapeId(id string) ListOption {
	return func(labels client.MatchingLabels) {
		labels[utils.LandscapeNameLabel] = id
	}
}

func WithVectorDeploymentId(id string) ListOption {
	return func(labels client.MatchingLabels) {
		labels[utils.VectorDeploymentNameLabel] = id
	}
}

type k8sRepository struct{ reader client.Reader }

func NewRepository(reader client.Reader) Repository {
	return &k8sRepository{reader: reader}
}

func (r *k8sRepository) Get(ctx context.Context, name string) (*konfidence.ArtifactDeployment, error) {
	var artifactDeployment konfidence.ArtifactDeployment
	if err := r.reader.Get(ctx, types.NamespacedName{Name: name}, &artifactDeployment); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting artifact deployment %q: %w", name, err)
	}

	return &artifactDeployment, nil
}

func (r *k8sRepository) ListForScope(ctx context.Context, namespace string, opts ...ListOption) ([]ResolvedArtifactDeployment, error) {
	resolved := make([]ResolvedArtifactDeployment, 0)
	labelSelector := client.MatchingLabels{}

	for _, opt := range opts {
		opt(labelSelector)
	}

	var landscapeId *string
	if landscapeNameLabel, ok := labelSelector[utils.LandscapeNameLabel]; ok {
		landscapeId = &landscapeNameLabel
	}

	var landscapes konfidence.LandscapeList
	if err := r.reader.List(ctx, &landscapes, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("listing landscapes in namespace %q: %w", namespace, err)
	}

	var scopedLandscapes []landscapedomain.ScopedLandscape
	for _, l := range landscapes.Items {
		if landscapeId != nil && l.Name != *landscapeId {
			continue
		}
		if l.Status.Namespace != "" {
			scopedLandscapes = append(scopedLandscapes, landscapedomain.ScopedLandscape{
				Landscape: l,
				Namespace: l.Status.Namespace,
			})
		}
	}

	if landscapeId != nil && len(scopedLandscapes) == 0 {
		return nil, fmt.Errorf("landscape %q not found", *landscapeId)
	}

	adLabelSelector := client.MatchingLabels{}
	if vdId, ok := labelSelector[utils.VectorDeploymentNameLabel]; ok {
		adLabelSelector[utils.VectorDeploymentNameLabel] = vdId
	}

	for _, scoped := range scopedLandscapes {
		var artifactDeployments konfidence.ArtifactDeploymentList
		if err := r.reader.List(ctx, &artifactDeployments, client.InNamespace(scoped.Namespace), adLabelSelector); err != nil {
			return nil, fmt.Errorf("listing artifact deployments in namespace %q: %w", scoped.Namespace, err)
		}

		var stageVersions konfidence.StageVersionList
		if err := r.reader.List(ctx, &stageVersions, client.InNamespace(scoped.Namespace)); err != nil {
			return nil, fmt.Errorf("listing stage versions in namespace %q: %w", scoped.Namespace, err)
		}

		stageVersionByName := make(map[string]*konfidence.StageVersion, len(stageVersions.Items))
		for i := range stageVersions.Items {
			sv := &stageVersions.Items[i]
			stageVersionByName[sv.Name] = sv
		}

		for i := range artifactDeployments.Items {
			ad := &artifactDeployments.Items[i]
			stageIds := extractStageIds(ad, stageVersionByName)
			vectorDeploymentIds := extractVectorDeploymentIds(ad)

			resolved = append(resolved, ResolvedArtifactDeployment{
				ArtifactDeployment:  *ad,
				LandscapeId:         scoped.Landscape.Name,
				StageIds:            stageIds,
				VectorDeploymentIds: vectorDeploymentIds,
			})
		}
	}

	return resolved, nil
}

func extractVectorDeploymentIds(ad *konfidence.ArtifactDeployment) []string {
	var vectorDeploymentIds []string
	if vdName, ok := ad.Labels[utils.VectorDeploymentNameLabel]; ok {
		vectorDeploymentIds = append(vectorDeploymentIds, vdName)
	}
	return vectorDeploymentIds
}

func extractStageIds(ad *konfidence.ArtifactDeployment, stageVersionByName map[string]*konfidence.StageVersion) []string {
	var stageIds []string
	for _, ownerRef := range ad.OwnerReferences {
		if ownerRef.Kind == konfidence.StageVersionKind {
			stageVersion := stageVersionByName[ownerRef.Name]
			if stageVersion != nil && stageVersion.Spec.StageRef != nil && stageVersion.Spec.StageRef.Name != "" {
				stageIds = append(stageIds, stageVersion.Spec.StageRef.Name)
			}
		}
	}
	return stageIds
}
