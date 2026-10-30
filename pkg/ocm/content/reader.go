package content

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/konfidence-project/konfidence/pkg/jsonschema"
	kompref "github.com/konfidence-project/konfidence/pkg/ocm/compref"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"github.com/konfidence-project/konfidence/pkg/ocm/util"
	"ocm.software/open-component-model/bindings/go/credentials"
	descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
)

//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_reader.go -package=mocks github.com/konfidence-project/konfidence/pkg/ocm/content Reader

const (
	resourceTypeArtifactManifest     = "cloud.konfidence.artifact.manifest"
	resourceTypeArtifactTaskManifest = "cloud.konfidence.artifact.task.manifest"
)

// VectorContent holds the parsed content of an OCM vector component descriptor.
type VectorContent struct {
	Components []VectorComponent
	Config     *VectorConfig
}

// VectorComponent is a single component reference within a vector.
type VectorComponent struct {
	ComponentName string
	Version       string
}

// VectorConfig holds the parsed vector-scoped configuration resource.
type VectorConfig struct {
	Features map[string]any
	Authored map[string]any
}

// ArtifactContent holds the parsed content of an OCM artifact component descriptor.
type ArtifactContent struct {
	Type       string
	AllowReuse bool
	Tasks      []ArtifactTask
	Resources  []ArtifactResource
}

// ArtifactTask is a single task manifest entry within an artifact.
type ArtifactTask struct {
	Name      string
	Type      string
	DependsOn []string
}

// ArtifactResource is a non-manifest OCM resource within an artifact.
type ArtifactResource struct {
	Name string
	Type string
}

// Reader fetches OCM content directly from OCI registries.
type Reader interface {
	GetVectorContent(ctx context.Context, rawOcmRef string) (VectorContent, error)
	GetArtifactContent(ctx context.Context, rawOcmRef string) (ArtifactContent, error)
}

type ocmReader struct {
	client        pkgocm.Client
	verifier      crypto.Verifier
	resolver      credentials.Resolver
	vectorSpecs   []crypto.SignatureSpec
	artifactSpecs []crypto.SignatureSpec
}

// NewReader creates a Reader backed by the given client, verifier, resolver, and
// signature specs. Empty spec slices disable verification for that code path.
func NewReader(
	client pkgocm.Client,
	verifier crypto.Verifier,
	resolver credentials.Resolver,
	vectorSpecs, artifactSpecs []crypto.SignatureSpec,
) Reader {
	return &ocmReader{
		client:        client,
		verifier:      verifier,
		resolver:      resolver,
		vectorSpecs:   vectorSpecs,
		artifactSpecs: artifactSpecs,
	}
}

func (r *ocmReader) GetVectorContent(ctx context.Context, rawOcmRef string) (VectorContent, error) {
	upstream, err := kompref.Parse(rawOcmRef)
	if err != nil {
		return VectorContent{}, fmt.Errorf("invalid vector ref %s: %w", rawOcmRef, err)
	}

	descriptor, err := r.client.Get(ctx, *upstream)
	if err != nil {
		return VectorContent{}, fmt.Errorf("get vector descriptor for %s: %w", upstream.String(), err)
	}

	if err := r.verifier.Verify(ctx, r.resolver, r.vectorSpecs, []*descruntime.Descriptor{&descriptor}); err != nil {
		return VectorContent{}, fmt.Errorf("verify vector signature for %s: %w", upstream.String(), err)
	}

	vectorConfigBlob, prunedResources, err := util.ExtractVectorConfigResource(ctx, r.client, *upstream, descriptor.Component.Resources)
	if err != nil {
		return VectorContent{}, err
	}
	descriptor.Component.Resources = prunedResources

	components := make([]VectorComponent, len(descriptor.Component.References))
	for i, cr := range descriptor.Component.References {
		components[i] = VectorComponent{
			ComponentName: cr.Component,
			Version:       cr.Version,
		}
	}

	config, err := parseVectorConfig(vectorConfigBlob)
	if err != nil {
		return VectorContent{}, err
	}

	return VectorContent{Components: components, Config: config}, nil
}

