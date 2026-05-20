//nolint:staticcheck // ST1001: allow dot-import for test utils using Gomega
package graph

import (
	"errors"

	landscape "github.com/konfidence-project/konfidence/api/star/v1alpha1"
)

func createAdjacencyList(tasks []landscape.TaskManifest) map[string][]landscape.TaskManifest {
	adjacencyList := map[string][]landscape.TaskManifest{}

	// TODO validate that dependsOn only contains valid task names
	for _, task := range tasks {
		if _, ok := adjacencyList[task.Name]; !ok {
			adjacencyList[task.Name] = nil
		}

		for _, dependency := range task.DependsOn {
			adjacencyList[dependency] = append(adjacencyList[dependency], task)
		}
	}

	return adjacencyList
}

// SortTasks implements topological sort of the task dependency graph with Kahn's algorithm see https://en.wikipedia.org/wiki/Topological_sorting
func SortTasks(tasks []landscape.TaskManifest) ([]landscape.TaskManifest, [][]landscape.TaskManifest, error) {
	if len(tasks) == 0 {
		return nil, nil, errors.New("task list is empty")
	}

	// create adjacency list
	adjacencyList := createAdjacencyList(tasks)

	var queue []landscape.TaskManifest // work queue for the algorithm
	indegreeCounts := map[string]int{} // number of incoming edges per node
	layerCounts := map[string]int{}    // marks in which layer a node becomes a leaf

	// compute indegree for each node
	for _, task := range tasks {
		dependencyCount := len(task.DependsOn)
		indegreeCounts[task.Name] = dependencyCount

		// add root nodes directly to the queue
		if dependencyCount < 1 {
			queue = append(queue, task)
			layerCounts[task.Name] = 0 // initial root nodes without any dependency are in layer 0
		}
	}

	if len(queue) < 1 {
		return nil, nil, errors.New("task dependency graph contains no root nodes")
	}

	layerMap := map[int][]landscape.TaskManifest{0: queue}
	var result []landscape.TaskManifest

	for len(queue) > 0 {
		// TODO maybe we could use a more efficient queue implementation here
		task := queue[0]
		queue = queue[1:]
		result = append(result, task)

		if len(adjacencyList[task.Name]) > 0 {
			for _, dependentTask := range adjacencyList[task.Name] {
				indegreeCounts[dependentTask.Name]--

				if indegreeCounts[dependentTask.Name] == 0 {
					queue = append(queue, dependentTask)
					layer := layerCounts[task.Name] + 1
					layerCounts[dependentTask.Name] = layer
					layerMap[layer] = append(layerMap[layer], dependentTask)
				}
			}
		}
	}

	if len(result) != len(tasks) {
		return nil, nil, errors.New("task dependency graph contains a cycle")
	}

	// copy layers in ordered list
	layers := make([][]landscape.TaskManifest, len(layerMap))
	for i, layer := range layerMap {
		layers[i] = layer
	}

	return result, layers, nil
}
