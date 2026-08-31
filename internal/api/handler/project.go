package handler

import (
	"context"
	"errors"
	"fmt"
	"sort"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	projectdomain "github.com/konfidence-project/konfidence/internal/project"
	vectorpromotiondomain "github.com/konfidence-project/konfidence/internal/vectorpromotion"
)

type projectHandler struct {
	projectRepo               projectdomain.Repository
	landscapeRepo             landscapedomain.Repository
	vectorPromotionRepo       vectorpromotiondomain.Repository
	vectorPromotionConfigRepo vectorpromotiondomain.ConfigRepository
}

func newProjectHandler(projectRepo projectdomain.Repository, landscapeRepo landscapedomain.Repository,
	vectorPromotionRepo vectorpromotiondomain.Repository, vectorPromotionConfigRepo vectorpromotiondomain.ConfigRepository) *projectHandler {
	return &projectHandler{
		projectRepo:               projectRepo,
		landscapeRepo:             landscapeRepo,
		vectorPromotionRepo:       vectorPromotionRepo,
		vectorPromotionConfigRepo: vectorPromotionConfigRepo,
	}
}

func (h *projectHandler) ListProjectsV1(ctx context.Context, _ openapi.ListProjectsV1RequestObject) (openapi.ListProjectsV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return openapi.ListProjectsV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
	}
	projects, err := h.projectRepo.List(ctx, identity.ProjectRoles)
	if err != nil {
		return openapi.ListProjectsV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	data := make([]openapi.Project, 0, len(projects))
	for _, p := range projects {
		data = append(data, toProjectResponse(p))
	}

	return openapi.ListProjectsV1200JSONResponse{Data: data}, nil
}

func (h *projectHandler) ListLandscapesV1(ctx context.Context, req openapi.ListLandscapesV1RequestObject) (openapi.ListLandscapesV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return openapi.ListLandscapesV1401JSONResponse{
			UnauthorizedJSONResponse: apierror.NewUnauthorizedResponse(),
		}, nil
	}

	project, err := h.projectRepo.Get(ctx, req.ProjectId, identity.ProjectRoles)
	if err != nil {
		if errors.Is(err, projectdomain.ErrNotFound) {
			return openapi.ListLandscapesV1404JSONResponse{
				NotFoundJSONResponse: apierror.NewNotFoundResponse(fmt.Sprintf("project %q not found", req.ProjectId)),
			}, nil
		}
		if errors.Is(err, projectdomain.ErrForbidden) {
			return openapi.ListLandscapesV1403JSONResponse{
				ForbiddenJSONResponse: apierror.NewForbiddenResponse(fmt.Sprintf("access to project %q is not allowed", req.ProjectId)),
			}, nil
		}
		return openapi.ListLandscapesV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	if project.Status.Namespace == "" {
		return openapi.ListLandscapesV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	landscapes, err := h.landscapeRepo.ListForProject(ctx, project.Status.Namespace)
	if err != nil {
		return openapi.ListLandscapesV1500JSONResponse{
			InternalErrorJSONResponse: apierror.NewInternalErrorResponse(),
		}, nil
	}

	data := make([]openapi.Landscape, len(landscapes))
	for i, l := range landscapes {
		data[i] = toLandscapeResponse(l)
	}

	return openapi.ListLandscapesV1200JSONResponse{Data: data}, nil
}

func toProjectResponse(p konfidence.Project) openapi.Project {
	return openapi.Project{
		Id:   p.Name,
		Name: p.Spec.DisplayName,
	}
}

func toLandscapeResponse(l konfidence.Landscape) openapi.Landscape {
	return openapi.Landscape{
		Id:   l.Name,
		Name: l.Spec.DisplayName,
	}
}

func (h *projectHandler) ListStagesV1(_ context.Context, _ openapi.ListStagesV1RequestObject) (openapi.ListStagesV1ResponseObject, error) {
	return nil, nil
}

func (h *projectHandler) ListVectorDeploymentsV1(_ context.Context,
	_ openapi.ListVectorDeploymentsV1RequestObject) (openapi.ListVectorDeploymentsV1ResponseObject, error) {
	return nil, nil
}

func (h *projectHandler) ListArtifactDeploymentsV1(_ context.Context,
	_ openapi.ListArtifactDeploymentsV1RequestObject) (openapi.ListArtifactDeploymentsV1ResponseObject, error) {
	return nil, nil
}

// resolveProjectContext handles the preamble common to handlers that operate on
// a project-scoped k8s namespace: it validates the session, checks authorization,
// fetches the project, and confirms the namespace is set. On failure it returns an
// *apierror.Error carrying the HTTP status and a client-safe message; the strict
// server's ResponseErrorHandler renders it, so callers can simply `return nil, err`.
func (h *projectHandler) resolveProjectContext(ctx context.Context, projectId string) (*session.Context, string, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return nil, "", apierror.NewUnauthorized()
	}
	if !identity.IsAuthenticatedForProject(projectId) {
		return nil, "", apierror.NewForbidden(fmt.Sprintf("access to project %q is not allowed", projectId))
	}
	project, err := h.projectRepo.Get(ctx, projectId, identity.ProjectRoles)
	if err != nil {
		if errors.Is(err, projectdomain.ErrNotFound) {
			return nil, "", apierror.NewNotFound("project", projectId)
		}
		if errors.Is(err, projectdomain.ErrForbidden) {
			return nil, "", apierror.NewForbidden(fmt.Sprintf("access to project %q is not allowed", projectId))
		}
		return nil, "", apierror.NewInternal(err)
	}
	if project.Status.Namespace == "" {
		return nil, "", apierror.NewInternal(fmt.Errorf("project %q has no namespace configured", projectId))
	}
	return identity, project.Status.Namespace, nil
}

