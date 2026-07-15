package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	dto "github.com/konfidence-project/konfidence/internal/api/handler/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/api/handler"
)

var mockStages = []dto.StageResponse{
	{
		Name:      "prod",
		Namespace: "konfidence-prod",
		Vector:    "github.com/konfidence-project/platform:1.3.9",
		Conditions: []dto.StageCondition{
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
}

var _ = Describe("ListStage handlers", func() {
	Describe("ListStages", func() {
		It("returns 200 with status ok and response body", func() {
			rec := httptest.NewRecorder()
			Expect(handler.ListStages(rec, httptest.NewRequest(http.MethodGet, "/api/v1/stages", nil))).To(Succeed())

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Header().Get("Content-Type")).To(Equal("application/json"))

			var body dto.StageListResponse
			Expect(json.NewDecoder(rec.Body).Decode(&body)).To(Succeed())
			Expect(body.Stages).To(ContainElement(mockStages[0]))
		})
	})
})

// TODO: API error tests can be implemented for non-mock api data
