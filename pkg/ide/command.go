package ide

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CommandExecutor handles command execution
type CommandExecutor struct {
	workDir string
	env     map[string]string
}

func NewCommandExecutor(workDir string) *CommandExecutor {
	return &CommandExecutor{
		workDir: workDir,
		env:     make(map[string]string),
	}
}

func (ce *CommandExecutor) SetEnv(key, value string) {
	ce.env[key] = value
}

func (ce *CommandExecutor) Execute(ctx context.Context, command string) (*CommandResult, error) {
	start := time.Now()

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = ce.workDir

	// Setup environment
	env := os.Environ()
	for k, v := range ce.env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	result := &CommandResult{
		Success:       err == nil,
		Output:        stdout.String(),
		Error:         stderr.String(),
		ExitCode:      cmd.ProcessState.ExitCode(),
		ExecutionTime: duration,
	}

	return result, nil
}