func (h *projectHandler) GetVectorPromotionConfigV1(ctx context.Context,
	request openapi.GetVectorPromotionConfigV1RequestObject) (openapi.GetVectorPromotionConfigV1ResponseObject, error) {
	_, namespace, err := h.resolveProjectContext(ctx, request.ProjectId)
	if err != nil {
		return nil, err
	}

	config, err := h.vectorPromotionConfigRepo.Get(ctx, namespace, request.VectorPromotionConfigId)
	if err != nil {
		if errors.Is(err, vectorpromotiondomain.ErrVectorPromotionConfigNotFound) {
			return nil, apierror.NewNotFound("vectorPromotionConfig", request.VectorPromotionConfigId)
		}
		return nil, apierror.NewInternal(err)
	}

	promotions, err := h.vectorPromotionRepo.ListForConfig(ctx, namespace, config.Name)
	if err != nil {
		return nil, apierror.NewInternal(err)
	}

	response := toVectorPromotionConfigResponse(*config, promotions)
	return openapi.GetVectorPromotionConfigV1200JSONResponse(response), nil
}

func (h *projectHandler) ListVectorPromotionConfigsV1(ctx context.Context,
	request openapi.ListVectorPromotionConfigsV1RequestObject) (openapi.ListVectorPromotionConfigsV1ResponseObject, error) {
	_, namespace, err := h.resolveProjectContext(ctx, request.ProjectId)
	if err != nil {
		return nil, err
	}

	configs, err := h.vectorPromotionConfigRepo.List(ctx, namespace)
	if err != nil {
		return nil, apierror.NewInternal(err)
	}

	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Name < configs[j].Name
	})

	data := make([]openapi.VectorPromotionConfig, 0, len(configs))
	for _, c := range configs {
		promotions, err := h.vectorPromotionRepo.ListForConfig(ctx, namespace, c.Name)
		if err != nil {
			return nil, apierror.NewInternal(err)
		}
		data = append(data, toVectorPromotionConfigResponse(c, promotions))
	}

	return openapi.ListVectorPromotionConfigsV1200JSONResponse{Data: data}, nil
}

func toVectorPromotionConfigResponse(c konfidence.VectorPromotionConfig,
	promotions []konfidence.VectorPromotion) openapi.VectorPromotionConfig {
	config := openapi.VectorPromotionConfig{
		Id:     c.Name,
		Source: toPromotionSourceReferenceResponse(c.Spec.Source),
		Target: toPromotionTargetReferenceResponse(c.Spec.Target),
	}

	if c.Spec.TTLAfterFinished != nil {
		ttl := c.Spec.TTLAfterFinished.Duration.String()
		config.TtlAfterFinished = &ttl
	}

	if c.Spec.KeepLastPromotions != nil {
		keepLast := int(*c.Spec.KeepLastPromotions)
		config.KeepLastPromotions = &keepLast
	}

	config.Promotions = make([]openapi.VectorPromotion, len(promotions))
	for i, promotion := range promotions {
		config.Promotions[i] = toVectorPromotionResponse(promotion)
	}

	return config
}

