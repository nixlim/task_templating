// taskval validates task definitions against the Structured Task Template Spec.
//
// Usage:
//
//	taskval [flags] <file.json>
//	taskval --mode=task <single_task.json>
//	taskval --mode=graph <task_graph.json>
//	cat task.json | taskval --mode=task -
//
// Output format:
//
//	--output=text   Human/LLM-readable text (default)
//	--output=json   Machine-readable JSON
//
// Beads integration:
//
//	--create-beads  On validation success, create Beads issues via bd CLI
//	--dry-run       Show bd commands that would be executed (requires --create-beads)
//	--epic-title    Override the auto-generated epic title (graph mode only)
//
// Exit codes:
//
//	0   Validation passed (no errors; warnings may be present)
//	1   Validation failed (one or more errors)
//	2   Usage error, internal error, or bd command failure
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nixlim/task_templating/internal/beads"
	"github.com/nixlim/task_templating/internal/validator"
)

func main() {
	os.Exit(run())
}

func run() int {
	mode := flag.String("mode", "graph", "Validation mode: 'task' for a single task node, 'graph' for a full task graph")
	output := flag.String("output", "text", "Output format: 'text' for human/LLM-readable, 'json' for machine-readable")
	createBeads := flag.Bool("create-beads", false, "On validation success, create Beads issues via bd CLI")
	dryRun := flag.Bool("dry-run", false, "Show bd commands that would be executed (requires --create-beads)")
	epicTitle := flag.String("epic-title", "", "Override the auto-generated epic title (graph mode only)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "taskval — Structured Task Template Spec validator\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  taskval [flags] <file.json>\n")
		fmt.Fprintf(os.Stderr, "  taskval [flags] -          (read from stdin)\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExit codes:\n")
		fmt.Fprintf(os.Stderr, "  0  Validation passed (no errors)\n")
		fmt.Fprintf(os.Stderr, "  1  Validation failed (errors found)\n")
		fmt.Fprintf(os.Stderr, "  2  Usage, internal, or bd error\n")
	}
	flag.Parse()

	// Validate flags.
	var valMode validator.Mode
	switch *mode {
	case "task":
		valMode = validator.ModeSingleTask
	case "graph":
		valMode = validator.ModeTaskGraph
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid mode '%s'. Must be 'task' or 'graph'.\n", *mode)
		return 2
	}

	if *output != "text" && *output != "json" {
		fmt.Fprintf(os.Stderr, "Error: invalid output format '%s'. Must be 'text' or 'json'.\n", *output)
		return 2
	}

	if *dryRun && !*createBeads {
		fmt.Fprintf(os.Stderr, "Error: --dry-run requires --create-beads.\n")
		return 2
	}

	// Read input.
	data, filename, err := readInput(flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return 2
	}

	// Run validation.
	result, err := validator.Validate(data, valMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Internal error: %s\n", err)
		return 2
	}

	// Output validation results.
	if *output == "text" {
		outputText(result)
	}

	if !result.Valid {
		if *output == "json" {
			outputJSON(result, nil)
		}
		return 1
	}

	// If --create-beads, proceed to beads creation.
	if *createBeads {
		exitCode := runBeadsCreation(result, valMode, *dryRun, *epicTitle, filename, *output)
		if exitCode != 0 {
			return exitCode
		}
	} else if *output == "json" {
		outputJSON(result, nil)
	}

	return 0
}

// runBeadsCreation handles the beads creation pipeline after successful validation.
func runBeadsCreation(result *validator.ValidationResult, mode validator.Mode, dryRun bool, epicTitle, filename, output string) int {
	if result.Graph == nil {
		fmt.Fprintf(os.Stderr, "Internal error: validation passed but no parsed graph available\n")
		return 2
	}

	// Pre-flight check (skip for dry-run since we don't execute commands).
	if !dryRun {
		if err := beads.PreFlightCheck(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			return 2
		}
	}

	creator := &beads.Creator{
		DryRun:    dryRun,
		EpicTitle: epicTitle,
		Filename:  filename,
	}

	// Build commands.
	var cmds []beads.BdCommand
	var err error

	switch mode {
	case validator.ModeSingleTask:
		if len(result.Graph.Tasks) == 0 {
			fmt.Fprintf(os.Stderr, "Internal error: graph has no tasks\n")
			return 2
		}
		cmds, err = creator.BuildSingleTaskCommands(&result.Graph.Tasks[0])
	case validator.ModeTaskGraph:
		cmds, err = creator.BuildGraphCommands(result.Graph)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building commands: %s\n", err)
		return 2
	}

	// Dry-run: print commands and exit.
	if dryRun {
		fmt.Print(beads.FormatDryRunOutput(cmds))
		if output == "json" {
			outputJSON(result, nil)
		}
		return 0
	}

	// Execute commands.
	creationResult, err := beads.ExecuteCommands(cmds)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		if creationResult != nil && output == "text" {
			fmt.Print(beads.FormatTextOutput(creationResult))
		}
		return 2
	}

	// Output beads creation results.
	switch output {
	case "text":
		fmt.Print(beads.FormatTextOutput(creationResult))
	case "json":
		outputJSON(result, beads.FormatJSONOutput(creationResult))
	}

	return 0
}

