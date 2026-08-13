package v1alpha1

// StageReference holds a reference to a Stage in the same namespace.
type StageReference struct {
	// Name is the name of the Stage. Required.
	Name string `json:"name"`
}

// StageVersionReference holds a reference to a StageVersion in the same namespace.
type StageVersionReference struct {
	// Name is the name of the StageVersion. Required.
	Name string `json:"name"`
}

// LocalArtifactDeploymentReference holds a reference to an ArtifactDeployment in the same namespace.
type LocalArtifactDeploymentReference struct {
	// Name	is the name of the ArtifactDeployment. Required.
	Name string `json:"name"`

	// CollisionCount salts the ArtifactDeployment name hash to recover from a
	// (rare) hash collision with a different artifact. nil and 0 both mean "no
	// salt" and yield the original, unsalted name. Once bumped it is permanent
	// for this artifact slot. Mirrors Deployment.Status.CollisionCount.
	// +optional
	CollisionCount *int32 `json:"collisionCount,omitempty"`
}

// LocalVectorDeploymentReference holds a reference to a VectorDeployment in the same namespace.
type LocalVectorDeploymentReference struct {
	// Name	is the name of the VectorDeployment. Required.
	Name string `json:"name"`
}

// LocalVectorAssignmentReference holds a reference to a VectorAssignment in the same namespace.
type LocalVectorAssignmentReference struct {
	// Name	is the name of the VectorAssignment. Required.
	Name string `json:"name"`
}

// VectorTemplateReference holds a reference to a VectorTemplate in the same namespace,
// used as the base of another VectorTemplate.
type VectorTemplateReference struct {
	// Kind is the kind of the referenced object. Only VectorTemplate is supported for now.
	// +kubebuilder:validation:Enum=VectorTemplate
	// +kubebuilder:default=VectorTemplate
	Kind string `json:"kind"`

	// Name is the name of the referenced VectorTemplate. Required.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// PromotionSourceReference identifies the in-cluster resource whose current
// vector is promoted from.
//
// +kubebuilder:validation:XValidation:rule="(self.kind == 'Stage') == has(self.landscape)",message="landscape is required for Stage references and must be omitted for VectorTemplate references"
//
//nolint:lll // Kubebuilder annotations are intentionally long.
type PromotionSourceReference struct {
	// Kind is the kind of the source resource. A `VectorTemplate` source
	// promotes its latest assembled vector; a `Stage` source promotes the
	// vector currently configured on that stage (`spec.vector`). Whether the resulting promotion
	// requires approval is recorded on the promotion itself
	// (`VectorPromotionSpec.RequireApproval`); the controller defaults it
	// from the source kind.
	// +kubebuilder:validation:Enum=VectorTemplate;Stage
	Kind string `json:"kind"`

	// Name is the name of the source resource.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`

	// Landscape is the `metadata.name` of the `Landscape` in the config's
	// namespace (not its managed namespace) whose namespace hosts the
	// referenced `Stage`. Required for `Stage` references; must be omitted
	// for `VectorTemplate` references, which are resolved in the config's
	// namespace.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +optional
	Landscape string `json:"landscape,omitempty"`
}

// PromotionTargetReference identifies the `Stage` whose `spec.vector` is the
// promotion target.
type PromotionTargetReference struct {
	// Kind is the kind of the target resource. Only `Stage` is supported.
	// +kubebuilder:validation:Enum=Stage
	Kind string `json:"kind"`

	// Name is the name of the target `Stage`.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`

	// Landscape is the `metadata.name` of the `Landscape` in the config's
	// namespace (not its managed namespace) whose namespace hosts the target
	// `Stage`.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Landscape string `json:"landscape"`
}
