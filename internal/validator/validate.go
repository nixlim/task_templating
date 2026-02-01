package validator

import (
	"encoding/json"
	"fmt"
)

// Mode indicates whether we're validating a single task or a full graph.
type Mode int

const (
	ModeSingleTask Mode = iota
	ModeTaskGraph
)

// Validate performs full validation (Tier 1 + Tier 2) on input JSON data.
// Returns a ValidationResult with all findings.
func Validate(data []byte, mode Mode) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	// Tier 1: JSON Schema validation.
	sv, err := NewSchemaValidator()
	if err != nil {
		return nil, fmt.Errorf("initializing schema validator: %w", err)
	}

	switch mode {
	case ModeSingleTask:
		sv.ValidateTaskNode(data, result)

		// If schema validation passed, proceed to Tier 2.
		if result.Valid {
			// Wrap single task in a graph for semantic validation.
			var task TaskNode
			if err := json.Unmarshal(data, &task); err != nil {
				return nil, fmt.Errorf("parsing task node: %w", err)
			}
			graph := &TaskGraph{
				Version: "0.1.0",
				Tasks:   []TaskNode{task},
			}
			sem := NewSemanticValidator()
			sem.ValidateTaskGraph(graph, result)
			if result.Valid {
				result.Graph = graph
			}
		}

	case ModeTaskGraph:
		sv.ValidateTaskGraph(data, result)

		// If schema validation passed, proceed to Tier 2.
		if result.Valid {
			var graph TaskGraph
			if err := json.Unmarshal(data, &graph); err != nil {
				return nil, fmt.Errorf("parsing task graph: %w", err)
			}
			sem := NewSemanticValidator()
			sem.ValidateTaskGraph(&graph, result)
			if result.Valid {
				result.Graph = &graph
			}
		}

	default:
		return nil, fmt.Errorf("unknown validation mode: %d", mode)
	}

	return result, nil
}
