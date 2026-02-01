package beads

import (
	"fmt"
	"strings"

	"github.com/nixlim/task_templating/internal/validator"
)

// Creator orchestrates the creation of Beads issues from validated task templates.
type Creator struct {
	// DryRun when true prints commands without executing them.
	DryRun bool

	// EpicTitle overrides the auto-generated epic title (graph mode only).
	EpicTitle string

	// Filename is the input file name, used for epic title derivation.
	Filename string
}

// CreationResult holds the outcome of a beads creation operation.
type CreationResult struct {
	// EpicID is the bd issue ID for the epic (graph mode only).
	EpicID string

	// EpicTitle is the title used for the epic.
	EpicTitle string

	// TaskIDs maps template task_id to bd issue ID.
	TaskIDs map[string]string

	// TaskTitles maps template task_id to the task_name used as title.
	TaskTitles map[string]string

	// Commands holds all bd commands executed (or that would be executed in dry-run).
	Commands []string

	// Created is the number of issues created.
	Created int

	// Deps is the number of dependencies linked.
	Deps int

	// DepsDetail holds dependency info for output formatting.
	DepsDetail []DepLink
}

// DepLink represents a dependency relationship between two beads issues.
type DepLink struct {
	TaskBdID string
	DepBdID  string
}

// BdCommand represents a bd CLI command to be executed.
type BdCommand struct {
	// Args is the full argument list (e.g., ["create", "--title", "..."]).
	Args []string

	// TaskID is the template task_id this command relates to (for ID mapping).
	TaskID string

	// Type indicates the purpose: "create-epic", "create-task", "dep-add", "update-design".
	Type string

	// DepTaskID and DepOnID are set for dep-add commands.
	DepTaskID string
	DepOnID   string
}

// BuildSingleTaskCommands constructs the bd commands for single task mode.
func (c *Creator) BuildSingleTaskCommands(task *validator.TaskNode) ([]BdCommand, error) {
	var cmds []BdCommand

	// Step 1: Create the task issue.
	createArgs := c.buildTaskCreateArgs(task, "")
	cmds = append(cmds, BdCommand{
		Args:   createArgs,
		TaskID: task.TaskID,
		Type:   "create-task",
	})

	// Step 2: Update with template metadata.
	designJSON, err := BuildTemplateMetadata(task)
	if err != nil {
		return nil, fmt.Errorf("building template metadata for '%s': %w", task.TaskID, err)
	}
	cmds = append(cmds, BdCommand{
		Args:   []string{"update", "<" + task.TaskID + "-id>", "--design", designJSON},
		TaskID: task.TaskID,
		Type:   "update-design",
	})

	return cmds, nil
}

// BuildGraphCommands constructs the bd commands for graph mode.
func (c *Creator) BuildGraphCommands(graph *validator.TaskGraph) ([]BdCommand, error) {
	var cmds []BdCommand

	// Step 1: Create the epic.
	epicTitle := c.resolveEpicTitle(graph)
	epicPriority := c.resolveGraphPriority(graph)
	epicArgs := []string{
		"create",
		"--title", epicTitle,
		"--type", "epic",
		"--priority", fmt.Sprintf("%d", epicPriority),
		"--labels", "taskval-managed",
		"--silent",
	}
	cmds = append(cmds, BdCommand{
		Args: epicArgs,
		Type: "create-epic",
	})

	// Step 2: Create tasks in topological order.
	ordered := topologicalSort(graph)

	for _, task := range ordered {
		createArgs := c.buildTaskCreateArgs(task, "<epic-id>")
		cmds = append(cmds, BdCommand{
			Args:   createArgs,
			TaskID: task.TaskID,
			Type:   "create-task",
		})
	}

	// Step 3: Add dependency links.
	for _, task := range ordered {
		deps, _, err := task.ParseDependsOn()
		if err != nil {
			continue
		}
		for _, dep := range deps {
			cmds = append(cmds, BdCommand{
				Args:      []string{"dep", "add", "<" + task.TaskID + "-id>", "<" + dep + "-id>"},
				Type:      "dep-add",
				DepTaskID: task.TaskID,
				DepOnID:   dep,
			})
		}
	}

	// Step 4: Update template metadata for each task.
	for _, task := range ordered {
		designJSON, err := BuildTemplateMetadata(task)
		if err != nil {
			return nil, fmt.Errorf("building template metadata for '%s': %w", task.TaskID, err)
		}
		cmds = append(cmds, BdCommand{
			Args:   []string{"update", "<" + task.TaskID + "-id>", "--design", designJSON},
			TaskID: task.TaskID,
			Type:   "update-design",
		})
	}

	return cmds, nil
}

// buildTaskCreateArgs constructs the arguments for a bd create command for a single task.
func (c *Creator) buildTaskCreateArgs(task *validator.TaskNode, parentID string) []string {
	args := []string{
		"create",
		"--title", truncate(task.TaskName, 500),
		"--type", "task",
		"--description", ComposeDescription(task),
	}

	acceptance := FormatAcceptance(task.Acceptance)
	if acceptance != "" {
		args = append(args, "--acceptance", acceptance)
	}

	args = append(args, "--priority", fmt.Sprintf("%d", MapPriority(task.Priority)))

	est := MapEstimate(task.Estimate)
	if est > 0 {
		args = append(args, "--estimate", fmt.Sprintf("%d", est))
	}

	if task.Notes != "" {
		args = append(args, "--notes", task.Notes)
	}

	if parentID != "" {
		args = append(args, "--parent", parentID)
	}

	args = append(args, "--labels", "taskval-managed", "--silent")
	return args
}

