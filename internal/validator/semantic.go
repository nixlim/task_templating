package validator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Forbidden words in GOAL field per spec Section 3.1.
var goalForbiddenWords = []string{"try", "explore", "investigate", "look into"}

// goalForbiddenPattern matches forbidden words as whole words (case-insensitive).
var goalForbiddenPatterns []*regexp.Regexp

// Weasel words/phrases that signal deferred or vague scope (V11).
var weaselWords = []string{
	"v1",
	"v2",
	"simplified version",
	"static for now",
	"hardcoded for now",
	"future enhancement",
	"placeholder",
	"basic version",
	"minimal implementation",
	"will be wired later",
	"dynamic in future phase",
	"skip for now",
}

// weaselWordPatterns matches weasel words as whole words/phrases (case-insensitive).
var weaselWordPatterns []*regexp.Regexp

func init() {
	for _, w := range goalForbiddenWords {
		// Use word boundaries. "look into" is a phrase, handle specially.
		pattern := `(?i)\b` + regexp.QuoteMeta(w) + `\b`
		goalForbiddenPatterns = append(goalForbiddenPatterns, regexp.MustCompile(pattern))
	}
	for _, w := range weaselWords {
		escaped := regexp.QuoteMeta(w)
		var pattern string
		if w == "v1" || w == "v2" {
			// Avoid matching semver (v1.2.3) or kebab suffixes (extract-records-v2).
			pattern = `(?i)(?:^|[\s,;(])` + escaped + `(?:[\s,;)]|$)`
		} else {
			pattern = `(?i)\b` + escaped + `\b`
		}
		weaselWordPatterns = append(weaselWordPatterns, regexp.MustCompile(pattern))
	}
}

// SemanticValidator performs Tier 2 validation: checks that require
// cross-node analysis or semantic understanding beyond JSON Schema.
type SemanticValidator struct{}

// NewSemanticValidator creates a new semantic validator.
func NewSemanticValidator() *SemanticValidator {
	return &SemanticValidator{}
}

// ValidateTaskGraph performs all semantic checks on a parsed task graph.
func (sv *SemanticValidator) ValidateTaskGraph(graph *TaskGraph, result *ValidationResult) {
	result.Stats.TotalTasks = len(graph.Tasks)

	// Build lookup for fast access.
	taskIndex := make(map[string]int, len(graph.Tasks))
	for i, t := range graph.Tasks {
		taskIndex[t.TaskID] = i
	}

	// V2: Unique TASK_IDs.
	sv.checkUniqueTaskIDs(graph, result)

	// V4: DEPENDS_ON reference integrity.
	sv.checkDependencyReferences(graph, taskIndex, result)

	// V5: DAG acyclicity.
	sv.checkDAGAcyclicity(graph, taskIndex, result)

	// V6: GOAL quality.
	sv.checkGoalQuality(graph, result)

	// V7: ACCEPTANCE quality.
	sv.checkAcceptanceQuality(graph, result)

	// V9: Contextual fields are present or N/A.
	sv.checkContextualFields(graph, result)

	// V10: FILES_SCOPE non-empty for implementation tasks.
	sv.checkFilesScope(graph, result)

	// Milestone checks.
	sv.checkMilestones(graph, taskIndex, result)

	// V11: Weasel words.
	sv.checkWeaselWords(graph, result)

	// V12: Cross-task contracts.
	sv.checkCrossTaskContracts(graph, result)

	// V13: Granularity heuristics.
	sv.checkGranularity(graph, result)

	// V14: Missing dependency links.
	sv.checkMissingDependencyLinks(graph, taskIndex, result)
}

// checkUniqueTaskIDs ensures no duplicate TASK_IDs exist (V2).
func (sv *SemanticValidator) checkUniqueTaskIDs(graph *TaskGraph, result *ValidationResult) {
	seen := make(map[string]int)
	for i, t := range graph.Tasks {
		if prev, exists := seen[t.TaskID]; exists {
			result.AddError(ValidationError{
				Rule:       "V2",
				Severity:   SeverityError,
				Path:       fmt.Sprintf("tasks[%d].task_id", i),
				Message:    fmt.Sprintf("Duplicate task_id '%s' — first occurrence at tasks[%d].", t.TaskID, prev),
				Suggestion: "Every task_id must be globally unique within the project. Rename one of the duplicates.",
				Context:    t.TaskID,
			})
		}
		seen[t.TaskID] = i
	}
}

