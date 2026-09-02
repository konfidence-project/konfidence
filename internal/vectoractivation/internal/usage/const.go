package usage

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
)

const (
	// ActiveStageVersion is the label marking the active StageVersionUsage of a stage.
	// It is part of the Stage's public contract and lives in api/v1alpha1.
	ActiveStageVersion              = konfidence.ActiveStageVersionLabel
	ActivationStageVersionUsage     = "konfidence.cloud/activation-stage-version-usage"
	StageVersionUsageActivationType = "Activation"
	StageVersionUsageActiveType     = "Active"
)