// resolveEpicTitle determines the epic title using the resolution order from the spec.
func (c *Creator) resolveEpicTitle(graph *validator.TaskGraph) string {
	// 1. Explicit override.
	if c.EpicTitle != "" {
		return c.EpicTitle
	}

	// 2. First milestone name.
	if len(graph.Milestones) > 0 {
		return "Task Graph: " + graph.Milestones[0].Name
	}

	// 3. Derive from filename.
	if c.Filename != "" && c.Filename != "-" {
		return "Task Graph: " + c.Filename
	}

	// 4. Stdin fallback.
	return "Task Graph: (stdin)"
}

// resolveGraphPriority picks the highest priority across all tasks.
func (c *Creator) resolveGraphPriority(graph *validator.TaskGraph) int {
	best := 2 // default medium
	for _, t := range graph.Tasks {
		p := MapPriority(t.Priority)
		if p < best {
			best = p
		}
	}
	return best
}

// FormatTextOutput formats the creation result as human-readable text.
func FormatTextOutput(result *CreationResult) string {
	var sb strings.Builder
	sb.WriteString("\nBEADS CREATION\n")

	if result.EpicID != "" {
		sb.WriteString(fmt.Sprintf("  Epic created: %s %q\n", result.EpicID, result.EpicTitle))
	}

	for taskID, bdID := range result.TaskIDs {
		title := result.TaskTitles[taskID]
		sb.WriteString(fmt.Sprintf("  Task created: %s %q (%s)\n", bdID, title, taskID))
	}

	for _, dep := range result.DepsDetail {
		sb.WriteString(fmt.Sprintf("  Dependency:   %s blocked-by %s\n", dep.TaskBdID, dep.DepBdID))
	}

	epicCount := 0
	if result.EpicID != "" {
		epicCount = 1
	}
	sb.WriteString(fmt.Sprintf("\n  Summary: %d epic + %d tasks created, %d dependencies linked.\n",
		epicCount, result.Created-epicCount, result.Deps))

	return sb.String()
}

// BeadsJSON is the JSON output structure for beads creation results.
type BeadsJSON struct {
	EpicID       string            `json:"epic_id,omitempty"`
	Tasks        map[string]string `json:"tasks"`
	DepsLinked   int               `json:"dependencies_linked"`
	TotalCreated int               `json:"total_created"`
}

// FormatJSONOutput creates the BeadsJSON structure from a CreationResult.
func FormatJSONOutput(result *CreationResult) *BeadsJSON {
	return &BeadsJSON{
		EpicID:       result.EpicID,
		Tasks:        result.TaskIDs,
		DepsLinked:   result.Deps,
		TotalCreated: result.Created,
	}
}

// FormatDryRunOutput formats the dry-run output showing commands that would be executed.
func FormatDryRunOutput(cmds []BdCommand) string {
	var sb strings.Builder
	sb.WriteString("\nBEADS CREATION (DRY RUN)\n")

	epicCount := 0
	taskCount := 0
	depCount := 0

	for _, cmd := range cmds {
		switch cmd.Type {
		case "create-epic":
			epicCount++
		case "create-task":
			taskCount++
		case "dep-add":
			depCount++
		}
		// Skip update-design in dry-run output for brevity.
		if cmd.Type == "update-design" {
			continue
		}
		sb.WriteString(fmt.Sprintf("  [DRY-RUN] bd %s\n", formatArgs(cmd.Args)))
	}

	sb.WriteString(fmt.Sprintf("\n  Summary: Would create %d epic + %d tasks, link %d dependencies.\n",
		epicCount, taskCount, depCount))

	return sb.String()
}

// topologicalSort returns tasks in dependency order (dependencies before dependents).
func topologicalSort(graph *validator.TaskGraph) []*validator.TaskNode {
	taskIndex := make(map[string]int, len(graph.Tasks))
	for i, t := range graph.Tasks {
		taskIndex[t.TaskID] = i
	}

	// Build adjacency list and in-degree count.
	adj := make(map[string][]string)
	inDegree := make(map[string]int)
	for _, t := range graph.Tasks {
		inDegree[t.TaskID] = 0
		adj[t.TaskID] = nil
	}
	for _, t := range graph.Tasks {
		deps, _, err := t.ParseDependsOn()
		if err != nil {
			continue
		}
		for _, dep := range deps {
			if _, exists := taskIndex[dep]; !exists {
				continue
			}
			adj[dep] = append(adj[dep], t.TaskID)
			inDegree[t.TaskID]++
		}
	}

	// Kahn's algorithm.
	var queue []string
	for _, t := range graph.Tasks {
		if inDegree[t.TaskID] == 0 {
			queue = append(queue, t.TaskID)
		}
	}

	var ordered []*validator.TaskNode
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		idx := taskIndex[id]
		ordered = append(ordered, &graph.Tasks[idx])
		for _, neighbor := range adj[id] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	return ordered
}

// truncate shortens a string to maxLen if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// formatArgs joins command arguments with proper quoting for display.
func formatArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t\n\"'") || strings.Contains(a, "--") && i > 0 {
			// Don't quote flags or simple values.
			if strings.HasPrefix(a, "--") {
				quoted[i] = a
			} else {
				quoted[i] = fmt.Sprintf("%q", a)
			}
		} else {
			quoted[i] = a
		}
	}
	return strings.Join(quoted, " ")
}
