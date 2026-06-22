package schema

import "github.com/invopop/jsonschema"

const SchemaID = "cloud.konfidence/konfidence-artifact-schema.json"

type ManifestInput struct {
	Type string `json:"type" jsonschema:"enum=file/v1,enum=File/v1"`
}

type ManifestResource struct {
	Type  string        `json:"type"`
	Input ManifestInput `json:"input"`
}

func (ManifestResource) JSONSchemaExtend(schema *jsonschema.Schema) {
	if p, ok := schema.Properties.Get("type"); ok {
		p.Const = "cloud.konfidence.artifact.manifest"
	}
}

type ManifestResourcesArray []ManifestResource

func (ManifestResourcesArray) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.Contains = schema.Items
	schema.Items = nil
	minContains, maxContains := uint64(1), uint64(1)
	schema.MinContains = &minContains
	schema.MaxContains = &maxContains
}

type Component struct {
	Resources ManifestResourcesArray `json:"resources,omitempty"`
}

type ArtifactConstructor struct {
	Components []Component            `json:"components,omitempty"`
	Resources  ManifestResourcesArray `json:"resources,omitempty"`
}

func (ArtifactConstructor) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.Title = "Konfidence Artifact"
	schema.Description = "Konfidence-specific validation rules layered on top of the OCM component constructor schema."
	schema.OneOf = []*jsonschema.Schema{
		{Required: []string{"components"}},
		{Required: []string{"resources"}},
	}
}
