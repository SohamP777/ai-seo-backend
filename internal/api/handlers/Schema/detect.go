package handlers

import (
	"encoding/json"
	"net/http"
	
	"ai-seo-backend/internal/core/fixer"
)

// DetectHandler handles /api/schema/detect
func DetectHandler(client *schema.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL string `json:"url"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		generator := schema.NewGenerator(client)
		detection, err := generator.DetectPageType(req.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detection)
	}
}