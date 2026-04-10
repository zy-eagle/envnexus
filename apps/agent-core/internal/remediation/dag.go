package remediation

import "fmt"

// DAG represents a directed acyclic graph of remediation steps.
type DAG struct {
	nodes    map[int]*RemediationStep
	children map[int][]int
	parents  map[int][]int
}

// BuildDAG constructs a DAG from the steps in a remediation plan.
// Returns an error if a cycle is detected or a dependency references a non-existent step.
func BuildDAG(steps []RemediationStep) (*DAG, error) {
	d := &DAG{
		nodes:    make(map[int]*RemediationStep, len(steps)),
		children: make(map[int][]int),
		parents:  make(map[int][]int),
	}

	for i := range steps {
		d.nodes[steps[i].StepID] = &steps[i]
	}

	for _, step := range steps {
		for _, dep := range step.DependsOn {
			if _, ok := d.nodes[dep]; !ok {
				return nil, fmt.Errorf("step %d depends on non-existent step %d", step.StepID, dep)
			}
			d.children[dep] = append(d.children[dep], step.StepID)
			d.parents[step.StepID] = append(d.parents[step.StepID], dep)
		}
	}

	if _, err := d.TopologicalSort(); err != nil {
		return nil, err
	}

	return d, nil
}

// TopologicalSort returns steps grouped by execution layer (Kahn's algorithm).
// Steps within the same layer have no dependencies on each other and can run in parallel.
func (d *DAG) TopologicalSort() ([][]int, error) {
	inDegree := make(map[int]int, len(d.nodes))
	for id := range d.nodes {
		inDegree[id] = len(d.parents[id])
	}

	var queue []int
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var layers [][]int
	visited := 0

	for len(queue) > 0 {
		layers = append(layers, queue)
		visited += len(queue)

		var next []int
		for _, id := range queue {
			for _, child := range d.children[id] {
				inDegree[child]--
				if inDegree[child] == 0 {
					next = append(next, child)
				}
			}
		}
		queue = next
	}

	if visited != len(d.nodes) {
		return nil, fmt.Errorf("cycle detected in remediation plan DAG: visited %d of %d nodes", visited, len(d.nodes))
	}

	return layers, nil
}

// ExecutionOrder returns a flat list of step IDs in topological order.
func (d *DAG) ExecutionOrder() ([]int, error) {
	layers, err := d.TopologicalSort()
	if err != nil {
		return nil, err
	}
	var order []int
	for _, layer := range layers {
		order = append(order, layer...)
	}
	return order, nil
}

// GetStep returns the step with the given ID.
func (d *DAG) GetStep(id int) (*RemediationStep, bool) {
	s, ok := d.nodes[id]
	return s, ok
}

// ReverseOrder returns step IDs in reverse topological order (for rollback).
func (d *DAG) ReverseOrder() ([]int, error) {
	order, err := d.ExecutionOrder()
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
		order[i], order[j] = order[j], order[i]
	}
	return order, nil
}
