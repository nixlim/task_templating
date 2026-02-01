package beads

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nixlim/task_templating/internal/validator"
)

// --- Task .13: Tests for priority and estimate mapping ---

func TestMapPriority(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"critical", 0},
		{"high", 1},
		{"medium", 2},
		{"low", 3},
		{"", 2},         // default
		{"unknown", 2},  // unrecognized
		{"Critical", 0}, // case insensitive
		{"HIGH", 1},     // case insensitive
	}
	for _, tt := range tests {
		got := MapPriority(tt.input)
		if got != tt.want {
			t.Errorf("MapPriority(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestMapEstimate(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"trivial", 15},
		{"small", 60},
		{"medium", 240},
		{"large", 480},
		{"unknown", 0},
		{"", 0},
		{"Trivial", 15}, // case insensitive
		{"LARGE", 480},  // case insensitive
	}
	for _, tt := range tests {
		got := MapEstimate(tt.input)
		if got != tt.want {
			t.Errorf("MapEstimate(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// --- Task .14: Tests for description composition ---

func TestComposeDescription_AllSections(t *testing.T) {
	task := &validator.TaskNode{
		Goal: "Calculate discounted total for an order.",
		Inputs: []validator.InputSpec{
			{Name: "price", Type: "f64", Constraints: "price > 0", Source: "Order record"},
		},
		Outputs: []validator.OutputSpec{
			{Name: "total", Type: "f64", Constraints: "total >= 0", Destination: "Return value"},
		},
		Constraints: json.RawMessage(`["Pure function", "No I/O"]`),
		NonGoals:    []string{"Tax calculation", "Currency conversion"},
		ErrorCases: []validator.ErrorSpec{
			{Condition: "price is zero", Behavior: "Return error", Output: "invalid price"},
		},
	}

	desc := ComposeDescription(task)

	// Check goal is first.
	if !strings.HasPrefix(desc, "Calculate discounted total") {
		t.Error("Description should start with goal text")
	}

	// Check sections exist.
	if !strings.Contains(desc, "## Inputs") {
		t.Error("Missing Inputs section")
	}
	if !strings.Contains(desc, "**price** (`f64`): price > 0 -- Source: Order record") {
		t.Error("Input not formatted correctly")
	}

	if !strings.Contains(desc, "## Outputs") {
		t.Error("Missing Outputs section")
	}
	if !strings.Contains(desc, "**total** (`f64`): total >= 0 -- Dest: Return value") {
		t.Error("Output not formatted correctly")
	}

	if !strings.Contains(desc, "## Constraints") {
		t.Error("Missing Constraints section")
	}
	if !strings.Contains(desc, "- Pure function") {
		t.Error("Constraint not listed")
	}

	if !strings.Contains(desc, "## Non-Goals") {
		t.Error("Missing Non-Goals section")
	}
	if !strings.Contains(desc, "- Tax calculation") {
		t.Error("Non-goal not listed")
	}

	if !strings.Contains(desc, "## Error Cases") {
		t.Error("Missing Error Cases section")
	}
	if !strings.Contains(desc, "**price is zero**") {
		t.Error("Error case condition not formatted correctly")
	}
}

func TestComposeDescription_GoalOnly(t *testing.T) {
	task := &validator.TaskNode{
		Goal: "Minimal task with only a goal.",
	}

	desc := ComposeDescription(task)
	if desc != "Minimal task with only a goal." {
		t.Errorf("Expected just goal text, got: %q", desc)
	}
}

func TestComposeDescription_NAFieldsOmitted(t *testing.T) {
	task := &validator.TaskNode{
		Goal:        "Task with N/A fields.",
		Constraints: json.RawMessage(`{"status": "N/A", "reason": "not applicable"}`),
	}

	desc := ComposeDescription(task)
	if strings.Contains(desc, "## Constraints") {
		t.Error("N/A constraints section should be omitted")
	}
}

func TestFormatAcceptance(t *testing.T) {
	criteria := []string{"Test A passes", "Test B returns 42", "No regressions"}
	result := FormatAcceptance(criteria)
	expected := "- Test A passes\n- Test B returns 42\n- No regressions"
	if result != expected {
		t.Errorf("FormatAcceptance got:\n%s\nwant:\n%s", result, expected)
	}

	// Empty
	if FormatAcceptance(nil) != "" {
		t.Error("FormatAcceptance(nil) should return empty string")
	}
}

func TestBuildTemplateMetadata(t *testing.T) {
	task := &validator.TaskNode{
		TaskID:     "test-task",
		FilesScope: json.RawMessage(`["file.go", "file_test.go"]`),
		Effects:    json.RawMessage(`"None"`),
		Inputs: []validator.InputSpec{
			{Name: "x", Type: "int", Constraints: "x > 0", Source: "arg"},
		},
		Outputs: []validator.OutputSpec{
			{Name: "y", Type: "int", Constraints: "y >= 0", Destination: "return"},
		},
	}

	jsonStr, err := BuildTemplateMetadata(task)
	if err != nil {
		t.Fatalf("BuildTemplateMetadata error: %v", err)
	}

	// Verify it's valid JSON.
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	tmpl, ok := parsed["_template"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing _template wrapper")
	}

	if tmpl["version"] != "0.2.0" {
		t.Errorf("version = %v, want 0.2.0", tmpl["version"])
	}
	if tmpl["task_id"] != "test-task" {
		t.Errorf("task_id = %v, want test-task", tmpl["task_id"])
	}
	if tmpl["effects"] != "None" {
		t.Errorf("effects = %v, want None", tmpl["effects"])
	}

	fs, ok := tmpl["files_scope"].([]interface{})
	if !ok || len(fs) != 2 {
		t.Errorf("files_scope = %v, want 2-element array", tmpl["files_scope"])
	}
}

// --- Task .15: Tests for command construction ---

func TestBuildSingleTaskCommands(t *testing.T) {
	task := &validator.TaskNode{
		TaskID:     "my-task",
		TaskName:   "Do the thing",
		Goal:       "The thing is done.",
		Priority:   "high",
		Estimate:   "small",
		Notes:      "Some notes",
		Acceptance: []string{"It works"},
		Inputs:     []validator.InputSpec{},
		Outputs:    []validator.OutputSpec{},
	}

	creator := &Creator{}
	cmds, err := creator.BuildSingleTaskCommands(task)
	if err != nil {
		t.Fatalf("BuildSingleTaskCommands error: %v", err)
	}

	if len(cmds) != 2 {
		t.Fatalf("Expected 2 commands (create + update), got %d", len(cmds))
	}

	// Check create command.
	create := cmds[0]
	if create.Type != "create-task" {
		t.Errorf("First command type = %s, want create-task", create.Type)
	}
	if create.TaskID != "my-task" {
		t.Errorf("TaskID = %s, want my-task", create.TaskID)
	}

	args := strings.Join(create.Args, " ")
	if !strings.Contains(args, "--title") {
		t.Error("Missing --title flag")
	}
	if !strings.Contains(args, "--type task") {
		t.Error("Missing --type task")
	}
	if !strings.Contains(args, "--priority 1") {
		t.Error("Priority should be 1 for 'high'")
	}
	if !strings.Contains(args, "--estimate 60") {
		t.Error("Estimate should be 60 for 'small'")
	}
	if !strings.Contains(args, "--labels taskval-managed") {
		t.Error("Missing --labels taskval-managed")
	}
	if !strings.Contains(args, "--silent") {
		t.Error("Missing --silent flag")
	}

	// Check update command.
	update := cmds[1]
	if update.Type != "update-design" {
		t.Errorf("Second command type = %s, want update-design", update.Type)
	}
}

func TestBuildGraphCommands(t *testing.T) {
	graph := &validator.TaskGraph{
		Version: "0.1.0",
		Milestones: []validator.Milestone{
			{Name: "Phase 1", TaskIDs: []string{"task-a"}},
		},
		Tasks: []validator.TaskNode{
			{
				TaskID:     "task-a",
				TaskName:   "Task A",
				Goal:       "Do A.",
				Inputs:     []validator.InputSpec{},
				Outputs:    []validator.OutputSpec{},
				Acceptance: []string{"A is done"},
			},
			{
				TaskID:     "task-b",
				TaskName:   "Task B",
				Goal:       "Do B.",
				Inputs:     []validator.InputSpec{},
				Outputs:    []validator.OutputSpec{},
				DependsOn:  json.RawMessage(`["task-a"]`),
				Acceptance: []string{"B is done"},
			},
		},
	}

	creator := &Creator{Filename: "test.json"}
	cmds, err := creator.BuildGraphCommands(graph)
	if err != nil {
		t.Fatalf("BuildGraphCommands error: %v", err)
	}

	// Expect: 1 epic + 2 tasks + 1 dep + 2 updates = 6 commands.
	if len(cmds) != 6 {
		t.Fatalf("Expected 6 commands, got %d", len(cmds))
	}

	// First command is epic creation.
	if cmds[0].Type != "create-epic" {
		t.Errorf("First command type = %s, want create-epic", cmds[0].Type)
	}
	epicArgs := strings.Join(cmds[0].Args, " ")
	if !strings.Contains(epicArgs, "--type epic") {
		t.Error("Epic missing --type epic")
	}
	// Milestone-based title.
	if !strings.Contains(epicArgs, "Task Graph: Phase 1") {
		t.Errorf("Epic title should use milestone name, got args: %s", epicArgs)
	}

	// Tasks should come next.
	if cmds[1].Type != "create-task" || cmds[2].Type != "create-task" {
		t.Error("Commands 2 and 3 should be create-task")
	}

	// Task A should be before Task B (topological order).
	if cmds[1].TaskID != "task-a" {
		t.Errorf("First task should be task-a (no deps), got %s", cmds[1].TaskID)
	}
	if cmds[2].TaskID != "task-b" {
		t.Errorf("Second task should be task-b (depends on task-a), got %s", cmds[2].TaskID)
	}

	// Task create args should include --parent <epic-id>.
	taskArgs := strings.Join(cmds[1].Args, " ")
	if !strings.Contains(taskArgs, "--parent <epic-id>") {
		t.Error("Task missing --parent <epic-id>")
	}

	// Dependency command.
	if cmds[3].Type != "dep-add" {
		t.Errorf("Command 4 type = %s, want dep-add", cmds[3].Type)
	}
}

func TestResolveEpicTitle(t *testing.T) {
	// 1. Explicit override.
	c := &Creator{EpicTitle: "Custom Title", Filename: "plan.json"}
	graph := &validator.TaskGraph{
		Milestones: []validator.Milestone{{Name: "M1"}},
	}
	if got := c.resolveEpicTitle(graph); got != "Custom Title" {
		t.Errorf("With explicit title: got %q", got)
	}

	// 2. Milestone-based.
	c = &Creator{Filename: "plan.json"}
	if got := c.resolveEpicTitle(graph); got != "Task Graph: M1" {
		t.Errorf("With milestone: got %q", got)
	}

	// 3. Filename-based.
	c = &Creator{Filename: "plan.json"}
	graph = &validator.TaskGraph{}
	if got := c.resolveEpicTitle(graph); got != "Task Graph: plan.json" {
		t.Errorf("With filename: got %q", got)
	}

	// 4. Stdin fallback.
	c = &Creator{Filename: "-"}
	if got := c.resolveEpicTitle(graph); got != "Task Graph: (stdin)" {
		t.Errorf("With stdin: got %q", got)
	}

	c = &Creator{}
	if got := c.resolveEpicTitle(graph); got != "Task Graph: (stdin)" {
		t.Errorf("With empty filename: got %q", got)
	}
}

func TestFormatDryRunOutput(t *testing.T) {
	cmds := []BdCommand{
		{Args: []string{"create", "--title", "Epic", "--type", "epic"}, Type: "create-epic"},
		{Args: []string{"create", "--title", "Task 1", "--type", "task"}, Type: "create-task"},
		{Args: []string{"dep", "add", "bd-1", "bd-2"}, Type: "dep-add"},
		{Args: []string{"update", "bd-1", "--design", "{}"}, Type: "update-design"},
	}

	output := FormatDryRunOutput(cmds)

	if !strings.Contains(output, "DRY RUN") {
		t.Error("Missing DRY RUN header")
	}
	if !strings.Contains(output, "[DRY-RUN] bd create") {
		t.Error("Missing [DRY-RUN] prefix for create commands")
	}
	if !strings.Contains(output, "[DRY-RUN] bd dep") {
		t.Error("Missing [DRY-RUN] prefix for dep commands")
	}
	// update-design should be skipped in dry-run output.
	if strings.Contains(output, "[DRY-RUN] bd update") {
		t.Error("update-design should not appear in dry-run output")
	}
	if !strings.Contains(output, "Would create 1 epic + 1 tasks, link 1 dependencies.") {
		t.Errorf("Summary line incorrect, got:\n%s", output)
	}
}

func TestFormatTextOutput(t *testing.T) {
	result := &CreationResult{
		EpicID:     "bd-abc",
		EpicTitle:  "Test Epic",
		TaskIDs:    map[string]string{"task-a": "bd-111"},
		TaskTitles: map[string]string{"task-a": "Task A"},
		Created:    2,
		Deps:       0,
	}

	output := FormatTextOutput(result)
	if !strings.Contains(output, "BEADS CREATION") {
		t.Error("Missing BEADS CREATION header")
	}
	if !strings.Contains(output, "bd-abc") {
		t.Error("Missing epic ID")
	}
	if !strings.Contains(output, "bd-111") {
		t.Error("Missing task ID")
	}
	if !strings.Contains(output, "1 epic + 1 tasks created") {
		t.Errorf("Summary incorrect, got:\n%s", output)
	}
}

func TestFormatJSONOutput(t *testing.T) {
	result := &CreationResult{
		EpicID:  "bd-abc",
		TaskIDs: map[string]string{"task-a": "bd-111", "task-b": "bd-222"},
		Created: 3,
		Deps:    1,
	}

	out := FormatJSONOutput(result)
	if out.EpicID != "bd-abc" {
		t.Errorf("EpicID = %s, want bd-abc", out.EpicID)
	}
	if len(out.Tasks) != 2 {
		t.Errorf("Tasks count = %d, want 2", len(out.Tasks))
	}
	if out.DepsLinked != 1 {
		t.Errorf("DepsLinked = %d, want 1", out.DepsLinked)
	}
	if out.TotalCreated != 3 {
		t.Errorf("TotalCreated = %d, want 3", out.TotalCreated)
	}
}