// checkDependencyReferences ensures all DEPENDS_ON references resolve (V4).
func (sv *SemanticValidator) checkDependencyReferences(graph *TaskGraph, taskIndex map[string]int, result *ValidationResult) {
	for i, t := range graph.Tasks {
		deps, _, err := t.ParseDependsOn()
		if err != nil {
			result.AddError(ValidationError{
				Rule:       "V4",
				Severity:   SeverityError,
				Path:       fmt.Sprintf("tasks[%d].depends_on", i),
				Message:    err.Error(),
				Suggestion: "depends_on must be an array of task_id strings or {\"status\": \"N/A\", \"reason\": \"...\"}.",
			})
			continue
		}

		for _, dep := range deps {
			if _, exists := taskIndex[dep]; !exists {
				result.AddError(ValidationError{
					Rule:     "V4",
					Severity: SeverityError,
					Path:     fmt.Sprintf("tasks[%d].depends_on", i),
					Message: fmt.Sprintf(
						"Task '%s' depends on '%s', but no task with that task_id exists in the graph.",
						t.TaskID, dep,
					),
					Suggestion: fmt.Sprintf(
						"Either add a task with task_id '%s' to the graph, or remove '%s' from the depends_on list of task '%s'.",
						dep, dep, t.TaskID,
					),
					Context: dep,
				})
			}

			// Self-dependency check.
			if dep == t.TaskID {
				result.AddError(ValidationError{
					Rule:       "V5",
					Severity:   SeverityError,
					Path:       fmt.Sprintf("tasks[%d].depends_on", i),
					Message:    fmt.Sprintf("Task '%s' depends on itself — this creates a trivial cycle.", t.TaskID),
					Suggestion: "Remove the self-reference from depends_on.",
					Context:    dep,
				})
			}
		}
	}
}

