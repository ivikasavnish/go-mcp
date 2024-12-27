// pkg/ide/project.go
package ide

import (
	"context"
	"encoding/json"
	"fmt"
	_ "github.com/gorilla/mux"
	"io/ioutil"
	_ "net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ProjectManager handles project-related operations
type ProjectManager struct {
	config      *ProjectConfig
	fileManager *FileManager
	cmdExecutor *CommandExecutor
	gitManager  *GitManager
	configPath  string
	mu          sync.RWMutex
}

func NewProjectManager(rootDir string) (*ProjectManager, error) {
	pm := &ProjectManager{
		fileManager: NewFileManager(rootDir),
		cmdExecutor: NewCommandExecutor(rootDir),
		gitManager:  NewGitManager(rootDir),
		configPath:  filepath.Join(rootDir, ".mcp", "project.json"),
	}

	if err := pm.loadConfig(); err != nil {
		// Create default config if not exists
		pm.config = &ProjectConfig{
			Name:         filepath.Base(rootDir),
			Root:         rootDir,
			BuildCommand: "go build",
			RunCommand:   "go run .",
			TestCommand:  "go test ./...",
			Environment:  make(map[string]string),
			GitEnabled:   true,
		}
		if err := pm.saveConfig(); err != nil {
			return nil, err
		}
	}

	return pm, nil
}

func (pm *ProjectManager) loadConfig() error {
	data, err := ioutil.ReadFile(pm.configPath)
	if err != nil {
		return err
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	return json.Unmarshal(data, &pm.config)
}

func (pm *ProjectManager) saveConfig() error {
	pm.mu.RLock()
	data, err := json.MarshalIndent(pm.config, "", "    ")
	pm.mu.RUnlock()

	if err != nil {
		return err
	}

	dir := filepath.Dir(pm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(pm.configPath, data, 0644)
}

func (pm *ProjectManager) UpdateConfig(config *ProjectConfig) error {
	pm.mu.Lock()
	pm.config = config
	pm.mu.Unlock()

	return pm.saveConfig()
}

func (pm *ProjectManager) GetConfig() *ProjectConfig {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.config
}

// Task represents a development task

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks:  make(map[string]*Task),
		cancel: make(map[string]context.CancelFunc),
	}
}

func (tm *TaskManager) StartTask(task *Task, executor *CommandExecutor) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tasks[task.ID]; exists {
		return fmt.Errorf("task %s already running", task.ID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	tm.tasks[task.ID] = task
	tm.cancel[task.ID] = cancel

	go func() {
		for {
			select {
			case <-ctx.Done():
				tm.mu.Lock()
				task.Status = "stopped"
				tm.mu.Unlock()
				return
			default:
				_, err := executor.Execute(ctx, task.Command)
				if err != nil {
					tm.mu.Lock()
					task.Status = fmt.Sprintf("error: %v", err)
					tm.mu.Unlock()
					if !task.AutoRestart {
						return
					}
				}
				if !task.AutoRestart {
					tm.mu.Lock()
					task.Status = "completed"
					tm.mu.Unlock()
					return
				}
				time.Sleep(time.Second) // Prevent rapid restarts
			}
		}
	}()

	return nil
}

func (tm *TaskManager) StopTask(taskID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	cancel, exists := tm.cancel[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	cancel()
	delete(tm.cancel, taskID)
	return nil
}

func (tm *TaskManager) GetTask(taskID string) *Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.tasks[taskID]
}

func (tm *TaskManager) ListTasks() []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}
