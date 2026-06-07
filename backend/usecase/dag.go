package usecase

import (
	"errors"
	"fmt"
)

type StepType string

const (
	StepHTTP   StepType = "http"
	StepScript StepType = "script"
	StepDelay  StepType = "delay"
	StepBranch StepType = "branch"
)

type RetryPolicy struct {
	MaxRetries        int     `json:"max_retries"`
	InitialIntervalMS int     `json:"initial_interval_ms"`
	BackoffFactor     float64 `json:"backoff_factor"`
}

type Step struct {
	ID          string      `json:"id"`
	Type        StepType    `json:"type"`
	DependsOn   []string    `json:"depends_on"`
	Config      interface{} `json:"config"`
	RetryPolicy RetryPolicy `json:"retry_policy"`
}

type WorkflowDefinition struct {
	Steps []Step `json:"steps"`
}

func ValidateAndSort(def WorkflowDefinition) ([]string, error) {
	if len(def.Steps) == 0 {
		return nil, errors.New("workflow definition must contain at least one step")
	}

	inDegree := make(map[string]int)
	adjList := make(map[string][]string)
	exists := make(map[string]bool)

	for _, step := range def.Steps {
		if step.ID == "" {
			return nil, errors.New("step ID cannot be empty")
		}
		if exists[step.ID] {
			return nil, fmt.Errorf("duplicate step ID found: %s", step.ID)
		}
		exists[step.ID] = true
		inDegree[step.ID] = 0
	}

	for _, step := range def.Steps {
		for _, dep := range step.DependsOn {
			if !exists[dep] {
				return nil, fmt.Errorf("step %s depends on non-existent step %s", step.ID, dep)
			}
			adjList[dep] = append(adjList[dep], step.ID)
			inDegree[step.ID]++
		}
	}

	var queue []string
	for _, step := range def.Steps {
		if inDegree[step.ID] == 0 {
			queue = append(queue, step.ID)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		sorted = append(sorted, curr)

		for _, neighbor := range adjList[curr] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(sorted) != len(def.Steps) {
		return nil, errors.New("circular dependency detected in workflow steps")
	}

	return sorted, nil
}
