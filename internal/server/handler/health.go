package handler

import (
	"net/http"
)

// Health returns a simple health check response.
func Health(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}
