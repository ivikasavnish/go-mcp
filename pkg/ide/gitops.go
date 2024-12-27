package ide

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// GitManager handles git operations
type GitManager struct {
	executor *CommandExecutor
}

func NewGitManager(workDir string) *GitManager {
	return &GitManager{
		executor: NewCommandExecutor(workDir),
	}
}

func (gm *GitManager) GetStatus() (*GitStatus, error) {
	ctx := context.Background()

	// Get current branch
	branchResult, err := gm.executor.Execute(ctx, "git rev-parse --abbrev-ref HEAD")
	if err != nil {
		return nil, err
	}

	// Get status
	statusResult, err := gm.executor.Execute(ctx, "git status --porcelain")
	if err != nil {
		return nil, err
	}

	// Get last commit info
	commitResult, err := gm.executor.Execute(ctx, "git log -1 --format=%H%n%an%n%at")
	if err != nil {
		return nil, err
	}

	status := &GitStatus{
		Branch:    strings.TrimSpace(branchResult.Output),
		IsClean:   statusResult.Output == "",
		Modified:  []string{},
		Untracked: []string{},
		Staged:    []string{},
	}

	// Parse status output
	for _, line := range strings.Split(statusResult.Output, "\n") {
		if len(line) < 3 {
			continue
		}

		state := line[:2]
		file := line[3:]

		switch {
		case state == "M ":
			status.Modified = append(status.Modified, file)
		case state == "??":
			status.Untracked = append(status.Untracked, file)
		case state == "A ":
			status.Staged = append(status.Staged, file)
		}
	}

	// Parse commit info
	commitInfo := strings.Split(commitResult.Output, "\n")
	if len(commitInfo) >= 3 {
		status.LastCommit = commitInfo[0]
		status.LastCommitAuthor = commitInfo[1]
		timestamp, _ := strconv.ParseInt(strings.TrimSpace(commitInfo[2]), 10, 64)
		status.LastCommitDate = time.Unix(timestamp, 0)
	}

	return status, nil
}

func (gm *GitManager) Pull() error {
	_, err := gm.executor.Execute(context.Background(), "git pull")
	return err
}

func (gm *GitManager) Push() error {
	_, err := gm.executor.Execute(context.Background(), "git push")
	return err
}

func (gm *GitManager) Commit(message string) error {
	ctx := context.Background()

	// Stage all changes
	_, err := gm.executor.Execute(ctx, "git add .")
	if err != nil {
		return err
	}

	// Commit
	_, err = gm.executor.Execute(ctx, fmt.Sprintf("git commit -m %q", message))
	return err
}