func (h *projectHandler) GetVectorPromotionV1(ctx context.Context,
	request openapi.GetVectorPromotionV1RequestObject) (openapi.GetVectorPromotionV1ResponseObject, error) {
	_, namespace, err := h.resolveProjectContext(ctx, request.ProjectId)
	if err != nil {
		return nil, err
	}

	promotion, err := h.vectorPromotionRepo.Get(ctx, namespace, request.VectorPromotionId)
	if err != nil {
		if errors.Is(err, vectorpromotiondomain.ErrVectorPromotionNotFound) {
			return nil, apierror.NewNotFound("vectorPromotion", request.VectorPromotionId)
		}
		return nil, apierror.NewInternal(err)
	}

	response := toVectorPromotionResponse(*promotion)
	return openapi.GetVectorPromotionV1200JSONResponse(response), nil
}

func (h *projectHandler) ApproveVectorPromotionV1(ctx context.Context,
	request openapi.ApproveVectorPromotionV1RequestObject) (openapi.ApproveVectorPromotionV1ResponseObject, error) {
	identity, namespace, err := h.resolveProjectContext(ctx, request.ProjectId)
	if err != nil {
		return nil, err
	}

	err = h.vectorPromotionRepo.Approve(ctx, namespace, request.VectorPromotionId, identity.Subject)
	switch {
	case err == nil, errors.Is(err, vectorpromotiondomain.ErrAlreadyApproved):
		// Approving twice is idempotent: the promotion is already approved,
		// so report success rather than a conflict.
		return openapi.ApproveVectorPromotionV1204Response{}, nil
	case errors.Is(err, vectorpromotiondomain.ErrVectorPromotionNotFound):
		return nil, apierror.NewNotFound("vectorPromotion", request.VectorPromotionId)
	case errors.Is(err, vectorpromotiondomain.ErrPromotionSuperseded),
		errors.Is(err, vectorpromotiondomain.ErrPromotionFinished),
		errors.Is(err, vectorpromotiondomain.ErrApprovalNotRequired):
		// These domain error strings are the canonical client-facing conflict message.
		return nil, apierror.NewConflict(err.Error())
	default:
		// ErrApproverMissing intentionally maps to 500: an empty subject indicates
		// server misconfiguration, not a client error.
		return nil, apierror.NewInternal(err)
	}
}

func toVectorPromotionResponse(p konfidence.VectorPromotion) openapi.VectorPromotion {
	requireApproval := p.Spec.RequireApproval
	sequence := p.Spec.Sequence

	promotion := openapi.VectorPromotion{
		Id:              p.Name,
		RequireApproval: &requireApproval,
		Sequence:        &sequence,
		Source:          toPromotionSourceReferenceResponse(p.Spec.Source),
		Target:          toPromotionTargetReferenceResponse(p.Spec.Target),
		Vector:          p.Spec.Vector,
	}

	if p.Status.State != "" {
		status := openapi.VectorPromotionStatus(p.Status.State)
		promotion.Status = &status
	}

	if p.Spec.TTLAfterFinished != nil {
		ttl := p.Spec.TTLAfterFinished.Duration.String()
		promotion.TtlAfterFinished = &ttl
	}

	if p.Status.Approval != nil {
		promotion.Approval = &openapi.PromotionApproval{
			ApprovedBy: p.Status.Approval.ApprovedBy,
			ApprovedAt: p.Status.Approval.ApprovedAt.Time,
		}
	}

	return promotion
}

func toPromotionSourceReferenceResponse(s konfidence.PromotionSourceReference) openapi.PromotionSourceReference {
	source := openapi.PromotionSourceReference{
		Kind: openapi.PromotionSourceReferenceKind(s.Kind),
		Name: s.Name,
	}
	if s.Landscape != "" {
		landscape := s.Landscape
		source.Landscape = &landscape
	}
	return source
}

func toPromotionTargetReferenceResponse(t konfidence.PromotionTargetReference) openapi.PromotionTargetReference {
	return openapi.PromotionTargetReference{
		Kind:      openapi.PromotionTargetReferenceKind(t.Kind),
		Name:      t.Name,
		Landscape: t.Landscape,
	}
}
