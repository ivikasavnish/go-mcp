package ide

import (
	"context"
	"sync"
	"time"
)

// FileInfo represents information about a file
type FileInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	IsDir       bool      `json:"is_dir"`
	ModTime     time.Time `json:"mod_time"`
	Permissions string    `json:"permissions"`
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Success       bool          `json:"success"`
	Output        string        `json:"output"`
	Error         string        `json:"error,omitempty"`
	ExitCode      int           `json:"exit_code"`
	ExecutionTime time.Duration `json:"execution_time"`
}

// GitStatus represents the status of a git repository
type GitStatus struct {
	Branch           string    `json:"branch"`
	IsClean          bool      `json:"is_clean"`
	Modified         []string  `json:"modified"`
	Untracked        []string  `json:"untracked"`
	Staged           []string  `json:"staged"`
	RemoteStatus     string    `json:"remote_status"`
	LastCommit       string    `json:"last_commit"`
	LastCommitAuthor string    `json:"last_commit_author"`
	LastCommitDate   time.Time `json:"last_commit_date"`
}

// ProjectConfig represents project configuration
type ProjectConfig struct {
	Name         string            `json:"name"`
	Root         string            `json:"root"`
	BuildCommand string            `json:"build_command"`
	RunCommand   string            `json:"run_command"`
	TestCommand  string            `json:"test_command"`
	Environment  map[string]string `json:"environment"`
	GitEnabled   bool              `json:"git_enabled"`
}
type Task struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Command     string `json:"command"`
	AutoRestart bool   `json:"auto_restart"`
	Status      string `json:"status"`
}

// TaskManager handles long-running development tasks
type TaskManager struct {
	tasks  map[string]*Task
	cancel map[string]context.CancelFunc
	mu     sync.RWMutex
}
