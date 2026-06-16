package pretty_test

import (
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TableModel View", func() {

	Describe("Init", func() {
		Context("when the data fetch function is nil", func() {
			It("should return cmd with no components initialized", func() {
				model := pretty.NewTableModel(nil, true, nil)
				cmd := model.Init()
				Expect(cmd).To(BeNil())
			})
		})
		Context("when the data fetch function is present", func() {
			It("should return cmd with components initialized", func() {
				model := pretty.NewTableModel(func(_ interface{}) *pretty.TableData {
					return &pretty.TableData{}
				}, true, nil)
				cmd := model.Init()
				Expect(cmd).NotTo(BeNil())
			})
		})
	})

	Describe("View", func() {
		Context("when the loading property is set to true", func() {
			It("should return view with spinner component", func() {
				model := pretty.NewTableModel(nil, true, nil)
				view := model.View()

				Expect(view).NotTo(BeNil())
				Expect(view.Content).NotTo(BeEmpty())
				Expect(view.Content).To(ContainSubstring("Loading..."))
			})
		})
		Context("when the loading property is set to false", func() {
			It("should return view with table component", func() {
				model := pretty.NewTableModel(nil, true, nil)
				model.Update(&pretty.TableData{})
				view := model.View()

				Expect(view).NotTo(BeNil())
				Expect(view.Content).NotTo(BeEmpty())
				Expect(view.Content).NotTo(ContainSubstring("Loading..."))
			})
		})
	})

	Describe("Update", func() {
		Context("when table data is loaded for update", func() {
			It("should return new table with the fetched data", func() {
				model := pretty.NewTableModel(nil, true, nil)
				teaModel, cmd := model.Update(&pretty.TableData{})

				Expect(teaModel).NotTo(BeNil())
				Expect(cmd).To(BeNil())
			})
		})
		Context("when quit keys are pressed", func() {
			It("should return command to quit the operation", func() {
				model := pretty.NewTableModel(nil, true, nil)
				teaModel, cmd := model.Update(tea.KeyPressMsg{Text: pretty.QuitKey})

				Expect(teaModel).NotTo(BeNil())
				Expect(cmd()).To(Equal(tea.QuitMsg{}))
			})
		})
		Context("when spinner's tick output is triggered", func() {
			It("should update the spinners's state", func() {
				model := pretty.NewTableModel(nil, true, nil)
				teaModel, cmd := model.Update(spinner.TickMsg{})

				Expect(teaModel).NotTo(BeNil())
				Expect(cmd).NotTo(BeNil())
			})
		})
	})
})
