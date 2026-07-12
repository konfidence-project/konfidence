package apiclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	handler "github.com/konfidence-project/konfidence/internal/api/handler/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
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

var _ = Describe("Client", func() {
	var srv *httptest.Server
	var client *apiclient.Client

	BeforeEach(func() {
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}))
		client = apiclient.New(srv.URL, 5*time.Second)
	})

	AfterEach(func() { srv.Close() })

	Describe("Healthz", func() {
		It("returns status ok from a live server", func() {
			resp, err := client.Healthz(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Status).To(Equal("ok"))
		})
	})

	Describe("Readyz", func() {
		It("returns status ok from a live server", func() {
			resp, err := client.Readyz(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Status).To(Equal("ok"))
		})
	})

	Describe("ListStages", func() {
		It("returns status ok and response body from a live server", func() {
			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(handler.StageListResponse{Stages: mockStages})
			}))
			defer srv.Close()
			client = apiclient.New(srv.URL, 5*time.Second)

			resp, err := client.ListStages(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Stages).To(Equal(mockStages))
		})
	})

	Describe("error handling", func() {
		It("returns an error when the server responds with non-200", func() {
			errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			defer errSrv.Close()

			c := apiclient.New(errSrv.URL, 5*time.Second)
			_, err := c.Healthz(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("503"))
		})

		It("returns an error when the server is unreachable", func() {
			c := apiclient.New("http://127.0.0.1:1", 100*time.Millisecond)
			_, err := c.Healthz(context.Background())
			Expect(err).To(HaveOccurred())
		})
	})
})
