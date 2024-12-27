package mcp

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/ivikasavnish/go-mcp/pkg/ide"
	"net/http"
)

// Additional request/response types
type UpdateProjectConfigRequest struct {
	Config ide.ProjectConfig `json:"config"`
}

type CreateTaskRequest struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	AutoRestart bool   `json:"auto_restart"`
}

// IDE server extension
type IDEServer struct {
	projectManager *ide.ProjectManager
	taskManager    *ide.TaskManager
}

func (s IDEServer) NewCommandExecutor(root string) interface{} {
	return ide.NewCommandExecutor(root)
}

func NewIDEServer(projectRoot string) (*IDEServer, error) {
	pm, err := ide.NewProjectManager(projectRoot)
	if err != nil {
		return nil, err
	}

	return &IDEServer{
		projectManager: pm,
		taskManager:    ide.NewTaskManager(),
	}, nil
}

func (s *Server) AddIDEServer(ideServer *IDEServer) {
	// Project management
	s.router.HandleFunc("/ide/project/config", handleGetProjectConfig(ideServer)).Methods("GET")
	s.router.HandleFunc("/ide/project/config", handleUpdateProjectConfig(ideServer)).Methods("PUT")

	// Task management
	s.router.HandleFunc("/ide/tasks", handleListTasks(ideServer)).Methods("GET")
	//s.router.HandleFunc("/ide/tasks", handleCreateTask(ideServer)).Methods("POST")
	s.router.HandleFunc("/ide/tasks/{id}", handleGetTask(ideServer)).Methods("GET")
	s.router.HandleFunc("/ide/tasks/{id}", handleStopTask(ideServer)).Methods("DELETE")
}

// Project config handlers
func handleGetProjectConfig(ide *IDEServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		config := ide.projectManager.GetConfig()
		writeJSON(w, http.StatusOK, config)
	}
}

func handleUpdateProjectConfig(ide *IDEServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req UpdateProjectConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		if err := ide.projectManager.UpdateConfig(&req.Config); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, req.Config)
	}
}

// Task management handlers
//func handleCreateTask(ide *IDEServer) http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		var req CreateTaskRequest
//		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//			writeError(w, http.StatusBadRequest, err)
//			return
//		}
//
//		taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
//		task := &ide.Task{
//			ID:          taskID,
//			Name:        req.Name,
//			Command:     req.Command,
//			AutoRestart: req.AutoRestart,
//			Status:      "starting",
//		}
//
//		task := ide.taskManager.StartTask()
//
//		config := ide.projectManager.GetConfig()
//		executor := ide.NewCommandExecutor(config.Root)
//		if err := ide.taskManager.StartTask(task, executor); err != nil {
//			writeError(w, http.StatusInternalServerError, err)
//			return
//		}
//
//		writeJSON(w, http.StatusCreated, task)
//	}
//}

func handleStopTask(ide *IDEServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := mux.Vars(r)["id"]
		if err := ide.taskManager.StopTask(taskID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "stopped",
			"id":     taskID,
		})
	}
}

func handleGetTask(ide *IDEServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := mux.Vars(r)["id"]
		task := ide.taskManager.GetTask(taskID)
		if task == nil {
			writeError(w, http.StatusNotFound, fmt.Errorf("task not found"))
			return
		}

		writeJSON(w, http.StatusOK, task)
	}
}

func handleListTasks(ide *IDEServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tasks := ide.taskManager.ListTasks()
		writeJSON(w, http.StatusOK, tasks)
	}
}
