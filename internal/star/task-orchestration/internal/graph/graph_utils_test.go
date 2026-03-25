//nolint:staticcheck // ST1001: allow dot-import for test utils using Gomega
package graph

import (
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Graph utility functions", func() {
	const (
		Task0 = "task-0"
		Task1 = "task-1"
		Task2 = "task-2"
		Task3 = "task-3"
		Task4 = "task-4"
		Task5 = "task-5"
		Task6 = "task-6"
	)

	Context("When processing a task dependency graph", func() {
		It("should successfully sort tasks topologically", func() {
			tasks := []landscape.TaskManifest{
				{
					Name: Task0,
					Type: "k8s",
				},
				{
					Name: Task1,
					Type: "k8s",
				},
				{
					Name:      Task2,
					Type:      "k8s",
					DependsOn: []string{Task0, Task1},
				},
				{
					Name:      Task3,
					Type:      "k8s",
					DependsOn: []string{Task0, Task2},
				},
				{
					Name:      Task4,
					Type:      "k8s",
					DependsOn: []string{Task2},
				},
				{
					Name:      Task5,
					Type:      "k8s",
					DependsOn: []string{Task3},
				},
				{
					Name:      Task6,
					Type:      "k8s",
					DependsOn: []string{Task4, Task5},
				},
			}

			result, layers, err := SortTasks(tasks)
			Expect(err).ToNot(HaveOccurred(), "Failed to sort task dependency graph")
			Expect(result).To(HaveLen(len(tasks)))

			var sortedTaskNames = make([]string, len(result), len(result))
			for i, task := range result {
				sortedTaskNames[i] = task.Name
			}

			expectedTaskNames := []string{Task0, Task1, Task2, Task3, Task4, Task5, Task6}
			Expect(sortedTaskNames).To(Equal(expectedTaskNames))

			var flattenedLayers []string
			for _, layer := range layers {
				for _, task := range layer {
					flattenedLayers = append(flattenedLayers, task.Name)
				}
			}

			Expect(flattenedLayers).To(Equal(expectedTaskNames))
		})
		It("should successfully sort tasks topologically if graph is disconnected", func() {
			tasks := []landscape.TaskManifest{
				{
					Name: Task0,
					Type: "k8s",
				},
				{
					Name:      Task1,
					Type:      "k8s",
					DependsOn: []string{Task0},
				},
				{
					Name:      Task2,
					Type:      "k8s",
					DependsOn: []string{Task1},
				},
				{
					Name: Task3,
					Type: "k8s",
				},
				{
					Name:      Task4,
					Type:      "k8s",
					DependsOn: []string{Task3},
				},
				{
					Name:      Task5,
					Type:      "k8s",
					DependsOn: []string{Task4},
				},
			}

			result, layers, err := SortTasks(tasks)
			Expect(err).ToNot(HaveOccurred(), "Failed to sort task dependency graph")
			Expect(result).To(HaveLen(len(tasks)))

			var sortedTaskNames = make([]string, len(result), len(result))
			for i, task := range result {
				sortedTaskNames[i] = task.Name
			}

			expectedTaskNames := []string{Task0, Task3, Task1, Task4, Task2, Task5}
			Expect(sortedTaskNames).To(Equal(expectedTaskNames))

			var flattenedLayers []string
			for _, layer := range layers {
				for _, task := range layer {
					flattenedLayers = append(flattenedLayers, task.Name)
				}
			}

			Expect(flattenedLayers).To(Equal(expectedTaskNames))
		})
		It("should fail when graph does not have any root nodes (graph is cyclic)", func() {
			tasks := []landscape.TaskManifest{
				{
					Name:      Task0,
					Type:      "k8s",
					DependsOn: []string{Task2},
				},
				{
					Name:      Task1,
					Type:      "k8s",
					DependsOn: []string{Task0},
				},
				{
					Name:      Task2,
					Type:      "k8s",
					DependsOn: []string{Task1},
				},
			}

			_, _, err := SortTasks(tasks)
			Expect(err).To(MatchError("task dependency graph contains no root nodes"))
		})
		It("should fail when graph contains a cycle", func() {
			tasks := []landscape.TaskManifest{
				{
					Name:      Task0,
					Type:      "k8s",
					DependsOn: []string{Task3},
				},
				{
					Name: Task1,
					Type: "k8s",
				},
				{
					Name:      Task2,
					Type:      "k8s",
					DependsOn: []string{Task0, Task1},
				},
				{
					Name:      Task3,
					Type:      "k8s",
					DependsOn: []string{Task1, Task2},
				},
			}

			_, _, err := SortTasks(tasks)
			Expect(err).To(MatchError("task dependency graph contains a cycle"))
		})
		It("should fail when task list is nil", func() {
			_, _, err := SortTasks(nil)
			Expect(err).To(MatchError("task list is empty"))
		})
		It("should fail when task list is empty", func() {
			tasks := []landscape.TaskManifest{}
			_, _, err := SortTasks(tasks)
			Expect(err).To(MatchError("task list is empty"))
		})
	})
})
