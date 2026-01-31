package validator

import (
	"encoding/json"
	"fmt"
)

// TaskGraph represents the top-level task graph document.
type TaskGraph struct {
	Version    string                       `json:"version"`
	Types      map[string]map[string]string `json:"types,omitempty"`
	Defaults   *Defaults                    `json:"defaults,omitempty"`
	Milestones []Milestone                  `json:"milestones,omitempty"`
	Tasks      []TaskNode                   `json:"tasks"`
}

// Defaults represents inheritable default field values.
type Defaults struct {
	Constraints []string `json:"constraints,omitempty"`
	Acceptance  []string `json:"acceptance,omitempty"`
	NonGoals    []string `json:"non_goals,omitempty"`
}

// Milestone represents a named grouping of tasks.
type Milestone struct {
	Name                string   `json:"name"`
	DependsOnMilestones []string `json:"depends_on_milestones,omitempty"`
	TaskIDs             []string `json:"task_ids"`
}

// TaskNode represents a single task in the graph.
type TaskNode struct {
	TaskID      string          `json:"task_id"`
	TaskName    string          `json:"task_name"`
	Goal        string          `json:"goal"`
	Inputs      []InputSpec     `json:"inputs"`
	Outputs     []OutputSpec    `json:"outputs"`
	Acceptance  []string        `json:"acceptance"`
	DependsOn   json.RawMessage `json:"depends_on,omitempty"`
	Constraints json.RawMessage `json:"constraints,omitempty"`
	FilesScope  json.RawMessage `json:"files_scope,omitempty"`
	NonGoals    []string        `json:"non_goals,omitempty"`
	Effects     json.RawMessage `json:"effects,omitempty"`
	ErrorCases  []ErrorSpec     `json:"error_cases,omitempty"`
	Priority    string          `json:"priority,omitempty"`
	Estimate    string          `json:"estimate,omitempty"`
	Notes       string          `json:"notes,omitempty"`
}

// InputSpec represents a single input the task requires.
type InputSpec struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Constraints string `json:"constraints"`
	Source      string `json:"source"`
}

// OutputSpec represents a single output the task produces.
type OutputSpec struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Constraints string `json:"constraints"`
	Destination string `json:"destination"`
}

// EffectSpec represents a declared side effect.
type EffectSpec struct {
	Type   string `json:"type"`
	Target string `json:"target"`
}

// ErrorSpec represents an expected failure mode.
type ErrorSpec struct {
	Condition string `json:"condition"`
	Behavior  string `json:"behavior"`
	Output    string `json:"output"`
}

// NotApplicable represents an explicit N/A for contextual fields.
type NotApplicable struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// ParseDependsOn extracts the depends_on field which can be either
// a list of task IDs or a NotApplicable object.
func (t *TaskNode) ParseDependsOn() (taskIDs []string, na *NotApplicable, err error) {
	if t.DependsOn == nil {
		return nil, nil, nil
	}

	// Try as array of strings first.
	var ids []string
	if err := json.Unmarshal(t.DependsOn, &ids); err == nil {
		return ids, nil, nil
	}

	// Try as NotApplicable object.
	var notAppl NotApplicable
	if err := json.Unmarshal(t.DependsOn, &notAppl); err == nil {
		if notAppl.Status == "N/A" {
			return nil, &notAppl, nil
		}
	}

	return nil, nil, fmt.Errorf("depends_on must be either an array of task IDs or {\"status\": \"N/A\", \"reason\": \"...\"}, got: %s", string(t.DependsOn))
}

// ParseFilesScope extracts the files_scope field which can be either
// a list of file paths or a NotApplicable object.
func (t *TaskNode) ParseFilesScope() (files []string, na *NotApplicable, err error) {
	if t.FilesScope == nil {
		return nil, nil, nil
	}

	var paths []string
	if err := json.Unmarshal(t.FilesScope, &paths); err == nil {
		return paths, nil, nil
	}

	var notAppl NotApplicable
	if err := json.Unmarshal(t.FilesScope, &notAppl); err == nil {
		if notAppl.Status == "N/A" {
			return nil, &notAppl, nil
		}
	}

	return nil, nil, fmt.Errorf("files_scope must be either an array of file paths or {\"status\": \"N/A\", \"reason\": \"...\"}, got: %s", string(t.FilesScope))
}
