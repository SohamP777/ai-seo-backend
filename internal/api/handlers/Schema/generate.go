package handlers

import (
	"encoding/json"
	"net/http"
	
	"ai-seo-backend/internal/core/fixer"
)

// GenerateHandler handles /api/schema/generate
func GenerateHandler(client *schema.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req schema.GenerateRequest
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		generator := schema.NewGenerator(client)
		response, err := generator.Generate(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}