func (r *ocmReader) GetArtifactContent(ctx context.Context, rawOcmRef string) (ArtifactContent, error) {
	upstream, err := kompref.Parse(rawOcmRef)
	if err != nil {
		return ArtifactContent{}, fmt.Errorf("invalid vector ref %s: %w", rawOcmRef, err)
	}

	descriptor, err := r.client.Get(ctx, *upstream)
	if err != nil {
		return ArtifactContent{}, fmt.Errorf("get artifact descriptor for %s: %w", upstream.String(), err)
	}

	if err := r.verifier.Verify(ctx, r.resolver, r.artifactSpecs, []*descruntime.Descriptor{&descriptor}); err != nil {
		return ArtifactContent{}, fmt.Errorf("verify artifact signature for %s: %w", upstream.String(), err)
	}

	var artifactType string
	var allowReuse bool
	var foundManifest bool
	var tasks []ArtifactTask
	var resources []ArtifactResource

	for i := range descriptor.Component.Resources {
		res := &descriptor.Component.Resources[i]
		switch res.Type {
		case resourceTypeArtifactManifest:
			data, err := util.GetLocalResourceBlob(ctx, r.client, *upstream, res)
			if err != nil {
				return ArtifactContent{}, fmt.Errorf("read artifact manifest blob in %s: %w", upstream.String(), err)
			}
			var dto struct {
				Type       string `json:"type"`
				AllowReuse bool   `json:"allowReuse"`
			}
			if err := json.Unmarshal(data, &dto); err != nil {
				return ArtifactContent{}, fmt.Errorf("unmarshal artifact manifest in %s: %w", upstream.String(), err)
			}
			if dto.Type == "" {
				return ArtifactContent{}, fmt.Errorf("artifact manifest in %s is missing required type field", upstream.String())
			}
			artifactType = dto.Type
			allowReuse = dto.AllowReuse
			foundManifest = true

		case resourceTypeArtifactTaskManifest:
			data, err := util.GetLocalResourceBlob(ctx, r.client, *upstream, res)
			if err != nil {
				return ArtifactContent{}, fmt.Errorf("read task manifest blob for resource %s in %s: %w", res.Name, upstream.String(), err)
			}
			var dto struct {
				Name      string   `json:"name"`
				Type      string   `json:"type"`
				DependsOn []string `json:"dependsOn"`
			}
			if err := json.Unmarshal(data, &dto); err != nil {
				return ArtifactContent{}, fmt.Errorf("unmarshal task manifest for resource %s in %s: %w", res.Name, upstream.String(), err)
			}
			tasks = append(tasks, ArtifactTask{
				Name:      dto.Name,
				Type:      dto.Type,
				DependsOn: dto.DependsOn,
			})

		default:
			resources = append(resources, ArtifactResource{
				Name: res.Name,
				Type: res.Type,
			})
		}
	}

	if !foundManifest {
		return ArtifactContent{}, fmt.Errorf("no artifact manifest resource found in %s", upstream.String())
	}

	return ArtifactContent{
		Type:       artifactType,
		AllowReuse: allowReuse,
		Tasks:      tasks,
		Resources:  resources,
	}, nil
}

func parseVectorConfig(blob []byte) (*VectorConfig, error) {
	if len(blob) == 0 {
		return nil, nil
	}
	var env jsonschema.VectorConfigurationV1
	if err := json.Unmarshal(blob, &env); err != nil {
		return nil, fmt.Errorf("vector config blob is not valid: %w", err)
	}
	if env.SchemaVersion != "" && env.SchemaVersion != jsonschema.VectorConfigurationV1SchemaVersion {
		return nil, fmt.Errorf("unsupported vector config schemaVersion %q (want %q)",
			env.SchemaVersion, jsonschema.VectorConfigurationV1SchemaVersion)
	}
	if len(env.Features) == 0 && len(env.Authored) == 0 {
		return nil, nil
	}
	cfg := &VectorConfig{}
	if len(env.Features) > 0 {
		var f map[string]any
		if err := json.Unmarshal(env.Features, &f); err != nil {
			return nil, fmt.Errorf("vector config features blob is malformed: %w", err)
		}
		cfg.Features = f
	}
	if len(env.Authored) > 0 {
		var a map[string]any
		if err := json.Unmarshal(env.Authored, &a); err != nil {
			return nil, fmt.Errorf("vector config authored blob is malformed: %w", err)
		}
		cfg.Authored = a
	}

	return cfg, nil
}
