package utils

import (
	"encoding/json"
	"net/http"
)

// JSONResponse sends a JSON response with the given status code and data
func JSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error but can't send response after status is written
		http.Error(w, "Error encoding JSON response", http.StatusInternalServerError)
	}
}

// JSONError sends a JSON error response
func JSONError(w http.ResponseWriter, status int, message string) {
	JSONResponse(w, status, map[string]string{"error": message})
}

// JSONSuccess sends a JSON success response
func JSONSuccess(w http.ResponseWriter, data interface{}) {
	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

// JSONCreated sends a JSON created response
func JSONCreated(w http.ResponseWriter, data interface{}) {
	JSONResponse(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

// JSONBadRequest sends a JSON bad request response
func JSONBadRequest(w http.ResponseWriter, message string) {
	JSONError(w, http.StatusBadRequest, message)
}

// JSONUnauthorized sends a JSON unauthorized response
func JSONUnauthorized(w http.ResponseWriter, message string) {
	JSONError(w, http.StatusUnauthorized, message)
}

// JSONForbidden sends a JSON forbidden response
func JSONForbidden(w http.ResponseWriter, message string) {
	JSONError(w, http.StatusForbidden, message)
}

// JSONNotFound sends a JSON not found response
func JSONNotFound(w http.ResponseWriter, message string) {
	JSONError(w, http.StatusNotFound, message)
}

// JSONInternalError sends a JSON internal server error response
func JSONInternalError(w http.ResponseWriter, message string) {
	JSONError(w, http.StatusInternalServerError, message)
}

// ParseJSONRequest parses JSON request body into the given struct
func ParseJSONRequest(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// GetContentType returns the content type header
func GetContentType(r *http.Request) string {
	return r.Header.Get("Content-Type")
}

// IsJSONContent checks if the request content type is JSON
func IsJSONContent(r *http.Request) bool {
	return GetContentType(r) == "application/json"
}

// GetBearerToken extracts bearer token from Authorization header
func GetBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}