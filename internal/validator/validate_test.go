package validator

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func hasFinding(r *ValidationResult, rule string, sev Severity) bool {
	for _, e := range r.Errors {
		if e.Rule == rule && e.Severity == sev {
			return true
		}
	}
	return false
}

func hasFindingAt(r *ValidationResult, rule string, sev Severity, pathSubstr string) bool {
	for _, e := range r.Errors {
		if e.Rule == rule && e.Severity == sev && strings.Contains(e.Path, pathSubstr) {
			return true
		}
	}
	return false
}

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

func TestWeaselWordsInGoal(t *testing.T) {
	task := map[string]any{
		"task_id":   "weasel-goal-task",
		"task_name": "Implement the offline auth fallback",
		"goal":      "The handler returns a placeholder token while the real OIDC flow is offline.",
		"inputs": []map[string]string{
			{"name": "req", "type": "string", "constraints": "len > 0", "source": "HTTP body"},
		},
		"outputs": []map[string]string{
			{"name": "token", "type": "string", "constraints": "len > 0", "destination": "HTTP body"},
		},
		"acceptance":  []string{"Given any auth request, returns HTTP 200 with a non-empty token string"},
		"depends_on":  map[string]string{"status": "N/A", "reason": "Standalone HTTP handler"},
		"constraints": []string{"Use stdlib net/http only"},
		"files_scope": []string{"internal/api/auth.go"},
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	result, err := Validate(data, ModeSingleTask)
	if err != nil {
		t.Fatalf("validation error: %v", err)
	}

	if !hasFinding(result, "V11", SeverityWarning) {
		t.Error("expected V11 warning for 'placeholder' in goal")
	}
}

func TestWeaselWordsInAcceptance(t *testing.T) {
	task := map[string]any{
		"task_id":   "weasel-acceptance-task",
		"task_name": "Implement the in-memory cache",
		"goal":      "The cache returns the stored value for a known key, otherwise signals miss.",
		"inputs": []map[string]string{
			{"name": "key", "type": "string", "constraints": "len > 0", "source": "caller"},
		},
		"outputs": []map[string]string{
			{"name": "value", "type": "string", "constraints": "none", "destination": "Return value"},
		},
		"acceptance": []string{
			"Given key 'k1' present in cache, returns ('v1-stored', true)",
			"TTL is hardcoded for now to 60 seconds for every entry",
		},
		"depends_on":  map[string]string{"status": "N/A", "reason": "Pure cache module"},
		"constraints": []string{"Use sync.Map for storage"},
		"files_scope": []string{"internal/cache/cache.go"},
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	result, err := Validate(data, ModeSingleTask)
	if err != nil {
		t.Fatalf("validation error: %v", err)
	}

	if !hasFinding(result, "V11", SeverityWarning) {
		t.Error("expected V11 warning for 'hardcoded for now' in acceptance")
	}
}

func TestWeaselWordsCleanGoal(t *testing.T) {
	task := map[string]any{
		"task_id":   "rate-limiter",
		"task_name": "Implement the per-IP rate limiter",
		"goal":      "The rate limiter rejects more than 100 requests per minute from a single IP.",
		"inputs": []map[string]string{
			{"name": "ip", "type": "string", "constraints": "len > 0", "source": "request remote address"},
		},
		"outputs": []map[string]string{
			{"name": "allowed", "type": "bool", "constraints": "none", "destination": "Return value"},
		},
		"acceptance":  []string{"After 100 calls in 60 seconds from one IP, the 101st call returns false"},
		"depends_on":  map[string]string{"status": "N/A", "reason": "Pure module, no upstream tasks"},
		"constraints": []string{"Use a token-bucket algorithm"},
		"files_scope": []string{"internal/ratelimit/limiter.go"},
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	result, err := Validate(data, ModeSingleTask)
	if err != nil {
		t.Fatalf("validation error: %v", err)
	}

	if hasFinding(result, "V11", SeverityWarning) {
		t.Error("did not expect V11 warning for clean goal/acceptance")
	}
}

func TestWeaselWordsSemverNoFalsePositive(t *testing.T) {
	task := map[string]any{
		"task_id":   "upgrade-api",
		"task_name": "Upgrade upstream API client",
		"goal":      "Targets API v1.2.3 of the upstream service.",
		"inputs": []map[string]string{
			{"name": "config", "type": "string", "constraints": "len > 0", "source": "config file"},
		},
		"outputs": []map[string]string{
			{"name": "client", "type": "object", "constraints": "none", "destination": "Return value"},
		},
		"acceptance":  []string{"Client connects to upstream v2.0.1 endpoint without errors"},
		"depends_on":  map[string]string{"status": "N/A", "reason": "Standalone"},
		"constraints": []string{"Must use TLS 1.3"},
		"files_scope": []string{"internal/api/client.go"},
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	result, err := Validate(data, ModeSingleTask)
	if err != nil {
		t.Fatalf("validation error: %v", err)
	}

	if hasFinding(result, "V11", SeverityWarning) {
		t.Error("V11 should not flag semver strings like v1.2.3 or v2.0.1")
	}
}

func TestCrossTaskContractMismatch(t *testing.T) {
	graph := map[string]any{
		"version": "0.1.0",
		"tasks": []map[string]any{
			{
				"task_id":   "extract-records",
				"task_name": "Implement the record extractor",
				"goal":      "Records are fetched from the upstream feed and emitted as a list of strings.",
				"inputs": []map[string]string{
					{"name": "feed", "type": "string", "constraints": "len > 0", "source": "Upstream HTTP feed"},
				},
				"outputs": []map[string]string{
					{"name": "records", "type": "list<string>", "constraints": "len >= 0", "destination": "Return value"},
				},
				"acceptance":  []string{"Given a 200 response with 3 items, returns a list of 3 strings"},
				"depends_on":  map[string]string{"status": "N/A", "reason": "Top of pipeline, no upstream tasks"},
				"constraints": []string{"No I/O after fetch"},
				"files_scope": []string{"internal/extract/extract.go"},
			},
			{
				"task_id":   "summarise-records",
				"task_name": "Implement the record summariser",
				"goal":      "A summariser produces one summary string from the extracted records.",
				"inputs": []map[string]string{
					{"name": "records", "type": "string", "constraints": "len > 0", "source": "Output records from extract-records"},
				},
				"outputs": []map[string]string{
					{"name": "summary", "type": "string", "constraints": "len > 0", "destination": "Return value"},
				},
				"acceptance":  []string{"Given 3 input records, returns a non-empty summary string"},
				"depends_on":  []string{"extract-records"},
				"constraints": []string{"Pure function with no I/O"},
				"files_scope": []string{"internal/summarise/summarise.go"},
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

	if !hasFinding(result, "V12", SeverityWarning) {
		t.Error("expected V12 warning for type mismatch (string vs list<string>) on records")
	}
}

func TestCrossTaskContractMatch(t *testing.T) {
	graph := map[string]any{
		"version": "0.1.0",
		"tasks": []map[string]any{
			{
				"task_id":   "extract-records",
				"task_name": "Implement the record extractor",
				"goal":      "Records are fetched from the upstream feed and emitted as a list of strings.",
				"inputs": []map[string]string{
					{"name": "feed", "type": "string", "constraints": "len > 0", "source": "Upstream HTTP feed"},
				},
				"outputs": []map[string]string{
					{"name": "records", "type": "list<string>", "constraints": "len >= 0", "destination": "Return value"},
				},
				"acceptance":  []string{"Given a 200 response with 3 items, returns a list of 3 strings"},
				"depends_on":  map[string]string{"status": "N/A", "reason": "Top of pipeline, no upstream tasks"},
				"constraints": []string{"No I/O after fetch"},
				"files_scope": []string{"internal/extract/extract.go"},
			},
			{
				"task_id":   "summarise-records",
				"task_name": "Implement the record summariser",
				"goal":      "A summariser produces one summary string from the extracted records.",
				"inputs": []map[string]string{
					{"name": "records", "type": "list<string>", "constraints": "len >= 0", "source": "Output records from extract-records"},
				},
				"outputs": []map[string]string{
					{"name": "summary", "type": "string", "constraints": "len > 0", "destination": "Return value"},
				},
				"acceptance":  []string{"Given 3 input records, returns a non-empty summary string"},
				"depends_on":  []string{"extract-records"},
				"constraints": []string{"Pure function with no I/O"},
				"files_scope": []string{"internal/summarise/summarise.go"},
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

	if hasFinding(result, "V12", SeverityWarning) {
		t.Error("did not expect V12 warning when input/output types match")
	}
}

func TestGranularityLargeEstimate(t *testing.T) {
	task := map[string]any{
		"task_id":   "monolith-pipeline",
		"task_name": "Implement the entire payment pipeline",
		"goal":      "The pipeline ingests an order and emits a settled payment record in the ledger.",
		"inputs": []map[string]string{
			{"name": "order", "type": "string", "constraints": "len > 0", "source": "checkout queue"},
		},
		"outputs": []map[string]string{
			{"name": "settlement", "type": "string", "constraints": "len > 0", "destination": "ledger table"},
		},
		"acceptance":  []string{"Given a valid order, the ledger contains a settlement row keyed by order id"},
		"depends_on":  map[string]string{"status": "N/A", "reason": "Top-level pipeline task"},
		"constraints": []string{"No third-party deps beyond stdlib and pgx"},
		"files_scope": []string{"internal/payments/pipeline.go"},
		"estimate":    "large",
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	result, err := Validate(data, ModeSingleTask)
	if err != nil {
		t.Fatalf("validation error: %v", err)
	}

	if !hasFinding(result, "V13", SeverityInfo) {
		t.Error("expected V13 info for estimate 'large'")
	}
}

func TestGranularityManyTasks(t *testing.T) {
	tasks := make([]map[string]any, 0, 21)
	for i := 0; i < 21; i++ {
		tasks = append(tasks, map[string]any{
			"task_id":   fmt.Sprintf("micro-task-%02d", i),
			"task_name": fmt.Sprintf("Implement micro task number %d", i),
			"goal":      "The function returns a deterministic value for any valid input.",
			"inputs": []map[string]string{
				{"name": "x", "type": "string", "constraints": "len > 0", "source": "caller"},
			},
			"outputs": []map[string]string{
				{"name": "y", "type": "string", "constraints": "none", "destination": "Return value"},
			},
			"acceptance":  []string{"Given input 'a', returns 'a-result' deterministically"},
			"depends_on":  map[string]string{"status": "N/A", "reason": "Independent micro-task"},
			"constraints": []string{"Pure function with no I/O"},
			"files_scope": []string{fmt.Sprintf("internal/micro/m%02d.go", i)},
		})
	}
	graph := map[string]any{"version": "0.1.0", "tasks": tasks}

	data, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("marshaling: %v", err)
	}

	result, err := Validate(data, ModeTaskGraph)
	if err != nil {
		t.Fatalf("validation error: %v", err)
	}

	foundManyTasks := false
	for _, e := range result.Errors {
		if e.Rule == "V13" && e.Severity == SeverityInfo && strings.Contains(e.Path, "tasks") &&
			strings.Contains(strings.ToLower(e.Message), "graph") {
			foundManyTasks = true
			break
		}
	}
	if !foundManyTasks {
		t.Error("expected V13 info naming the >20-task graph")
	}
}

func TestGranularityMilestoneOverload(t *testing.T) {
	tasks := make([]map[string]any, 0, 9)
	ids := make([]string, 0, 9)
	for i := 0; i < 9; i++ {
		id := fmt.Sprintf("ms-task-%02d", i)
		ids = append(ids, id)
		tasks = append(tasks, map[string]any{
			"task_id":   id,
			"task_name": fmt.Sprintf("Implement milestone-bound task %d", i),
			"goal":      "The function returns a deterministic value for any valid input.",
			"inputs": []map[string]string{
				{"name": "x", "type": "string", "constraints": "len > 0", "source": "caller"},
			},
			"outputs": []map[string]string{
				{"name": "y", "type": "string", "constraints": "none", "destination": "Return value"},
			},
			"acceptance":  []string{"Given input 'a', returns 'a-result' deterministically"},
			"depends_on":  map[string]string{"status": "N/A", "reason": "Independent task"},
			"constraints": []string{"Pure function with no I/O"},
			"files_scope": []string{fmt.Sprintf("internal/x/m%02d.go", i)},
		})
	}
	graph := map[string]any{
		"version": "0.1.0",
		"tasks":   tasks,
		"milestones": []map[string]any{
			{"name": "M1 - Overloaded", "task_ids": ids},
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

	foundMilestone := false
	for _, e := range result.Errors {
		if e.Rule == "V13" && e.Severity == SeverityInfo && strings.Contains(strings.ToLower(e.Message), "milestone") {
			foundMilestone = true
			break
		}
	}
	if !foundMilestone {
		t.Error("expected V13 info naming 'milestone' for milestone with >8 tasks")
	}
}

func TestMissingDependencyLink(t *testing.T) {
	graph := map[string]any{
		"version": "0.1.0",
		"tasks": []map[string]any{
			{
				"task_id":   "ingest-rows",
				"task_name": "Implement the ingest worker",
				"goal":      "The ingest worker reads rows from the upstream feed and emits them.",
				"inputs": []map[string]string{
					{"name": "feed", "type": "string", "constraints": "len > 0", "source": "Upstream HTTP feed"},
				},
				"outputs": []map[string]string{
					{"name": "rows", "type": "list<string>", "constraints": "len >= 0", "destination": "Return value"},
				},
				"acceptance":  []string{"Given a 200 response with rows, returns the rows as a list"},
				"depends_on":  map[string]string{"status": "N/A", "reason": "Top of pipeline"},
				"constraints": []string{"No retries on 4xx responses"},
				"files_scope": []string{"internal/ingest/ingest.go"},
			},
			{
				"task_id":   "transform-rows",
				"task_name": "Implement the row transformer",
				"goal":      "The transformer normalises each row and emits a list of records.",
				"inputs": []map[string]string{
					{"name": "rows", "type": "list<string>", "constraints": "len >= 0", "source": "Output rows from ingest-rows"},
				},
				"outputs": []map[string]string{
					{"name": "records", "type": "list<string>", "constraints": "len >= 0", "destination": "Return value"},
				},
				"acceptance":  []string{"Given 3 input rows, returns a list of 3 normalised records"},
				"depends_on":  map[string]string{"status": "N/A", "reason": "Wrong: should declare ingest-rows"},
				"constraints": []string{"Pure function with no I/O"},
				"files_scope": []string{"internal/transform/transform.go"},
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

	if !hasFinding(result, "V14", SeverityWarning) {
		t.Error("expected V14 warning for undeclared dependency on ingest-rows")
	}
}

func TestMissingDependencyLinkCorrect(t *testing.T) {
	graph := map[string]any{
		"version": "0.1.0",
		"tasks": []map[string]any{
			{
				"task_id":   "ingest-rows",
				"task_name": "Implement the ingest worker",
				"goal":      "The ingest worker reads rows from the upstream feed and emits them.",
				"inputs": []map[string]string{
					{"name": "feed", "type": "string", "constraints": "len > 0", "source": "Upstream HTTP feed"},
				},
				"outputs": []map[string]string{
					{"name": "rows", "type": "list<string>", "constraints": "len >= 0", "destination": "Return value"},
				},
				"acceptance":  []string{"Given a 200 response with rows, returns the rows as a list"},
				"depends_on":  map[string]string{"status": "N/A", "reason": "Top of pipeline"},
				"constraints": []string{"No retries on 4xx responses"},
				"files_scope": []string{"internal/ingest/ingest.go"},
			},
			{
				"task_id":   "transform-rows",
				"task_name": "Implement the row transformer",
				"goal":      "The transformer normalises each row and emits a list of records.",
				"inputs": []map[string]string{
					{"name": "rows", "type": "list<string>", "constraints": "len >= 0", "source": "Output rows from ingest-rows"},
				},
				"outputs": []map[string]string{
					{"name": "records", "type": "list<string>", "constraints": "len >= 0", "destination": "Return value"},
				},
				"acceptance":  []string{"Given 3 input rows, returns a list of 3 normalised records"},
				"depends_on":  []string{"ingest-rows"},
				"constraints": []string{"Pure function with no I/O"},
				"files_scope": []string{"internal/transform/transform.go"},
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

	if hasFinding(result, "V14", SeverityWarning) {
		t.Error("did not expect V14 warning when dependency IS declared in depends_on")
	}
}

func TestCrossTaskContractOptionalWrap(t *testing.T) {
	graph := map[string]any{
		"version": "0.1.0",
		"tasks": []map[string]any{
			{
				"task_id":   "produce-data",
				"task_name": "Produce data",
				"goal":      "Produce a string output.",
				"inputs":    []map[string]string{},
				"outputs": []map[string]string{
					{"name": "payload", "type": "string", "constraints": "len > 0", "destination": "Return value"},
				},
				"acceptance":  []string{"Returns a non-empty string"},
				"depends_on":  map[string]string{"status": "N/A", "reason": "No upstream"},
				"constraints": []string{},
				"files_scope": []string{"internal/produce.go"},
			},
			{
				"task_id":   "consume-data",
				"task_name": "Consume data",
				"goal":      "Consume the payload, tolerating absence.",
				"inputs": []map[string]string{
					{"name": "payload", "type": "optional<string>", "constraints": "none", "source": "Output payload from produce-data"},
				},
				"outputs": []map[string]string{
					{"name": "result", "type": "string", "constraints": "len > 0", "destination": "Return value"},
				},
				"acceptance":  []string{"Handles missing payload gracefully"},
				"depends_on":  []string{"produce-data"},
				"constraints": []string{},
				"files_scope": []string{"internal/consume.go"},
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

	if hasFinding(result, "V12", SeverityWarning) {
		t.Error("V12 should not flag optional<string> consuming string — optional wrapping is compatible")
	}
}

func TestContainsWord(t *testing.T) {
	cases := []struct {
		s, substr string
		want      bool
	}{
		{"Output rows from ingest-rows", "ingest-rows", true},
		{"Output rows from ingest-rows-v2", "ingest-rows", false},
		{"extract-records produces output", "extract-records", true},
		{"pre-extract-records pipeline", "extract-records", false},
		{"task_one is ready", "task_one", true},
		{"task_one_extended is ready", "task_one", false},
		{"", "anything", false},
		{"something", "", false},
		{"exact", "exact", true},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s_in_%s", tc.substr, tc.s), func(t *testing.T) {
			got := containsWord(tc.s, tc.substr)
			if got != tc.want {
				t.Errorf("containsWord(%q, %q) = %v, want %v", tc.s, tc.substr, got, tc.want)
			}
		})
	}
}
