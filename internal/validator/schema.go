package validator

import (
	"embed"
	"fmt"
	"strings"

	"github.com/kaptinlin/jsonschema"
)

//go:embed schemas/*.json
var embeddedSchemas embed.FS

// SchemaValidator performs Tier 1 structural validation using JSON Schema.
type SchemaValidator struct {
	taskNodeSchema  *jsonschema.Schema
	taskGraphSchema *jsonschema.Schema
}

// NewSchemaValidator creates a validator with the embedded JSON schemas.
func NewSchemaValidator() (*SchemaValidator, error) {
	c := jsonschema.NewCompiler()

	// Load and compile the task node schema.
	nodeData, err := embeddedSchemas.ReadFile("schemas/task_node.schema.json")
	if err != nil {
		return nil, fmt.Errorf("reading embedded task_node schema: %w", err)
	}

	nodeSchema, err := c.Compile(nodeData)
	if err != nil {
		return nil, fmt.Errorf("compiling task_node schema: %w", err)
	}

	// Load and compile the task graph schema.
	graphData, err := embeddedSchemas.ReadFile("schemas/task_graph.schema.json")
	if err != nil {
		return nil, fmt.Errorf("reading embedded task_graph schema: %w", err)
	}

	graphSchema, err := c.Compile(graphData)
	if err != nil {
		return nil, fmt.Errorf("compiling task_graph schema: %w", err)
	}

	return &SchemaValidator{
		taskNodeSchema:  nodeSchema,
		taskGraphSchema: graphSchema,
	}, nil
}

// ValidateTaskNode validates a single task node JSON against the schema.
func (sv *SchemaValidator) ValidateTaskNode(data []byte, result *ValidationResult) {
	schemaResult := sv.taskNodeSchema.Validate(data)
	if !schemaResult.IsValid() {
		convertSchemaErrors(schemaResult, result)
	}
}

// ValidateTaskGraph validates a task graph JSON against the schema.
func (sv *SchemaValidator) ValidateTaskGraph(data []byte, result *ValidationResult) {
	schemaResult := sv.taskGraphSchema.Validate(data)
	if !schemaResult.IsValid() {
		convertSchemaErrors(schemaResult, result)
	}
}

// convertSchemaErrors translates kaptinlin/jsonschema validation results
// into our LLM-friendly ValidationError format.
func convertSchemaErrors(schemaResult *jsonschema.EvaluationResult, result *ValidationResult) {
	// GetDetailedErrors returns map[fieldPath]errorMessage with all leaf errors.
	errors := schemaResult.GetDetailedErrors()
	for path, msg := range errors {
		if path == "" {
			path = "$"
		}

		suggestion := generateSchemaSuggestion(path, msg)

		result.AddError(ValidationError{
			Rule:       "SCHEMA",
			Severity:   SeverityError,
			Path:       path,
			Message:    msg,
			Suggestion: suggestion,
		})
	}
}

// generateSchemaSuggestion produces actionable fix advice based on the
// JSON path and error message.
func generateSchemaSuggestion(path, msg string) string {
	lowerMsg := strings.ToLower(msg)

	switch {
	case strings.Contains(lowerMsg, "required"):
		return fmt.Sprintf("Add the missing required field at '%s'. Check the spec's Quick Reference (Appendix A) for the expected format.", path)
	case strings.Contains(lowerMsg, "pattern"):
		if strings.Contains(path, "task_id") {
			return "task_id must be kebab-case (lowercase letters, numbers, hyphens). Example: 'my-task-name'. Pattern: ^[a-z0-9]+(-[a-z0-9]+)*$"
		}
		return fmt.Sprintf("The value at '%s' does not match the required pattern. Check the schema for the expected format.", path)
	case strings.Contains(lowerMsg, "enum") || strings.Contains(lowerMsg, "const"):
		return fmt.Sprintf("The value at '%s' must be one of the allowed values. Check the schema definition for valid options.", path)
	case strings.Contains(lowerMsg, "maxlength") || strings.Contains(lowerMsg, "maximum"):
		return fmt.Sprintf("The value at '%s' exceeds the maximum length. Shorten it.", path)
	case strings.Contains(lowerMsg, "minlength") || strings.Contains(lowerMsg, "minimum"):
		return fmt.Sprintf("The value at '%s' is too short or empty. Provide a meaningful value.", path)
	case strings.Contains(lowerMsg, "minitems"):
		return fmt.Sprintf("The array at '%s' must have at least one item. Add the required elements.", path)
	case strings.Contains(lowerMsg, "additional"):
		return fmt.Sprintf("The field at '%s' is not recognized. Remove it or check for typos. Valid fields are listed in the schema.", path)
	case strings.Contains(lowerMsg, "type"):
		return fmt.Sprintf("The value at '%s' has the wrong type. Check the schema for the expected type (string, array, object, etc.).", path)
	case strings.Contains(lowerMsg, "oneof"):
		return fmt.Sprintf("The value at '%s' must match exactly one of the allowed schemas. Check the spec for valid formats.", path)
	default:
		return ""
	}
}
