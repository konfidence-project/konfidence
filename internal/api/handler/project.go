package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/apierror"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	artifactdeploymentdomain "github.com/konfidence-project/konfidence/internal/artifactdeployment"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	projectdomain "github.com/konfidence-project/konfidence/internal/project"
	stagedomain "github.com/konfidence-project/konfidence/internal/stage"
	vectordeploymentdomain "github.com/konfidence-project/konfidence/internal/vectordeployment"
	vectorpromotiondomain "github.com/konfidence-project/konfidence/internal/vectorpromotion"
	pkgcontent "github.com/konfidence-project/konfidence/pkg/ocm/content"
	pkgcredentials "github.com/konfidence-project/konfidence/pkg/ocm/credentials"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type projectHandler struct {
	projectRepo               projectdomain.Repository
	landscapeRepo             landscapedomain.Repository
	stageRepo                 stagedomain.Repository
	vectorDeploymentRepo      vectordeploymentdomain.Repository
	artifactDeploymentRepo    artifactdeploymentdomain.Repository
	vectorPromotionRepo       vectorpromotiondomain.Repository
	vectorPromotionConfigRepo vectorpromotiondomain.ConfigRepository
	k8sClient                 client.Reader
	log                       logr.Logger
	verifier                  crypto.Verifier
	// ocmReaderOverride bypasses per-request credential resolution when set.
	// Used in tests to inject a mock reader.
	ocmReaderOverride pkgcontent.Reader
}

func newProjectHandler(projectRepo projectdomain.Repository,
	landscapeRepo landscapedomain.Repository, stageRepo stagedomain.Repository,
	artifactDeploymentRepo artifactdeploymentdomain.Repository, vectorDeploymentRepo vectordeploymentdomain.Repository,
	vectorPromotionRepo vectorpromotiondomain.Repository,
	vectorPromotionConfigRepo vectorpromotiondomain.ConfigRepository,
	k8sClient client.Reader,
	log logr.Logger,
	verifier crypto.Verifier,
) *projectHandler {
	return &projectHandler{
		projectRepo:               projectRepo,
		landscapeRepo:             landscapeRepo,
		stageRepo:                 stageRepo,
		vectorDeploymentRepo:      vectorDeploymentRepo,
		artifactDeploymentRepo:    artifactDeploymentRepo,
		vectorPromotionRepo:       vectorPromotionRepo,
		vectorPromotionConfigRepo: vectorPromotionConfigRepo,
		k8sClient:                 k8sClient,
		log:                       log,
		verifier:                  verifier,
	}
}

// buildOCMReaderForNamespace builds a content.Reader with credentials and
// signature specs resolved from the VectorTemplate in the given namespace.
// If no VectorTemplate exists or carries no credentials the reader runs unauthenticated.
// Returns the override reader directly when one is set (tests only).
func (h *projectHandler) buildOCMReaderForNamespace(ctx context.Context, namespace string) (pkgcontent.Reader, error) {
	if h.ocmReaderOverride != nil {
		return h.ocmReaderOverride, nil
	}
	vtList := &konfidence.VectorTemplateList{}
	if err := h.k8sClient.List(ctx, vtList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("list VectorTemplates in namespace %q: %w", namespace, err)
	}

	var creds *konfidence.Credentials
	var vectorSpecs []crypto.SignatureSpec
	var artifactSpecs []crypto.SignatureSpec
	if len(vtList.Items) > 0 {
		vt := &vtList.Items[0]
		creds = vt.Spec.Credentials
		vectorSpecs = crypto.SpecsFromVerify(vt.Spec.VerifyVector)
		artifactSpecs = crypto.SpecsFromVerify(vt.Spec.VerifyArtifacts)
	}

	resolver, err := pkgcredentials.ResolverFromCredentials(ctx, h.k8sClient, namespace, creds)
	if err != nil {
		return nil, fmt.Errorf("resolve OCM credentials for namespace %q: %w", namespace, err)
	}

	ocmClient, err := pkgocm.NewOciClientBuilder().
		WithLogger(h.log).
		WithResolver(resolver).
		Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("build OCI client for namespace %q: %w", namespace, err)
	}

	return pkgcontent.NewReader(ocmClient, h.verifier, resolver, vectorSpecs, artifactSpecs), nil
}

func (h *projectHandler) ListProjectsV1(ctx context.Context, _ openapi.ListProjectsV1RequestObject) (openapi.ListProjectsV1ResponseObject, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return nil, apierror.NewUnauthorized()
	}
	projects, err := h.projectRepo.List(ctx, identity.ProjectRoles)
	if err != nil {
		return nil, apierror.NewInternal(err)
	}

	data := make([]openapi.Project, len(projects))
	for i, p := range projects {
		data[i] = toProjectResponse(p)
	}

	return openapi.ListProjectsV1200JSONResponse{Data: data}, nil
}

func toProjectResponse(p konfidence.Project) openapi.Project {
	return openapi.Project{
		Id:   p.Name,
		Name: p.Spec.DisplayName,
	}
}

// resolveProjectRequest resolves the authenticated identity and namespace for
// an endpoint operating on a project.
func (h *projectHandler) resolveProjectRequest(ctx context.Context, projectId string) (*session.Context, string, error) {
	identity, err := session.FromContext(ctx)
	if err != nil {
		return nil, "", apierror.NewUnauthorized()
	}

	namespace, err := h.resolveProjectNamespace(ctx, identity, projectId)
	if err != nil {
		return nil, "", err
	}

	return identity, namespace, nil
}

// resolveProjectNamespace handles the preamble common to handlers that operate on
// a project-scoped k8s namespace: it checks authorization, fetches the project, and
// confirms the namespace is set. On failure, it returns an *apierror.Error carrying
// the HTTP status and a client-safe message.
func (h *projectHandler) resolveProjectNamespace(ctx context.Context, identity *session.Context, projectId string) (string, error) {
	if !identity.IsAuthenticatedForProject(projectId) {
		return "", apierror.NewForbidden(fmt.Sprintf("access to project %q is not allowed", projectId))
	}
	project, err := h.projectRepo.Get(ctx, projectId)
	if err != nil {
		if errors.Is(err, projectdomain.ErrProjectNotFound) {
			return "", apierror.NewNotFound("project", projectId)
		}
		return "", apierror.NewInternal(err)
	}
	if project.Status.Namespace == "" {
		return "", apierror.NewInternal(fmt.Errorf("project %q has no namespace configured", projectId))
	}
	return project.Status.Namespace, nil
}