// checkDAGAcyclicity detects cycles in the dependency graph (V5).
// Uses Kahn's algorithm (topological sort via in-degree counting).
func (sv *SemanticValidator) checkDAGAcyclicity(graph *TaskGraph, taskIndex map[string]int, result *ValidationResult) {
	// Build adjacency list.
	adj := make(map[string][]string) // task -> tasks that depend on it
	inDegree := make(map[string]int) // task -> number of dependencies

	for _, t := range graph.Tasks {
		if _, exists := inDegree[t.TaskID]; !exists {
			inDegree[t.TaskID] = 0
		}
		if _, exists := adj[t.TaskID]; !exists {
			adj[t.TaskID] = nil
		}
	}

	for _, t := range graph.Tasks {
		deps, _, err := t.ParseDependsOn()
		if err != nil {
			continue // Already reported in reference check.
		}
		for _, dep := range deps {
			if _, exists := taskIndex[dep]; !exists {
				continue // Already reported in reference check.
			}
			adj[dep] = append(adj[dep], t.TaskID)
			inDegree[t.TaskID]++
		}
	}

	// Kahn's algorithm.
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	visited := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		visited++

		for _, neighbor := range adj[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if visited < len(graph.Tasks) {
		// Find which tasks are in the cycle(s).
		var cycleMembers []string
		for id, deg := range inDegree {
			if deg > 0 {
				cycleMembers = append(cycleMembers, id)
			}
		}

		result.AddError(ValidationError{
			Rule:     "V5",
			Severity: SeverityError,
			Path:     "tasks",
			Message: fmt.Sprintf(
				"Dependency graph contains a cycle. %d task(s) are involved: [%s]. A valid task graph must be a DAG (Directed Acyclic Graph).",
				len(cycleMembers), strings.Join(cycleMembers, ", "),
			),
			Suggestion: "Review the depends_on fields of the listed tasks. Break the cycle by removing one dependency or decomposing a task into sub-tasks.",
			Context:    strings.Join(cycleMembers, ", "),
		})
	}
}

// checkGoalQuality ensures GOAL fields meet spec requirements (V6).
func (sv *SemanticValidator) checkGoalQuality(graph *TaskGraph, result *ValidationResult) {
	for i, t := range graph.Tasks {
		for j, pattern := range goalForbiddenPatterns {
			if pattern.MatchString(t.Goal) {
				result.AddError(ValidationError{
					Rule:     "V6",
					Severity: SeverityError,
					Path:     fmt.Sprintf("tasks[%d].goal", i),
					Message: fmt.Sprintf(
						"Goal contains the forbidden word/phrase '%s'. Goals must describe testable outcomes, not activities or explorations.",
						goalForbiddenWords[j],
					),
					Suggestion: fmt.Sprintf(
						"Rewrite the goal as a concrete, testable outcome. Instead of '%s ...', describe what the system does when the task is complete. Example: 'The function returns X when given Y.'",
						goalForbiddenWords[j],
					),
					Context: t.Goal,
				})
			}
		}

		// Check goal is phrased as outcome (heuristic: should not start with "To " which indicates activity).
		if strings.HasPrefix(strings.TrimSpace(t.Goal), "To ") {
			result.AddError(ValidationError{
				Rule:       "V6",
				Severity:   SeverityWarning,
				Path:       fmt.Sprintf("tasks[%d].goal", i),
				Message:    "Goal starts with 'To ...' which suggests an activity rather than a testable outcome.",
				Suggestion: "Rewrite as a state-of-the-world assertion. Example: Instead of 'To add search functionality', write 'The Search() function returns ranked results from Weaviate hybrid search.'",
				Context:    t.Goal,
			})
		}
	}
}

// checkAcceptanceQuality validates ACCEPTANCE criteria quality (V7).
func (sv *SemanticValidator) checkAcceptanceQuality(graph *TaskGraph, result *ValidationResult) {
	// Vague phrases that indicate non-verifiable criteria.
	vaguePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(works? correctly)\b`),
		regexp.MustCompile(`(?i)\b(is correct)\b`),
		regexp.MustCompile(`(?i)\b(is good)\b`),
		regexp.MustCompile(`(?i)\b(looks? right)\b`),
		regexp.MustCompile(`(?i)\b(properly)\b`),
		regexp.MustCompile(`(?i)\b(as expected)\b`),
		regexp.MustCompile(`(?i)\b(should work)\b`),
		regexp.MustCompile(`(?i)\b(is fine)\b`),
	}

	vagueNames := []string{
		"works correctly", "is correct", "is good", "looks right",
		"properly", "as expected", "should work", "is fine",
	}

	for i, t := range graph.Tasks {
		for j, criterion := range t.Acceptance {
			for k, pattern := range vaguePatterns {
				if pattern.MatchString(criterion) {
					result.AddError(ValidationError{
						Rule:     "V7",
						Severity: SeverityWarning,
						Path:     fmt.Sprintf("tasks[%d].acceptance[%d]", i, j),
						Message: fmt.Sprintf(
							"Acceptance criterion contains the vague phrase '%s'. Criteria must be independently verifiable with concrete expected values.",
							vagueNames[k],
						),
						Suggestion: "Replace with a specific assertion. Example: Instead of 'it works correctly', write 'Given input \"test\", the function returns [\"result1\", \"result2\"] with status 200.'",
						Context:    criterion,
					})
				}
			}
		}
	}
}

// checkContextualFields ensures contextual fields are present or explicitly N/A (V9).
func (sv *SemanticValidator) checkContextualFields(graph *TaskGraph, result *ValidationResult) {
	contextualFields := []string{"depends_on", "constraints", "files_scope"}

	for i, t := range graph.Tasks {
		for _, field := range contextualFields {
			var raw json.RawMessage
			switch field {
			case "depends_on":
				raw = t.DependsOn
			case "constraints":
				raw = t.Constraints
			case "files_scope":
				raw = t.FilesScope
			}

			if raw == nil {
				result.AddError(ValidationError{
					Rule:     "V9",
					Severity: SeverityWarning,
					Path:     fmt.Sprintf("tasks[%d].%s", i, field),
					Message: fmt.Sprintf(
						"Contextual field '%s' is missing from task '%s'. Contextual fields should be explicitly present or set to {\"status\": \"N/A\", \"reason\": \"...\"}.",
						field, t.TaskID,
					),
					Suggestion: fmt.Sprintf(
						"Either provide a value for '%s' or explicitly mark it as not applicable: {\"status\": \"N/A\", \"reason\": \"your justification here\"}.",
						field,
					),
				})
			}
		}
	}
}

// checkFilesScope warns if FILES_SCOPE is empty for implementation tasks (V10).
func (sv *SemanticValidator) checkFilesScope(graph *TaskGraph, result *ValidationResult) {
	// Heuristic: tasks with verbs like "Implement", "Add", "Fix" in task_name
	// are likely implementation tasks.
	implVerbs := []string{"implement", "add", "fix", "create", "build", "write"}

	for i, t := range graph.Tasks {
		nameLower := strings.ToLower(t.TaskName)
		isImplTask := false
		for _, verb := range implVerbs {
			if strings.HasPrefix(nameLower, verb) {
				isImplTask = true
				break
			}
		}

		if !isImplTask {
			continue
		}

		files, na, err := t.ParseFilesScope()
		if err != nil {
			continue // Already reported elsewhere.
		}
		if files == nil && na == nil {
			result.AddError(ValidationError{
				Rule:     "V10",
				Severity: SeverityWarning,
				Path:     fmt.Sprintf("tasks[%d].files_scope", i),
				Message: fmt.Sprintf(
					"Task '%s' appears to be an implementation task (name starts with an implementation verb) but has no files_scope defined.",
					t.TaskID,
				),
				Suggestion: "Add a files_scope listing the files the agent should create or modify. This prevents unintended changes to other parts of the codebase.",
			})
		}
	}
}

// checkMilestones validates milestone definitions.
func (sv *SemanticValidator) checkMilestones(graph *TaskGraph, taskIndex map[string]int, result *ValidationResult) {
	if graph.Milestones == nil {
		return
	}

	milestoneIndex := make(map[string]int)
	for i, m := range graph.Milestones {
		// Check for duplicate milestone names.
		if prev, exists := milestoneIndex[m.Name]; exists {
			result.AddError(ValidationError{
				Rule:       "MILESTONE",
				Severity:   SeverityError,
				Path:       fmt.Sprintf("milestones[%d].name", i),
				Message:    fmt.Sprintf("Duplicate milestone name '%s' — first occurrence at milestones[%d].", m.Name, prev),
				Suggestion: "Every milestone name must be unique. Rename one of the duplicates.",
			})
		}
		milestoneIndex[m.Name] = i

		// Check that all task_ids in milestone exist.
		for _, tid := range m.TaskIDs {
			if _, exists := taskIndex[tid]; !exists {
				result.AddError(ValidationError{
					Rule:     "MILESTONE",
					Severity: SeverityError,
					Path:     fmt.Sprintf("milestones[%d].task_ids", i),
					Message: fmt.Sprintf(
						"Milestone '%s' references task_id '%s', but no task with that ID exists in the graph.",
						m.Name, tid,
					),
					Suggestion: fmt.Sprintf("Add a task with task_id '%s' or remove it from the milestone.", tid),
				})
			}
		}
	}

	// Check milestone dependency references.
	for i, m := range graph.Milestones {
		for _, dep := range m.DependsOnMilestones {
			if _, exists := milestoneIndex[dep]; !exists {
				result.AddError(ValidationError{
					Rule:     "MILESTONE",
					Severity: SeverityError,
					Path:     fmt.Sprintf("milestones[%d].depends_on_milestones", i),
					Message: fmt.Sprintf(
						"Milestone '%s' depends on milestone '%s', but no milestone with that name exists.",
						m.Name, dep,
					),
					Suggestion: fmt.Sprintf("Add a milestone named '%s' or remove it from depends_on_milestones.", dep),
				})
			}
		}
	}
}

// checkWeaselWords flags deferral / vague-scope language in goals and acceptance criteria (V11).
func (sv *SemanticValidator) checkWeaselWords(graph *TaskGraph, result *ValidationResult) {
	for i, t := range graph.Tasks {
		for j, pattern := range weaselWordPatterns {
			if pattern.MatchString(t.Goal) {
				result.AddError(ValidationError{
					Rule:     "V11",
					Severity: SeverityWarning,
					Path:     fmt.Sprintf("tasks[%d].goal", i),
					Message: fmt.Sprintf(
						"Goal contains the weasel word/phrase '%s', which signals deferred or unspecified scope.",
						weaselWords[j],
					),
					Suggestion: "State the goal as a concrete, testable outcome for the version under construction. If the behavior is genuinely out of scope, list it under non_goals; do not leave deferral language in the goal.",
					Context:    t.Goal,
				})
			}
		}

		for j, criterion := range t.Acceptance {
			for k, pattern := range weaselWordPatterns {
				if pattern.MatchString(criterion) {
					result.AddError(ValidationError{
						Rule:     "V11",
						Severity: SeverityWarning,
						Path:     fmt.Sprintf("tasks[%d].acceptance[%d]", i, j),
						Message: fmt.Sprintf(
							"Acceptance criterion contains the weasel word/phrase '%s', which makes the criterion unverifiable now.",
							weaselWords[k],
						),
						Suggestion: "Replace with a concrete, verifiable assertion (specific inputs, expected outputs). If the behavior is being deferred to a later task, move it there or capture it under non_goals — do not leave deferral language in acceptance criteria.",
						Context:    criterion,
					})
				}
			}
		}
	}
}

// checkCrossTaskContracts flags type mismatches when a task input sources from a
// dependency's declared output (V12).
func (sv *SemanticValidator) checkCrossTaskContracts(graph *TaskGraph, result *ValidationResult) {
	// Build outputIndex: taskID -> outputName -> OutputSpec.
	outputIndex := make(map[string]map[string]OutputSpec, len(graph.Tasks))
	outputPatterns := make(map[string]map[string]*regexp.Regexp, len(graph.Tasks))
	for _, t := range graph.Tasks {
		outs := make(map[string]OutputSpec, len(t.Outputs))
		pats := make(map[string]*regexp.Regexp, len(t.Outputs))
		for _, o := range t.Outputs {
			outs[o.Name] = o
			if o.Name != "" {
				pats[o.Name] = regexp.MustCompile(`\b` + regexp.QuoteMeta(o.Name) + `\b`)
			}
		}
		outputIndex[t.TaskID] = outs
		outputPatterns[t.TaskID] = pats
	}

	for i, t := range graph.Tasks {
		deps, _, err := t.ParseDependsOn()
		if err != nil || len(deps) == 0 {
			continue
		}
		for j, in := range t.Inputs {
			if in.Source == "" {
				continue
			}
			for _, dep := range deps {
				depOutputs, ok := outputIndex[dep]
				if !ok {
					continue
				}
				depPats := outputPatterns[dep]
				for outName, outSpec := range depOutputs {
					if outName == "" {
						continue
					}
					matched := depPats[outName].MatchString(in.Source)
					if !matched {
						continue
					}
					if !typesCompatible(in.Type, outSpec.Type) {
						result.AddError(ValidationError{
							Rule:     "V12",
							Severity: SeverityWarning,
							Path:     fmt.Sprintf("tasks[%d].inputs[%d]", i, j),
							Message: fmt.Sprintf(
								"Input '%s' (type '%s') of task '%s' sources from output '%s' of task '%s' (type '%s'), but the types are not compatible.",
								in.Name, in.Type, t.TaskID, outName, dep, outSpec.Type,
							),
							Suggestion: fmt.Sprintf(
								"Align the type contract: change input.type on '%s.%s' to '%s', or change output.type on '%s.%s' to '%s'. Compatible types are exact matches or one wrapping the other in optional<...>.",
								t.TaskID, in.Name, outSpec.Type, dep, outName, in.Type,
							),
							Context: in.Source,
						})
					}
				}
			}
		}
	}
}

// typesCompatible returns true when an input type can legally consume an output type.
// Compatibility is exact-string match, or one side wrapping the other in optional<...>.
func typesCompatible(inputType, outputType string) bool {
	if inputType == outputType {
		return true
	}
	if inputType == "optional<"+outputType+">" {
		return true
	}
	if outputType == "optional<"+inputType+">" {
		return true
	}
	return false
}

// checkGranularity applies Nyquist Compliance heuristics for task granularity (V13).
// Flags potentially over-large tasks, over-large graphs, and overloaded milestones
// as INFO findings (advisory, not blocking).
func (sv *SemanticValidator) checkGranularity(graph *TaskGraph, result *ValidationResult) {
	for i, t := range graph.Tasks {
		if strings.EqualFold(strings.TrimSpace(t.Estimate), "large") {
			result.AddError(ValidationError{
				Rule:     "V13",
				Severity: SeverityInfo,
				Path:     fmt.Sprintf("tasks[%d].estimate", i),
				Message: fmt.Sprintf(
					"Task '%s' has estimate 'large'. Large tasks tend to violate Nyquist Compliance — they bundle too many concerns to verify atomically.",
					t.TaskID,
				),
				Suggestion: "Decompose this task into 2-4 smaller tasks, each with its own goal and acceptance criteria.",
				Context:    t.Estimate,
			})
		}
	}

	if len(graph.Tasks) > 20 {
		result.AddError(ValidationError{
			Rule:     "V13",
			Severity: SeverityInfo,
			Path:     "tasks",
			Message: fmt.Sprintf(
				"Task graph has %d tasks (>20). Very large graphs are hard to plan, track, and review; this often indicates over-decomposition.",
				len(graph.Tasks),
			),
			Suggestion: "Consider consolidating tasks that always execute together, or grouping related work into a smaller number of higher-level tasks.",
			Context:    fmt.Sprintf("%d tasks", len(graph.Tasks)),
		})
	}

	for i, m := range graph.Milestones {
		if len(m.TaskIDs) > 8 {
			result.AddError(ValidationError{
				Rule:     "V13",
				Severity: SeverityInfo,
				Path:     fmt.Sprintf("milestones[%d]", i),
				Message: fmt.Sprintf(
					"Milestone '%s' contains %d tasks (>8). Overloaded milestones obscure progress and are harder to plan around.",
					m.Name, len(m.TaskIDs),
				),
				Suggestion: "Split this milestone into smaller sub-milestones with focused, distinct outcomes.",
				Context:    fmt.Sprintf("%d tasks", len(m.TaskIDs)),
			})
		}
	}
}

// checkMissingDependencyLinks detects undeclared dependencies (V14).
// If task B's input.source textually references task A's task_id or one of A's
// output names but A is not in B's depends_on, that's a hidden edge in the graph.
func (sv *SemanticValidator) checkMissingDependencyLinks(graph *TaskGraph, taskIndex map[string]int, result *ValidationResult) {
	// Map every output name to the task that produces it.
	// On collisions (same output name from multiple tasks), first writer wins.
	outputToTask := make(map[string]string)
	for _, t := range graph.Tasks {
		for _, out := range t.Outputs {
			if out.Name == "" {
				continue
			}
			// On collisions (same output name from multiple tasks), first writer wins.
			if _, exists := outputToTask[out.Name]; !exists {
				outputToTask[out.Name] = t.TaskID
			}
		}
	}

	for i, t := range graph.Tasks {
		deps, _, err := t.ParseDependsOn()
		if err != nil {
			continue // Already reported by V4.
		}
		declared := make(map[string]bool, len(deps))
		for _, d := range deps {
			declared[d] = true
		}

		// Avoid emitting the same (task, referenced) pair more than once even if
		// it shows up across multiple inputs.
		alreadyFlagged := make(map[string]bool)

		for j, in := range t.Inputs {
			source := in.Source
			if source == "" {
				continue
			}

			referenced := make(map[string]bool)

			for otherID := range taskIndex {
				if otherID == t.TaskID {
					continue
				}
				if containsWord(source, otherID) {
					referenced[otherID] = true
				}
			}

			for outName, ownerID := range outputToTask {
				if ownerID == t.TaskID {
					continue
				}
				if containsWord(source, outName) {
					referenced[ownerID] = true
				}
			}

			for refID := range referenced {
				if declared[refID] || alreadyFlagged[refID] {
					continue
				}
				alreadyFlagged[refID] = true
				result.AddError(ValidationError{
					Rule:     "V14",
					Severity: SeverityWarning,
					Path:     fmt.Sprintf("tasks[%d].inputs[%d].source", i, j),
					Message: fmt.Sprintf(
						"Task '%s' references task '%s' in input.source but does not declare it in depends_on. This hides a real dependency edge from the graph.",
						t.TaskID, refID,
					),
					Suggestion: fmt.Sprintf(
						"Add '%s' to tasks[%d].depends_on, or rephrase input.source if the reference is incidental.",
						refID, i,
					),
					Context: source,
				})
			}
		}
	}
}

// containsWord reports whether s contains substr as a whole token, where a token
// is delimited by anything other than [A-Za-z0-9_-]. Hyphens are included because
// task IDs use kebab-case (^[a-z0-9]+(-[a-z0-9]+)*$).
func containsWord(s, substr string) bool {
	if substr == "" || len(substr) > len(s) {
		return false
	}
	from := 0
	for from <= len(s)-len(substr) {
		i := strings.Index(s[from:], substr)
		if i < 0 {
			return false
		}
		start := from + i
		end := start + len(substr)
		leftOK := start == 0 || !isIdentByte(s[start-1])
		rightOK := end == len(s) || !isIdentByte(s[end])
		if leftOK && rightOK {
			return true
		}
		from = start + 1
	}
	return false
}

func isIdentByte(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '_' || b == '-'
}
