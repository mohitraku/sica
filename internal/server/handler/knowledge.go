package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mojitrk/sica/internal/knowledge"
)

type KnowledgeHandler struct {
	Store *knowledge.Store
}

func (h *KnowledgeHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL  string   `json:"url"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "url is required")
		return
	}

	result, err := knowledge.IngestURL(req.URL)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	doc, err := h.Store.Save(result.Title, result.Body, result.SourceURL, req.Tags)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusCreated, doc)
}

func (h *KnowledgeHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		docs, err := h.Store.List()
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respond(w, http.StatusOK, docs)
		return
	}

	docs, err := h.Store.Search(q)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, docs)
}

func (h *KnowledgeHandler) GetDoc(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	content, err := h.Store.ReadContent(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if content == "" {
		respondError(w, http.StatusNotFound, "doc not found")
		return
	}

	doc, _ := h.Store.Get(id)
	respond(w, http.StatusOK, map[string]interface{}{
		"doc":     doc,
		"content": content,
	})
}

func (h *KnowledgeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.Store.Delete(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusNoContent, nil)
}
