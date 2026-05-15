package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mojitrk/sica/internal/core"
	"github.com/mojitrk/sica/internal/tasks"
)

type TasksHandler struct {
	Store *tasks.Store
}

func (h *TasksHandler) List(w http.ResponseWriter, r *http.Request) {
	f := tasks.Filter{
		Status:   r.URL.Query().Get("status"),
		Priority: r.URL.Query().Get("priority"),
	}
	if pid := r.URL.Query().Get("project_id"); pid != "" {
		id, _ := strconv.ParseInt(pid, 10, 64)
		f.ProjectID = &id
	}
	taskList, err := h.Store.ListTasks(f)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if taskList == nil {
		taskList = []core.Task{}
	}
	respond(w, http.StatusOK, taskList)
}

func (h *TasksHandler) Create(w http.ResponseWriter, r *http.Request) {
	var task core.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		respondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if task.Status == "" {
		task.Status = "todo"
	}
	if task.Priority == "" {
		task.Priority = "med"
	}
	if err := h.Store.CreateTask(&task); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, task)
}

func (h *TasksHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	task, err := h.Store.GetTask(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if task == nil {
		respondError(w, http.StatusNotFound, "task not found")
		return
	}
	respond(w, http.StatusOK, task)
}

func (h *TasksHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var task core.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		respondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	task.ID = id
	if err := h.Store.UpdateTask(&task); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, task)
}

func (h *TasksHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.Store.CompleteTask(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *TasksHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.Store.DeleteTask(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusNoContent, nil)
}

func (h *TasksHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.Store.ListProjects()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if projects == nil {
		projects = []core.Project{}
	}
	respond(w, http.StatusOK, projects)
}

func (h *TasksHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var project core.Project
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		respondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.Store.CreateProject(&project); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, project)
}
