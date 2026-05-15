package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mojitrk/sica/internal/core"
	"github.com/mojitrk/sica/internal/habits"
)

type HabitsHandler struct {
	Store *habits.Store
}

func (h *HabitsHandler) List(w http.ResponseWriter, r *http.Request) {
	habits, err := h.Store.List(false)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if habits == nil {
		habits = []core.Habit{}
	}
	respond(w, http.StatusOK, habits)
}

func (h *HabitsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var habit core.Habit
	if err := json.NewDecoder(r.Body).Decode(&habit); err != nil {
		respondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if habit.Frequency == "" {
		habit.Frequency = "daily"
	}
	if err := h.Store.Create(&habit); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, habit)
}

func (h *HabitsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var habit core.Habit
	if err := json.NewDecoder(r.Body).Decode(&habit); err != nil {
		respondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	habit.ID = id
	if err := h.Store.Update(&habit); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, habit)
}

func (h *HabitsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.Store.Delete(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusNoContent, nil)
}

func (h *HabitsHandler) AddEntry(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var entry struct {
		Date  string `json:"date"`
		Value int    `json:"value"`
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		respondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if entry.Value == 0 {
		entry.Value = 1
	}
	if err := h.Store.AddEntry(id, entry.Date, entry.Value, entry.Notes); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HabitsHandler) RemoveEntry(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var entry struct {
		Date string `json:"date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		respondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.Store.RemoveEntry(id, entry.Date); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusNoContent, nil)
}

func (h *HabitsHandler) Stats(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	stats, err := h.Store.Stats(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if stats == nil {
		respondError(w, http.StatusNotFound, "habit not found")
		return
	}
	respond(w, http.StatusOK, stats)
}
