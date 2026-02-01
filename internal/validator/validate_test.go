package validator

import (
	"encoding/json"
	"testing"
)

func TestValidSingleTask(t *testing.T) {
	task := map[string]any{
		"task_id":   "test-task",
		"task_name": "Implement a test feature",
		"goal":      "The test feature returns correct results for all inputs.",
		"inputs": []map[string]string{
			{"name": "data", "type": "string", "constraints": "len > 0", "source": "User input"},
		},
		"outputs": []map[string]string{
			{"name": "result", "type": "string", "constraints": "none", "destination": "stdout"},
		},
		"acceptance": []string{
			"Given input 'hello', output is 'HELLO'",
		},
		"depends_on":  map[string]string{"status": "N/A", "reason": "Standalone function"},
		"constraints": []string{"No external dependencies allowed"},
		"files_scope": []string{"internal/test.go"},
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshaling test data: %v", err)
	}

	result, err := Validate(data, ModeSingleTask)
	if err != nil {
		t.Fatalf("validation error: %v", err)
	}

	if !result.Valid {
		for _, e := range result.Errors {
			t.Errorf("unexpected error: %s", e.Error())
		}
	}
}

func TestInvalidTaskID(t *testing.T) {
	task := map[string]any{
		"task_id":   "Invalid_Task_ID",
		"task_name": "Implement a test feature",
		"goal":      "The test feature works as specified.",
		"inputs": []map[string]string{
			{"name": "data", "type": "string", "constraints": "none", "source": "User input"},
		},
		"outputs": []map[string]string{
			{"name": "result", "type": "string", "constraints": "none", "destination": "stdout"},
		},
		"acceptance": []string{"Given input 'x', output is 'y'"},
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	result, err := Validate(data, ModeSingleTask)
	if err != nil {
		t.Fatalf("validation error: %v", err)
	}

	if result.Valid {
		t.Error("expected validation to fail for invalid task_id")
	}

	found := false
	for _, e := range result.Errors {
		if e.Rule == "SCHEMA" {
			found = true
		}
	}
	if !found {
		t.Error("expected SCHEMA error for invalid task_id pattern")
	}
}

func TestCycleDetection(t *testing.T) {
	graph := map[string]any{
		"version": "0.1.0",
		"tasks": []map[string]any{
			{
				"task_id":     "task-a",
				"task_name":   "Implement task A",
				"goal":        "Task A produces output X.",
				"inputs":      []map[string]string{{"name": "in", "type": "string", "constraints": "none", "source": "caller"}},
				"outputs":     []map[string]string{{"name": "out", "type": "string", "constraints": "none", "destination": "return"}},
				"acceptance":  []string{"Output X is produced"},
				"depends_on":  []string{"task-b"},
				"constraints": []string{"No constraints"},
				"files_scope": []string{"a.go"},
			},
			{
				"task_id":     "task-b",
				"task_name":   "Implement task B",
				"goal":        "Task B produces output Y.",
				"inputs":      []map[string]string{{"name": "in", "type": "string", "constraints": "none", "source": "caller"}},
				"outputs":     []map[string]string{{"name": "out", "type": "string", "constraints": "none", "destination": "return"}},
				"acceptance":  []string{"Output Y is produced"},
				"depends_on":  []string{"task-a"},
				"constraints": []string{"No constraints"},
				"files_scope": []string{"b.go"},
			},
		},
	}

	data, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	result, err := Validate(data, ModeTaskGraph)
	if err != nil {
		t.Fatalf("validation error: %v", err)
	}

	if result.Valid {
		t.Error("expected validation to fail for cyclic dependencies")
	}

	foundCycle := false
	for _, e := range result.Errors {
		if e.Rule == "V5" {
			foundCycle = true
		}
	}
	if !foundCycle {
		t.Error("expected V5 cycle error")
	}
}

func TestGoalForbiddenWords(t *testing.T) {
	tests := []struct {
		goal    string
		wantErr bool
	}{
		{"Try to add search functionality.", true},
		{"Explore various caching strategies.", true},
		{"Investigate why the build is slow.", true},
		{"Look into adding an export feature.", true},
		{"The Search function returns ranked results from the database.", false},
	}

	for _, tc := range tests {
		task := map[string]any{
			"task_id":     "goal-test",
			"task_name":   "Implement goal test",
			"goal":        tc.goal,
			"inputs":      []map[string]string{{"name": "in", "type": "string", "constraints": "none", "source": "test"}},
			"outputs":     []map[string]string{{"name": "out", "type": "string", "constraints": "none", "destination": "test"}},
			"acceptance":  []string{"Concrete verifiable assertion here"},
			"depends_on":  map[string]string{"status": "N/A", "reason": "Standalone function, no dependencies"},
			"constraints": []string{"Test constraint"},
			"files_scope": []string{"test.go"},
		}

		data, err := json.Marshal(task)
		if err != nil {
			t.Fatalf("marshaling: %v", err)
		}

		result, err := Validate(data, ModeSingleTask)
		if err != nil {
			t.Fatalf("validation error: %v", err)
		}

		hasGoalError := false
		for _, e := range result.Errors {
			if e.Rule == "V6" && e.Severity == SeverityError {
				hasGoalError = true
			}
		}

		if tc.wantErr && !hasGoalError {
			t.Errorf("goal %q: expected V6 error, got none", tc.goal)
		}
		if !tc.wantErr && hasGoalError {
			t.Errorf("goal %q: unexpected V6 error", tc.goal)
		}
	}
}

func TestDanglingDependencyReference(t *testing.T) {
	graph := map[string]any{
		"version": "0.1.0",
		"tasks": []map[string]any{
			{
				"task_id":     "task-a",
				"task_name":   "Implement task A",
				"goal":        "Task A produces output X.",
				"inputs":      []map[string]string{{"name": "in", "type": "string", "constraints": "none", "source": "caller"}},
				"outputs":     []map[string]string{{"name": "out", "type": "string", "constraints": "none", "destination": "return"}},
				"acceptance":  []string{"Output X is produced"},
				"depends_on":  []string{"does-not-exist"},
				"constraints": []string{"None"},
				"files_scope": []string{"a.go"},
			},
		},
	}

	data, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	result, err := Validate(data, ModeTaskGraph)
	if err != nil {
		t.Fatalf("validation error: %v", err)
	}

	if result.Valid {
		t.Error("expected validation to fail for dangling dependency")
	}

	foundV4 := false
	for _, e := range result.Errors {
		if e.Rule == "V4" {
			foundV4 = true
		}
	}
	if !foundV4 {
		t.Error("expected V4 error for dangling dependency reference")
	}
}

func TestGraphFieldPopulatedOnSuccess(t *testing.T) {
	graph := map[string]any{
		"version": "0.1.0",
		"tasks": []map[string]any{
			{
				"task_id":     "task-a",
				"task_name":   "Implement task A",
				"goal":        "Task A produces output X.",
				"inputs":      []map[string]string{{"name": "in", "type": "string", "constraints": "none", "source": "caller"}},
				"outputs":     []map[string]string{{"name": "out", "type": "string", "constraints": "none", "destination": "return"}},
				"acceptance":  []string{"Output X is produced"},
				"depends_on":  map[string]string{"status": "N/A", "reason": "No deps"},
				"constraints": []string{"No constraints"},
				"files_scope": []string{"a.go"},
			},
		},
	}

	data, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	result, err := Validate(data, ModeTaskGraph)
	if err != nil {
		t.Fatalf("validation error: %v", err)
	}

	if !result.Valid {
		for _, e := range result.Errors {
			t.Errorf("unexpected error: %s", e.Error())
		}
		t.Fatal("validation should pass")
	}

	if result.Graph == nil {
		t.Fatal("Graph should be non-nil after successful graph validation")
	}

	if len(result.Graph.Tasks) != 1 {
		t.Errorf("Graph.Tasks length = %d, want 1", len(result.Graph.Tasks))
	}

	if result.Graph.Tasks[0].TaskID != "task-a" {
		t.Errorf("Graph.Tasks[0].TaskID = %q, want task-a", result.Graph.Tasks[0].TaskID)
	}
}

func TestGraphFieldPopulatedOnSingleTaskSuccess(t *testing.T) {
	task := map[string]any{
		"task_id":     "single-task",
		"task_name":   "Implement a single task",
		"goal":        "The single task returns correct results.",
		"inputs":      []map[string]string{{"name": "data", "type": "string", "constraints": "len > 0", "source": "User input"}},
		"outputs":     []map[string]string{{"name": "result", "type": "string", "constraints": "none", "destination": "stdout"}},
		"acceptance":  []string{"Given input 'hello', output is 'HELLO'"},
		"depends_on":  map[string]string{"status": "N/A", "reason": "Standalone"},
		"constraints": []string{"No external deps"},
		"files_scope": []string{"internal/test.go"},
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	result, err := Validate(data, ModeSingleTask)
	if err != nil {
		t.Fatalf("validation error: %v", err)
	}

	if !result.Valid {
		for _, e := range result.Errors {
			t.Errorf("unexpected error: %s", e.Error())
		}
		t.Fatal("validation should pass")
	}

	if result.Graph == nil {
		t.Fatal("Graph should be non-nil after successful single task validation")
	}

	if len(result.Graph.Tasks) != 1 {
		t.Errorf("Graph.Tasks length = %d, want 1", len(result.Graph.Tasks))
	}
}

func TestGraphFieldNilOnFailure(t *testing.T) {
	// Invalid task_id will fail schema validation.
	task := map[string]any{
		"task_id":    "Invalid_Task_ID",
		"task_name":  "Implement test",
		"goal":       "The test works.",
		"inputs":     []map[string]string{{"name": "in", "type": "string", "constraints": "none", "source": "test"}},
		"outputs":    []map[string]string{{"name": "out", "type": "string", "constraints": "none", "destination": "test"}},
		"acceptance": []string{"Given input, output is correct"},
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	result, err := Validate(data, ModeSingleTask)
	if err != nil {
		t.Fatalf("validation error: %v", err)
	}

	if result.Valid {
		t.Fatal("validation should fail for invalid task_id")
	}

	if result.Graph != nil {
		t.Error("Graph should be nil when validation fails")
	}
}

func TestGraphFieldExcludedFromJSON(t *testing.T) {
	// The Graph field has json:"-" tag, so it should not appear in JSON output.
	result := &ValidationResult{
		Valid: true,
		Stats: ValidationStats{TotalTasks: 1},
		Graph: &TaskGraph{
			Version: "0.1.0",
			Tasks:   []TaskNode{{TaskID: "test"}},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	jsonStr := string(data)
	if contains := "graph"; len(jsonStr) > 0 {
		// Parse back and check there's no "graph" key.
		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("unmarshaling: %v", err)
		}
		if _, exists := parsed["graph"]; exists {
			t.Error("Graph field should be excluded from JSON output (json:\"-\" tag)")
		}
		_ = contains
	}
}

func TestAcceptanceVagueness(t *testing.T) {
	task := map[string]any{
		"task_id":   "vague-test",
		"task_name": "Implement vague test",
		"goal":      "The function processes data and returns results.",
		"inputs":    []map[string]string{{"name": "in", "type": "string", "constraints": "none", "source": "test"}},
		"outputs":   []map[string]string{{"name": "out", "type": "string", "constraints": "none", "destination": "test"}},
		"acceptance": []string{
			"it works correctly",
			"Given input 5, output is 25",
			"output should work as expected",
		},
		"depends_on":  map[string]string{"status": "N/A", "reason": "Standalone function"},
		"constraints": []string{"Test constraint"},
		"files_scope": []string{"test.go"},
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	result, err := Validate(data, ModeSingleTask)
	if err != nil {
		t.Fatalf("validation error: %v", err)
	}

	v7Count := 0
	for _, e := range result.Errors {
		if e.Rule == "V7" {
			v7Count++
		}
	}

	if v7Count < 2 {
		t.Errorf("expected at least 2 V7 warnings, got %d", v7Count)
	}
}
