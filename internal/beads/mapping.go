// Package beads integrates taskval with the Beads (bd) issue tracker.
// It maps validated task template fields to bd CLI commands for creating
// and linking issues.
package beads

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nixlim/task_templating/internal/validator"
)

// MapPriority maps a task template priority string to a bd numeric priority.
// Returns 2 (medium) as default for empty or unrecognized values.
func MapPriority(priority string) int {
	switch strings.ToLower(priority) {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 2
	}
}

// MapEstimate maps a task template estimate string to minutes.
// Returns 0 for "unknown" or empty string, signaling the estimate should be omitted.
func MapEstimate(estimate string) int {
	switch strings.ToLower(estimate) {
	case "trivial":
		return 15
	case "small":
		return 60
	case "medium":
		return 240
	case "large":
		return 480
	default:
		return 0
	}
}

// ComposeDescription builds a structured markdown description from task template
// fields for use with the bd --description flag. Sections with no data or N/A
// status are omitted.
func ComposeDescription(task *validator.TaskNode) string {
	var sb strings.Builder

	// Goal is always first.
	sb.WriteString(task.Goal)

	// Inputs section.
	if len(task.Inputs) > 0 {
		sb.WriteString("\n\n## Inputs\n")
		for _, in := range task.Inputs {
			sb.WriteString(fmt.Sprintf("- **%s** (`%s`): %s -- Source: %s\n", in.Name, in.Type, in.Constraints, in.Source))
		}
	}

	// Outputs section.
	if len(task.Outputs) > 0 {
		sb.WriteString("\n## Outputs\n")
		for _, out := range task.Outputs {
			sb.WriteString(fmt.Sprintf("- **%s** (`%s`): %s -- Dest: %s\n", out.Name, out.Type, out.Constraints, out.Destination))
		}
	}

	// Constraints section.
	constraints := parseStringArrayOrNA(task.Constraints)
	if len(constraints) > 0 {
		sb.WriteString("\n## Constraints\n")
		for _, c := range constraints {
			sb.WriteString(fmt.Sprintf("- %s\n", c))
		}
	}

	// Non-Goals section.
	if len(task.NonGoals) > 0 {
		sb.WriteString("\n## Non-Goals\n")
		for _, ng := range task.NonGoals {
			sb.WriteString(fmt.Sprintf("- %s\n", ng))
		}
	}

	// Error Cases section.
	if len(task.ErrorCases) > 0 {
		sb.WriteString("\n## Error Cases\n")
		for _, ec := range task.ErrorCases {
			sb.WriteString(fmt.Sprintf("- **%s**: %s -> %s\n", ec.Condition, ec.Behavior, ec.Output))
		}
	}

	return sb.String()
}

// templateMetadata is the structure stored in the bd --design field.
type templateMetadata struct {
	Template templateData `json:"_template"`
}

type templateData struct {
	Version    string                 `json:"version"`
	TaskID     string                 `json:"task_id"`
	FilesScope []string               `json:"files_scope"`
	Effects    string                 `json:"effects"`
	Inputs     []validator.InputSpec  `json:"inputs"`
	Outputs    []validator.OutputSpec `json:"outputs"`
}

// BuildTemplateMetadata builds a JSON string containing machine-readable
// template metadata for the bd --design flag.
func BuildTemplateMetadata(task *validator.TaskNode) (string, error) {
	filesScope := parseStringArrayOrNA(task.FilesScope)
	if filesScope == nil {
		filesScope = []string{}
	}

	effects := parseEffectsOrNA(task.Effects)

	meta := templateMetadata{
		Template: templateData{
			Version:    "0.2.0",
			TaskID:     task.TaskID,
			FilesScope: filesScope,
			Effects:    effects,
			Inputs:     task.Inputs,
			Outputs:    task.Outputs,
		},
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("marshaling template metadata: %w", err)
	}
	return string(data), nil
}

// FormatAcceptance joins acceptance criteria into a markdown checklist.
func FormatAcceptance(criteria []string) string {
	if len(criteria) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, c := range criteria {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("- " + c)
	}
	return sb.String()
}

// parseStringArrayOrNA attempts to parse a json.RawMessage as a string array.
// Returns nil if the field is nil, empty, or an N/A object.
func parseStringArrayOrNA(raw json.RawMessage) []string {
	if raw == nil {
		return nil
	}

	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr
	}

	// It's an N/A object or something else; skip it.
	return nil
}

// parseEffectsOrNA attempts to parse the effects field.
// Effects can be a string like "None", an array of EffectSpec objects, or N/A.
func parseEffectsOrNA(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}

	// Try as string first.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	// Try as array of effect specs.
	var effects []validator.EffectSpec
	if err := json.Unmarshal(raw, &effects); err == nil {
		parts := make([]string, len(effects))
		for i, e := range effects {
			parts[i] = fmt.Sprintf("%s: %s", e.Type, e.Target)
		}
		return strings.Join(parts, "; ")
	}

	// N/A or unrecognized.
	return ""
}
