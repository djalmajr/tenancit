package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const maxJSONBody int64 = 1 << 20 // 1 MiB

// decodeJSON accepts one JSON value, keeps unknown fields forward-compatible,
// and ensures even trailing input is consumed under the same size limit.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.ContentLength > maxJSONBody {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body too large"})
		return false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		writeDecodeError(w, err)
		return false
	}

	var trailing any
	err := dec.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return true
	}
	if err != nil {
		writeDecodeError(w, err)
		return false
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	return false
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body too large"})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
}