func readInput(args []string) ([]byte, string, error) {
	if len(args) == 0 {
		return nil, "", fmt.Errorf("no input file specified. Use 'taskval <file.json>' or 'taskval -' for stdin")
	}

	if len(args) > 1 {
		return nil, "", fmt.Errorf("expected exactly one input file, got %d", len(args))
	}

	filename := args[0]
	if filename == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, "-", fmt.Errorf("reading stdin: %w", err)
		}
		return data, "-", nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, filename, fmt.Errorf("reading file '%s': %w", filename, err)
	}
	return data, filename, nil
}

// combinedOutput holds validation result plus optional beads creation result for JSON output.
type combinedOutput struct {
	Valid  bool                        `json:"valid"`
	Errors []validator.ValidationError `json:"errors,omitempty"`
	Stats  validator.ValidationStats   `json:"stats"`
	Beads  *beads.BeadsJSON            `json:"beads,omitempty"`
}

func outputJSON(result *validator.ValidationResult, beadsResult *beads.BeadsJSON) {
	out := combinedOutput{
		Valid:  result.Valid,
		Errors: result.Errors,
		Stats:  result.Stats,
		Beads:  beadsResult,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func outputText(result *validator.ValidationResult) {
	if result.Valid && result.Stats.WarningCount == 0 && result.Stats.InfoCount == 0 {
		fmt.Println("VALIDATION PASSED")
		fmt.Printf("  Tasks validated: %d\n", result.Stats.TotalTasks)
		fmt.Println("  No errors or warnings.")
		return
	}

	if result.Valid {
		fmt.Println("VALIDATION PASSED (with warnings)")
	} else {
		fmt.Println("VALIDATION FAILED")
	}

	fmt.Printf("\nSummary: %d error(s), %d warning(s), %d info(s) across %d task(s)\n",
		result.Stats.ErrorCount,
		result.Stats.WarningCount,
		result.Stats.InfoCount,
		result.Stats.TotalTasks,
	)

	// Group errors by severity for readability.
	if result.Stats.ErrorCount > 0 {
		fmt.Println("\n--- ERRORS (must fix) ---")
		for i, e := range result.Errors {
			if e.Severity != validator.SeverityError {
				continue
			}
			printError(i+1, e)
		}
	}

	if result.Stats.WarningCount > 0 {
		fmt.Println("\n--- WARNINGS (should fix) ---")
		for i, e := range result.Errors {
			if e.Severity != validator.SeverityWarning {
				continue
			}
			printError(i+1, e)
		}
	}

	if result.Stats.InfoCount > 0 {
		fmt.Println("\n--- INFO ---")
		for i, e := range result.Errors {
			if e.Severity != validator.SeverityInfo {
				continue
			}
			printError(i+1, e)
		}
	}
}

func printError(num int, e validator.ValidationError) {
	fmt.Printf("\n  %d. [%s] Rule %s\n", num, e.Severity, e.Rule)
	fmt.Printf("     Path:    %s\n", e.Path)
	fmt.Printf("     Problem: %s\n", wrapText(e.Message, 14, 80))
	if e.Suggestion != "" {
		fmt.Printf("     Fix:     %s\n", wrapText(e.Suggestion, 14, 80))
	}
	if e.Context != "" {
		ctx := e.Context
		if len(ctx) > 120 {
			ctx = ctx[:117] + "..."
		}
		fmt.Printf("     Value:   %q\n", ctx)
	}
}

// wrapText wraps long text at the given line width, with indent for continuation lines.
func wrapText(text string, indent, width int) string {
	if len(text) <= width-indent {
		return text
	}

	words := strings.Fields(text)
	var lines []string
	current := ""
	maxLen := width - indent

	for _, word := range words {
		if current == "" {
			current = word
			continue
		}
		if len(current)+1+len(word) > maxLen {
			lines = append(lines, current)
			current = word
		} else {
			current += " " + word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}

	if len(lines) <= 1 {
		return text
	}

	pad := strings.Repeat(" ", indent)
	return lines[0] + "\n" + pad + strings.Join(lines[1:], "\n"+pad)
}
