// Package validator provides structural (JSON Schema) and semantic validation
// for task nodes and task graphs conforming to the Structured Task Template Spec.
package validator

import "fmt"

// Severity classifies how critical a validation finding is.
type Severity string

const (
	SeverityError   Severity = "ERROR"
	SeverityWarning Severity = "WARNING"
	SeverityInfo    Severity = "INFO"
)

// ValidationError represents a single validation finding with enough context
// for an LLM agent to understand exactly what needs to be fixed.
type ValidationError struct {
	// Rule is the validation rule ID (e.g., "V1", "V5", "SCHEMA").
	Rule string `json:"rule"`

	// Severity indicates if this is a blocking error, warning, or info.
	Severity Severity `json:"severity"`

	// Path is the JSON path to the problematic field (e.g., "tasks[0].goal").
	Path string `json:"path"`

	// Message is a human/LLM-readable description of the problem.
	Message string `json:"message"`

	// Suggestion is an actionable fix recommendation.
	Suggestion string `json:"suggestion,omitempty"`

	// Context provides the actual value that caused the error, if applicable.
	Context string `json:"context,omitempty"`
}

// Error implements the error interface.
func (ve ValidationError) Error() string {
	s := fmt.Sprintf("[%s] %s at '%s': %s", ve.Severity, ve.Rule, ve.Path, ve.Message)
	if ve.Suggestion != "" {
		s += fmt.Sprintf(" -> Fix: %s", ve.Suggestion)
	}
	return s
}

// ValidationResult aggregates all findings from a validation run.
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
	Stats  ValidationStats   `json:"stats"`
	Graph  *TaskGraph        `json:"-"` // Parsed graph, not included in JSON output
}

// ValidationStats provides summary counts.
type ValidationStats struct {
	TotalTasks   int `json:"total_tasks"`
	ErrorCount   int `json:"error_count"`
	WarningCount int `json:"warning_count"`
	InfoCount    int `json:"info_count"`
}

// AddError appends a validation error and updates stats.
func (vr *ValidationResult) AddError(ve ValidationError) {
	vr.Errors = append(vr.Errors, ve)
	switch ve.Severity {
	case SeverityError:
		vr.Stats.ErrorCount++
		vr.Valid = false
	case SeverityWarning:
		vr.Stats.WarningCount++
	case SeverityInfo:
		vr.Stats.InfoCount++
	}
}
