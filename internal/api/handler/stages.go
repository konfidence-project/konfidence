package handler

import (
	"net/http"

	handler "github.com/konfidence-project/konfidence/internal/api/handler/dto"
)

var mockStages = []handler.StageResponse{
	{
		Name:      "dev",
		Namespace: "konfidence-dev",
		Vector:    "github.com/konfidence-project/kden:latest",
		Conditions: []handler.StageCondition{
			{
				Type:               "VectorDeploymentCreated",
				Status:             "True",
				Reason:             "DeploymentCreated",
				Message:            "VectorDeployment resource created successfully",
				LastTransitionTime: "2026-07-01T08:12:00Z",
			},
			{
				Type:               "VectorDeployed",
				Status:             "True",
				Reason:             "ArtifactsDeployed",
				Message:            "All vector artifacts deployed and assigned",
				LastTransitionTime: "2026-07-01T08:15:34Z",
			},
			{
				Type:               "VectorMigrated",
				Status:             "True",
				Reason:             "MigrationComplete",
				Message:            "All migration tasks completed",
				LastTransitionTime: "2026-07-01T08:18:10Z",
			},
			{
				Type:               "Ready",
				Status:             "True",
				Reason:             "StageReady",
				Message:            "Stage is ready",
				LastTransitionTime: "2026-07-01T08:18:10Z",
			},
		},
	},
	{
		Name:      "staging",
		Namespace: "konfidence-staging",
		Vector:    "github.com/konfidence-project/konfidence:1.4.2",
		Conditions: []handler.StageCondition{
			{
				Type:               "VectorDeploymentCreated",
				Status:             "True",
				Reason:             "DeploymentCreated",
				Message:            "VectorDeployment resource created successfully",
				LastTransitionTime: "2026-07-02T14:00:00Z",
			},
			{
				Type:               "VectorDeployed",
				Status:             "True",
				Reason:             "ArtifactsDeployed",
				Message:            "All vector artifacts deployed and assigned",
				LastTransitionTime: "2026-07-02T14:04:22Z",
			},
			{
				Type:               "VectorMigrated",
				Status:             "False",
				Reason:             "MigrationFailed",
				Message:            "Migration task \"db-schema-v2\" failed: timeout after 300s",
				LastTransitionTime: "2026-07-02T14:10:05Z",
			},
			{
				Type:               "Ready",
				Status:             "False",
				Reason:             "MigrationNotComplete",
				Message:            "Stage is not ready: migration incomplete",
				LastTransitionTime: "2026-07-02T14:10:05Z",
			},
		},
	},
	{
		Name:      "prod",
		Namespace: "konfidence-prod",
		Vector:    "github.com/konfidence-project/platform:1.3.9",
		Conditions: []handler.StageCondition{
			{
				Type:               "VectorDeploymentCreated",
				Status:             "True",
				Reason:             "DeploymentCreated",
				Message:            "VectorDeployment resource created successfully",
				LastTransitionTime: "2026-06-28T10:00:00Z",
			},
			{
				Type:               "VectorDeployed",
				Status:             "True",
				Reason:             "ArtifactsDeployed",
				Message:            "All vector artifacts deployed and assigned",
				LastTransitionTime: "2026-06-28T10:06:17Z",
			},
			{
				Type:               "VectorMigrated",
				Status:             "True",
				Reason:             "MigrationComplete",
				Message:            "All migration tasks completed",
				LastTransitionTime: "2026-06-28T10:09:43Z",
			},
			{
				Type:               "Ready",
				Status:             "True",
				Reason:             "StageReady",
				Message:            "Stage is ready",
				LastTransitionTime: "2026-06-28T10:09:43Z",
			},
		},
	},
	{
		Name:      "canary",
		Namespace: "konfidence-prod",
		Vector:    "github.com/konfidence-project/platform:1.4.2",
		Conditions: []handler.StageCondition{
			{
				Type:               "VectorDeploymentCreated",
				Status:             "True",
				Reason:             "DeploymentCreated",
				Message:            "VectorDeployment resource created successfully",
				LastTransitionTime: "2026-07-09T07:30:00Z",
			},
			{
				Type:               "VectorDeployed",
				Status:             "Unknown",
				Reason:             "DeploymentInProgress",
				Message:            "Waiting for artifacts to be assigned",
				LastTransitionTime: "2026-07-09T07:30:00Z",
			},
		},
	},
}

func ListStages(w http.ResponseWriter, _ *http.Request) error {
	writeJSON(w, http.StatusOK, handler.StageListResponse{Stages: mockStages})
	return nil
}
