package api

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/api/openapi"
)

var _ = Describe("StageHandler", func() {
	var h openapi.StrictServerInterface

	BeforeEach(func() {
		h = &StageHandler{k8s: nil}
	})

	Describe("ListStages", func() {
		It("returns all four mock stages", func() {
			resp, err := h.ListStages(context.Background(), openapi.ListStagesRequestObject{})
			Expect(err).NotTo(HaveOccurred())

			ok, is200 := resp.(openapi.ListStages200JSONResponse)
			Expect(is200).To(BeTrue())
			Expect(ok.Items).To(HaveLen(4))

			names := make([]string, len(ok.Items))
			for i, s := range ok.Items {
				names[i] = s.Name
			}
			Expect(names).To(ConsistOf("dev", "staging", "prod", "canary"))
		})

		It("returns stages with at least one condition each", func() {
			resp, err := h.ListStages(context.Background(), openapi.ListStagesRequestObject{})
			Expect(err).NotTo(HaveOccurred())

			for _, s := range resp.(openapi.ListStages200JSONResponse).Items {
				Expect(s.Conditions).NotTo(BeEmpty(), "stage %q has no conditions", s.Name)
			}
		})

		It("returns correct namespaces", func() {
			resp, err := h.ListStages(context.Background(), openapi.ListStagesRequestObject{})
			Expect(err).NotTo(HaveOccurred())

			byName := map[string]openapi.StageResponse{}
			for _, s := range resp.(openapi.ListStages200JSONResponse).Items {
				byName[s.Name] = s
			}
			Expect(byName["dev"].Namespace).To(Equal("konfidence-dev"))
			Expect(byName["staging"].Namespace).To(Equal("konfidence-staging"))
			Expect(byName["prod"].Namespace).To(Equal("konfidence-prod"))
			Expect(byName["canary"].Namespace).To(Equal("konfidence-prod"))
		})
	})

	Describe("GetStage", func() {
		DescribeTable("returns the correct stage by name",
			func(name, namespace string) {
				resp, err := h.GetStage(context.Background(), openapi.GetStageRequestObject{Name: name})
				Expect(err).NotTo(HaveOccurred())

				ok, is200 := resp.(getStage200Response)
				Expect(is200).To(BeTrue())
				Expect(ok.Items).To(HaveLen(1))
				Expect(ok.Items[0].Name).To(Equal(name))
				Expect(ok.Items[0].Namespace).To(Equal(namespace))
			},
			Entry("dev", "dev", "konfidence-dev"),
			Entry("staging", "staging", "konfidence-staging"),
			Entry("prod", "prod", "konfidence-prod"),
			Entry("canary", "canary", "konfidence-prod"),
		)

		It("returns 404 for an unknown stage", func() {
			resp, err := h.GetStage(context.Background(), openapi.GetStageRequestObject{Name: "nonexistent"})
			Expect(err).NotTo(HaveOccurred())

			r, is404 := resp.(openapi.GetStage404JSONResponse)
			Expect(is404).To(BeTrue())
			Expect(r.Error.Code).To(Equal("not_found"))
		})

		It("includes the stage name in the 404 error message", func() {
			resp, err := h.GetStage(context.Background(), openapi.GetStageRequestObject{Name: "ghost"})
			Expect(err).NotTo(HaveOccurred())

			r := resp.(openapi.GetStage404JSONResponse)
			Expect(r.Error.Message).To(ContainSubstring("ghost"))
		})

		It("staging has a failed migration condition", func() {
			resp, err := h.GetStage(context.Background(), openapi.GetStageRequestObject{Name: "staging"})
			Expect(err).NotTo(HaveOccurred())

			s := resp.(getStage200Response)
			Expect(s.Items).To(HaveLen(1))
			var migrated *openapi.StageCondition
			for i, c := range s.Items[0].Conditions {
				if c.Type == "VectorMigrated" {
					migrated = &s.Items[0].Conditions[i]
					break
				}
			}
			Expect(migrated).NotTo(BeNil())
			Expect(migrated.Status).To(Equal(openapi.StageConditionStatus("False")))
			Expect(migrated.Reason).To(Equal("MigrationFailed"))
		})
	})
})
