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
