package beads

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// PreFlightCheck verifies that bd is available and beads is initialized.
// Returns a user-friendly error message if either check fails.
func PreFlightCheck() error {
	// Check bd is on PATH.
	bdPath, err := exec.LookPath("bd")
	if err != nil {
		return fmt.Errorf("bd not found on PATH. Install beads: go install github.com/steveyegge/beads/cmd/bd@latest")
	}

	// Check beads is initialized.
	cmd := exec.Command(bdPath, "list", "--limit", "0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if strings.Contains(errMsg, "no beads database") {
			return fmt.Errorf("beads not initialized. Run 'bd init' first")
		}
		return fmt.Errorf("bd pre-flight check failed: %s", errMsg)
	}

	return nil
}

// ExecuteCommands runs the bd commands and builds the CreationResult.
// Commands are executed sequentially. Placeholder IDs in later commands
// are replaced with actual IDs from earlier create commands.
func ExecuteCommands(cmds []BdCommand) (*CreationResult, error) {
	result := &CreationResult{
		TaskIDs:    make(map[string]string),
		TaskTitles: make(map[string]string),
	}

	// ID replacement map: placeholder -> actual bd ID.
	idMap := make(map[string]string)

	for _, cmd := range cmds {
		// Replace placeholder IDs with actual IDs.
		args := replaceIDs(cmd.Args, idMap)

		// Execute the command.
		bdID, err := runBdCommand(args)
		if err != nil {
			// Report partial results.
			return result, fmt.Errorf("bd command failed: bd %s\n  Error: %w\n  %d issues created before failure",
				strings.Join(args, " "), err, result.Created)
		}

		// Record results based on command type.
		switch cmd.Type {
		case "create-epic":
			result.EpicID = bdID
			// Extract title from args.
			for i, a := range cmd.Args {
				if a == "--title" && i+1 < len(cmd.Args) {
					result.EpicTitle = cmd.Args[i+1]
					break
				}
			}
			idMap["<epic-id>"] = bdID
			result.Created++

		case "create-task":
			result.TaskIDs[cmd.TaskID] = bdID
			// Extract title from args.
			for i, a := range cmd.Args {
				if a == "--title" && i+1 < len(cmd.Args) {
					result.TaskTitles[cmd.TaskID] = cmd.Args[i+1]
					break
				}
			}
			idMap["<"+cmd.TaskID+"-id>"] = bdID
			result.Created++

		case "dep-add":
			result.Deps++
			result.DepsDetail = append(result.DepsDetail, DepLink{
				TaskBdID: idMap["<"+cmd.DepTaskID+"-id>"],
				DepBdID:  idMap["<"+cmd.DepOnID+"-id>"],
			})

		case "update-design":
			// No counting needed, just record the command.
		}

		result.Commands = append(result.Commands, "bd "+strings.Join(args, " "))
	}

	return result, nil
}

// runBdCommand executes a single bd command and returns the issue ID (from --silent output).
func runBdCommand(args []string) (string, error) {
	cmd := exec.Command("bd", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("%s", errMsg)
	}

	// For create commands with --silent, stdout contains just the issue ID.
	id := strings.TrimSpace(stdout.String())
	return id, nil
}

// replaceIDs substitutes placeholder IDs with actual IDs in command arguments.
func replaceIDs(args []string, idMap map[string]string) []string {
	replaced := make([]string, len(args))
	for i, a := range args {
		replaced[i] = a
		for placeholder, actual := range idMap {
			if strings.Contains(a, placeholder) {
				replaced[i] = strings.ReplaceAll(replaced[i], placeholder, actual)
			}
		}
	}
	return replaced
}
