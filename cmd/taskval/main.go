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
// Exit codes:
//
//	0   Validation passed (no errors; warnings may be present)
//	1   Validation failed (one or more errors)
//	2   Usage error or internal error
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/foundry-zero/task-templating/internal/validator"
)

func main() {
	os.Exit(run())
}

func run() int {
	mode := flag.String("mode", "graph", "Validation mode: 'task' for a single task node, 'graph' for a full task graph")
	output := flag.String("output", "text", "Output format: 'text' for human/LLM-readable, 'json' for machine-readable")
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
		fmt.Fprintf(os.Stderr, "  2  Usage or internal error\n")
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

	// Read input.
	data, err := readInput(flag.Args())
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

	// Output results.
	switch *output {
	case "json":
		outputJSON(result)
	case "text":
		outputText(result)
	}

	if !result.Valid {
		return 1
	}
	return 0
}

func readInput(args []string) ([]byte, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no input file specified. Use 'taskval <file.json>' or 'taskval -' for stdin")
	}

	if len(args) > 1 {
		return nil, fmt.Errorf("expected exactly one input file, got %d", len(args))
	}

	filename := args[0]
	if filename == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
		return data, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file '%s': %w", filename, err)
	}
	return data, nil
}

func outputJSON(result *validator.ValidationResult) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(result)
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
