package apiclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
)

func mustTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func cond(condType string, status apiclient.StageConditionStatus, reason, message, ts string) apiclient.StageCondition {
	return apiclient.StageCondition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: mustTime(ts),
	}
}

var mockStages = []apiclient.StageResponse{
	{
		Name: "dev", Namespace: "konfidence-dev", Vector: "github.com/konfidence-project/kden:latest",
		Conditions: []apiclient.StageCondition{
			cond("VectorDeploymentCreated", apiclient.True, "DeploymentCreated", "VectorDeployment resource created successfully", "2026-07-01T08:12:00Z"),
			cond("VectorDeployed", apiclient.True, "ArtifactsDeployed", "All vector artifacts deployed and assigned", "2026-07-01T08:15:34Z"),
			cond("VectorMigrated", apiclient.True, "MigrationComplete", "All migration tasks completed", "2026-07-01T08:18:10Z"),
			cond("Ready", apiclient.True, "StageReady", "Stage is ready", "2026-07-01T08:18:10Z"),
		},
	},
	{
		Name: "staging", Namespace: "konfidence-staging", Vector: "github.com/konfidence-project/konfidence:1.4.2",
		Conditions: []apiclient.StageCondition{
			cond("VectorDeploymentCreated", apiclient.True, "DeploymentCreated", "VectorDeployment resource created successfully", "2026-07-02T14:00:00Z"),
			cond("VectorDeployed", apiclient.True, "ArtifactsDeployed", "All vector artifacts deployed and assigned", "2026-07-02T14:04:22Z"),
			cond("VectorMigrated", apiclient.False, "MigrationFailed", "Migration task \"db-schema-v2\" failed: timeout after 300s", "2026-07-02T14:10:05Z"),
			cond("Ready", apiclient.False, "MigrationNotComplete", "Stage is not ready: migration incomplete", "2026-07-02T14:10:05Z"),
		},
	},
	{
		Name: "prod", Namespace: "konfidence-prod", Vector: "github.com/konfidence-project/platform:1.3.9",
		Conditions: []apiclient.StageCondition{
			cond("VectorDeploymentCreated", apiclient.True, "DeploymentCreated", "VectorDeployment resource created successfully", "2026-06-28T10:00:00Z"),
			cond("VectorDeployed", apiclient.True, "ArtifactsDeployed", "All vector artifacts deployed and assigned", "2026-06-28T10:06:17Z"),
			cond("VectorMigrated", apiclient.True, "MigrationComplete", "All migration tasks completed", "2026-06-28T10:09:43Z"),
			cond("Ready", apiclient.True, "StageReady", "Stage is ready", "2026-06-28T10:09:43Z"),
		},
	},
	{
		Name: "canary", Namespace: "konfidence-prod", Vector: "github.com/konfidence-project/platform:1.4.2",
		Conditions: []apiclient.StageCondition{
			cond("VectorDeploymentCreated", apiclient.True, "DeploymentCreated", "VectorDeployment resource created successfully", "2026-07-09T07:30:00Z"),
			cond("VectorDeployed", apiclient.Unknown, "DeploymentInProgress", "Waiting for artifacts to be assigned", "2026-07-09T07:30:00Z"),
		},
	},
}

var _ = Describe("Client", func() {
	var srv *httptest.Server
	var client *apiclient.ClientWithResponses

	BeforeEach(func() {
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.URL.Path == "/api/v1/stages" && r.Method == http.MethodGet:
				_ = json.NewEncoder(w).Encode(apiclient.StageListResponse{Items: mockStages})
			case r.Method == http.MethodGet && len(r.URL.Path) > len("/api/v1/stages/"):
				name := r.URL.Path[len("/api/v1/stages/"):]
				for _, s := range mockStages {
					if s.Name == name {
						_ = json.NewEncoder(w).Encode(s)
						return
					}
				}
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{"code": "not_found", "message": "stage not found"},
				})
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		var err error
		client, err = apiclient.NewClientWithResponses(srv.URL + "/api/v1")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() { srv.Close() })

	Describe("ListStages", func() {
		It("returns all mock stages from a live server", func() {
			resp, err := client.ListStagesWithResponse(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode()).To(Equal(http.StatusOK))
			Expect(resp.JSON200).NotTo(BeNil())
			Expect(resp.JSON200.Items).To(HaveLen(4))
			Expect(resp.JSON200.Items[0].Name).To(Equal("dev"))
			Expect(resp.JSON200.Items[1].Name).To(Equal("staging"))
			Expect(resp.JSON200.Items[2].Name).To(Equal("prod"))
			Expect(resp.JSON200.Items[3].Name).To(Equal("canary"))
		})

		It("includes staging with a failed migration condition", func() {
			resp, err := client.ListStagesWithResponse(context.Background())
			Expect(err).NotTo(HaveOccurred())
			staging := resp.JSON200.Items[1]
			Expect(staging.Name).To(Equal("staging"))
			ready := staging.Conditions[len(staging.Conditions)-1]
			Expect(string(ready.Status)).To(Equal(string(apiclient.False)))
			Expect(ready.Reason).To(Equal("MigrationNotComplete"))
		})

		It("includes canary with an in-progress deployment condition", func() {
			resp, err := client.ListStagesWithResponse(context.Background())
			Expect(err).NotTo(HaveOccurred())
			canary := resp.JSON200.Items[3]
			Expect(canary.Name).To(Equal("canary"))
			Expect(canary.Conditions).To(HaveLen(2))
			Expect(string(canary.Conditions[1].Status)).To(Equal(string(apiclient.Unknown)))
			Expect(canary.Conditions[1].Reason).To(Equal("DeploymentInProgress"))
		})
	})

	Describe("GetStage", func() {
		DescribeTable("returns the correct stage by name",
			func(name string, expectedVector string, expectedConditions int) {
				resp, err := client.GetStageWithResponse(context.Background(), name)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode()).To(Equal(http.StatusOK))
				Expect(resp.JSON200).NotTo(BeNil())
				Expect(resp.JSON200.Name).To(Equal(name))
				Expect(resp.JSON200.Vector).To(Equal(expectedVector))
				Expect(resp.JSON200.Conditions).To(HaveLen(expectedConditions))
			},
			Entry("dev — all 4 conditions", "dev", "github.com/konfidence-project/kden:latest", 4),
			Entry("staging — failed migration", "staging", "github.com/konfidence-project/konfidence:1.4.2", 4),
			Entry("prod — fully ready", "prod", "github.com/konfidence-project/platform:1.3.9", 4),
			Entry("canary — in progress", "canary", "github.com/konfidence-project/platform:1.4.2", 2),
		)

		It("returns 404 for an unknown stage", func() {
			resp, err := client.GetStageWithResponse(context.Background(), "unknown")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode()).To(Equal(http.StatusNotFound))
		})
	})

	Describe("error handling", func() {
		It("returns an error when the server responds with non-200", func() {
			errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			defer errSrv.Close()

			c, err := apiclient.NewClientWithResponses(errSrv.URL + "/api/v1")
			Expect(err).NotTo(HaveOccurred())
			resp, err := c.ListStagesWithResponse(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode()).To(Equal(http.StatusServiceUnavailable))
		})

		It("returns an error when the server is unreachable", func() {
			c, err := apiclient.NewClientWithResponses("http://127.0.0.1:1/api/v1")
			Expect(err).NotTo(HaveOccurred())
			_, err = c.ListStagesWithResponse(context.Background())
			Expect(err).To(HaveOccurred())
		})
	})
})
