package pretty_test

import (
	"fmt"

	"charm.land/bubbles/v2/table"
	"github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
	"github.com/konfidence-project/konfidence/internal/kden/validation/output"
	"github.com/konfidence-project/konfidence/pkg/build"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Column titles shared across the list-table specs, extracted so goconst does
// not flag the repeated literals once several tables reuse the same titles.
const (
	colName   = "Name"
	colStatus = "Status"
)

var _ = Describe("GetModelFuncMap", func() {

	Describe("GetModelFuncMap", func() {
		It("should contain the validate key", func() {
			m := pretty.GetModelFuncMap()
			Expect(m).NotTo(BeNil())
			Expect(m).To(HaveKey("validate"))
		})
	})

	Describe("validate model function", func() {
		validationErrors := []output.SchemaValidationError{
			{File: "a.yaml", Path: "/foo", Message: "required"},
			{File: "b.yaml", Path: "/bar", Message: "invalid type"},
		}

		Context("when called with a valid []SchemaValidationError", func() {
			It("should return TableData with correct columns", func() {
				result := pretty.GetModelFuncMap()["validate"](validationErrors)

				Expect(result.Err).ToNot(HaveOccurred())
				Expect(result.Columns).To(HaveLen(3))
				Expect(result.Columns[0].Title).To(Equal("File"))
				Expect(result.Columns[1].Title).To(Equal("Path"))
				Expect(result.Columns[2].Title).To(Equal("Message"))
			})

			It("should return TableData with one row per error", func() {
				result := pretty.GetModelFuncMap()["validate"](validationErrors)

				Expect(result.Err).ToNot(HaveOccurred())
				Expect(result.Rows).To(HaveLen(len(validationErrors)))
				Expect(result.Rows[0]).To(Equal(table.Row{validationErrors[0].File, validationErrors[0].Path, validationErrors[0].Message}))
				Expect(result.Rows[1]).To(Equal(table.Row{validationErrors[1].File, validationErrors[1].Path, validationErrors[1].Message}))
			})

			It("returns empty rows for an empty slice", func() {
				result := pretty.GetModelFuncMap()["validate"]([]output.SchemaValidationError{})

				Expect(result.Err).NotTo(HaveOccurred())
				Expect(result.Rows).To(BeEmpty())
			})
		})

		Context("when called with the wrong type", func() {
			It("should return TableData with a descriptive error", func() {
				result := pretty.GetModelFuncMap()["validate"]("not a slice")

				Expect(result).NotTo(BeNil())
				Expect(result.Err).To(HaveOccurred())
				Expect(result.Err.Error()).To(ContainSubstring("error while creating table for command validate"))
				Expect(result.Err.Error()).To(ContainSubstring(fmt.Sprintf("expected %T", ([]output.SchemaValidationError)(nil))))
			})
		})
	})

	Describe("version model function", func() {
		info := build.Info{
			Version:   "1.0.0",
			Commit:    "foo",
			GoVersion: "go1.21",
			Platform:  "bar",
			Date:      "2026-01-01",
		}

		It("returns version fields and footer", func() {
			result := pretty.GetModelFuncMap()["version"](info)

			Expect(result.Err).NotTo(HaveOccurred())
			Expect(result.Columns).To(Equal([]table.Column{
				{Title: "Field", Width: 12},
				{Title: "Value", Width: 60},
			}))
			Expect(result.Rows).To(Equal([]table.Row{
				{"Version", info.Version},
				{"Commit", info.Commit},
				{"Go", info.GoVersion},
				{"Platform", info.Platform},
				{"built", info.Date},
			}))
			Expect(result.Footer).NotTo(BeEmpty())
		})

		It("rejects an unexpected response type", func() {
			result := pretty.GetModelFuncMap()["version"]("not a build.Info")

			Expect(result.Err).To(MatchError(ContainSubstring(fmt.Sprintf("expected %T", build.Info{}))))
		})
	})

	//nolint:dupl // list model function tests intentionally share a common ID/Name table shape across entities
	Describe("project list model function", func() {
		projectList := &apiclient.ProjectList{Data: []apiclient.Project{
			{Id: "project-a", Name: "Project A"},
			{Id: "project-b", Name: "Project B"},
		}}

		It("returns project ID and name rows", func() {
			result := pretty.GetModelFuncMap()["project-list"](projectList)

			Expect(result.Err).NotTo(HaveOccurred())
			Expect(result.Columns).To(Equal([]table.Column{
				{Title: "ID", Width: 40},
				{Title: colName, Width: 40},
			}))
			Expect(result.Rows).To(Equal([]table.Row{
				{projectList.Data[0].Id, projectList.Data[0].Name},
				{projectList.Data[1].Id, projectList.Data[1].Name},
			}))
		})

		It("returns empty rows for an empty list", func() {
			result := pretty.GetModelFuncMap()["project-list"](&apiclient.ProjectList{})

			Expect(result.Err).NotTo(HaveOccurred())
			Expect(result.Rows).To(BeEmpty())
		})

		It("rejects an unexpected response type", func() {
			result := pretty.GetModelFuncMap()["project-list"]("not a project list")

			Expect(result.Err).To(MatchError(ContainSubstring(fmt.Sprintf("expected %T", (*apiclient.ProjectList)(nil)))))
		})
	})

	//nolint:dupl // list model function tests intentionally share a common ID/Name table shape across entities
	Describe("landscape list model function", func() {
		landscapeList := &apiclient.LandscapeList{Data: []apiclient.Landscape{
			{Id: "landscape-a", Name: "Landscape A"},
			{Id: "landscape-b", Name: "Landscape B"},
		}}

		It("returns landscape ID and name rows", func() {
			result := pretty.GetModelFuncMap()["landscape-list"](landscapeList)

			Expect(result.Err).NotTo(HaveOccurred())
			Expect(result.Columns).To(Equal([]table.Column{
				{Title: "ID", Width: 40},
				{Title: colName, Width: 40},
			}))
			Expect(result.Rows).To(Equal([]table.Row{
				{landscapeList.Data[0].Id, landscapeList.Data[0].Name},
				{landscapeList.Data[1].Id, landscapeList.Data[1].Name},
			}))
		})

		It("returns empty rows for an empty list", func() {
			result := pretty.GetModelFuncMap()["landscape-list"](&apiclient.LandscapeList{})

			Expect(result.Err).NotTo(HaveOccurred())
			Expect(result.Rows).To(BeEmpty())
		})

		It("rejects an unexpected response type", func() {
			result := pretty.GetModelFuncMap()["landscape-list"]("not a landscape list")

			Expect(result.Err).To(MatchError(ContainSubstring(fmt.Sprintf("expected %T", (*apiclient.LandscapeList)(nil)))))
		})
	})

	Describe("stage list model function", func() {
		activeVersion := &apiclient.StageVersion{Id: "sv-1", Status: "Ready"}
		stageList := &apiclient.StageList{Data: []apiclient.Stage{
			{Id: "stage-a", Name: "Stage A", LandscapeId: "landscape-a", ActiveStageVersion: activeVersion},
			{Id: "stage-b", Name: "Stage B", LandscapeId: "landscape-b"},
		}}

		It("returns a row per stage with the active version and status", func() {
			result := pretty.GetModelFuncMap()["stage-list"](stageList)

			Expect(result.Err).NotTo(HaveOccurred())
			Expect(result.Columns).To(Equal([]table.Column{
				{Title: "ID", Width: 40},
				{Title: colName, Width: 30},
				{Title: "Landscape", Width: 30},
				{Title: "Active Version", Width: 40},
				{Title: colStatus, Width: 20},
			}))
			Expect(result.Rows).To(Equal([]table.Row{
				{stageList.Data[0].Id, stageList.Data[0].Name, stageList.Data[0].LandscapeId, activeVersion.Id, string(activeVersion.Status)},
				{stageList.Data[1].Id, stageList.Data[1].Name, stageList.Data[1].LandscapeId, "-", "-"},
			}))
		})

		It("renders a dash when no stage version is active", func() {
			stage := apiclient.Stage{Id: "stage-c", Name: "Stage C", LandscapeId: "landscape-c"}
			result := pretty.GetModelFuncMap()["stage-list"](&apiclient.StageList{Data: []apiclient.Stage{stage}})

			Expect(result.Err).NotTo(HaveOccurred())
			Expect(result.Rows).To(Equal([]table.Row{{stage.Id, stage.Name, stage.LandscapeId, "-", "-"}}))
		})

		It("returns empty rows for an empty list", func() {
			result := pretty.GetModelFuncMap()["stage-list"](&apiclient.StageList{})

			Expect(result.Err).NotTo(HaveOccurred())
			Expect(result.Rows).To(BeEmpty())
		})

		It("rejects an unexpected response type", func() {
			result := pretty.GetModelFuncMap()["stage-list"]("not a stage list")

			Expect(result.Err).To(MatchError(ContainSubstring(fmt.Sprintf("expected %T", (*apiclient.StageList)(nil)))))
		})
	})

	Describe("vector deployment list model function", func() {
		deploymentList := &apiclient.VectorDeploymentList{Data: []apiclient.VectorDeployment{
			{
				Id: "vd-a", StageId: "stage-d", LandscapeId: "landscape-d", Status: "Ready",
				Vector: apiclient.VectorReference{ComponentName: "comp", ComponentVersion: "1.2.3"},
			},
		}}

		It("returns a row per deployment with a formatted vector", func() {
			result := pretty.GetModelFuncMap()["vector-deployment-list"](deploymentList)

			d := deploymentList.Data[0]
			Expect(result.Err).NotTo(HaveOccurred())
			Expect(result.Columns).To(Equal([]table.Column{
				{Title: "ID", Width: 40},
				{Title: "Stage", Width: 30},
				{Title: "Landscape", Width: 30},
				{Title: colStatus, Width: 20},
				{Title: "Vector", Width: 40},
			}))
			Expect(result.Rows).To(Equal([]table.Row{
				{d.Id, d.StageId, d.LandscapeId, string(d.Status), fmt.Sprintf("%s:%s", d.Vector.ComponentName, d.Vector.ComponentVersion)},
			}))
		})

		It("returns empty rows for an empty list", func() {
			result := pretty.GetModelFuncMap()["vector-deployment-list"](&apiclient.VectorDeploymentList{})

			Expect(result.Err).NotTo(HaveOccurred())
			Expect(result.Rows).To(BeEmpty())
		})

		It("rejects an unexpected response type", func() {
			result := pretty.GetModelFuncMap()["vector-deployment-list"]("not a deployment list")

			Expect(result.Err).To(MatchError(ContainSubstring(fmt.Sprintf("expected %T", (*apiclient.VectorDeploymentList)(nil)))))
		})
	})

	Describe("vector promotion config list model function", func() {
		promoting := apiclient.VectorPromotionStatus("Promoting")
		configList := &apiclient.VectorPromotionConfigList{Data: []apiclient.VectorPromotionConfig{
			{
				Id:     "cfg-a",
				Source: apiclient.PromotionSourceReference{Name: "src-a"},
				Target: apiclient.PromotionTargetReference{Name: "tgt-a"},
				Promotions: []apiclient.VectorPromotion{
					{
						Id:     "promo-1",
						Source: apiclient.PromotionSourceReference{Name: "psrc-1"},
						Target: apiclient.PromotionTargetReference{Name: "ptgt-1"},
						Vector: "1.0.0",
						Status: &promoting,
					},
					{
						Id:     "promo-2",
						Source: apiclient.PromotionSourceReference{Name: "psrc-2"},
						Target: apiclient.PromotionTargetReference{Name: "ptgt-2"},
						Vector: "2.0.0",
					},
				},
			},
			{
				Id:     "cfg-b",
				Source: apiclient.PromotionSourceReference{Name: "src-b"},
				Target: apiclient.PromotionTargetReference{Name: "tgt-b"},
			},
		}}

		It("returns one row per promotion and a placeholder row for configs without promotions", func() {
			result := pretty.GetModelFuncMap()["vector-promotion-config-list"](configList)

			cfgA := configList.Data[0]
			p1, p2 := cfgA.Promotions[0], cfgA.Promotions[1]
			cfgB := configList.Data[1]
			Expect(result.Err).NotTo(HaveOccurred())
			Expect(result.Columns).To(Equal([]table.Column{
				{Title: "Config ID", Width: 40},
				{Title: "Promotion ID", Width: 40},
				{Title: "Source", Width: 24},
				{Title: "Target", Width: 24},
				{Title: "Vector", Width: 30},
				{Title: colStatus, Width: 16},
			}))
			Expect(result.Rows).To(Equal([]table.Row{
				{cfgA.Id, p1.Id, p1.Source.Name, p1.Target.Name, p1.Vector, string(*p1.Status)},
				{cfgA.Id, p2.Id, p2.Source.Name, p2.Target.Name, p2.Vector, "-"},
				{cfgB.Id, "-", cfgB.Source.Name, cfgB.Target.Name, "-", "-"},
			}))
		})

		It("returns empty rows for an empty list", func() {
			result := pretty.GetModelFuncMap()["vector-promotion-config-list"](&apiclient.VectorPromotionConfigList{})

			Expect(result.Err).NotTo(HaveOccurred())
			Expect(result.Rows).To(BeEmpty())
		})

		It("rejects an unexpected response type", func() {
			result := pretty.GetModelFuncMap()["vector-promotion-config-list"]("not a promotion config list")

			Expect(result.Err).To(MatchError(ContainSubstring(fmt.Sprintf("expected %T", (*apiclient.VectorPromotionConfigList)(nil)))))
		})
	})
})
