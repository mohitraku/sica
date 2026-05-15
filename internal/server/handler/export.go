package handler

import "net/http"

// Respond writes a JSON response. Exported for use by the server package.
func Respond(w http.ResponseWriter, status int, data interface{}) {
	respond(w, status, data)
}
