package handlers

import (
	"encoding/json"
	"net/http"
)

type BaseHandler struct{}

func (h *BaseHandler) ParseJSON(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		h.ErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid JSON", nil)
		return false
	}
	return true
}

func (h *BaseHandler) JSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *BaseHandler) SuccessResponse(w http.ResponseWriter, data interface{}, message string) {
	response := map[string]interface{}{
		"success": true,
		"data":    data,
		"message": message,
	}
	h.JSONResponse(w, http.StatusOK, response)
}

func (h *BaseHandler) ErrorResponse(w http.ResponseWriter, status int, code, message string, details map[string]interface{}) {
	response := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"details": details,
		},
	}
	h.JSONResponse(w, status, response)
}

func (h *BaseHandler) GetUserID(r *http.Request) string {
	// This will be populated by JWT middleware
	userID := r.Context().Value("user_id")
	if userID == nil {
		return ""
	}
	return userID.(string)
}


