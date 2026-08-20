package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	
	"ai-seo-backend/internal/core/fixer"
)

// InjectHandler handles /api/schema/inject
func InjectHandler(client *schema.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req schema.InjectRequest
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		injector := schema.NewInjector(client)
		result, err := injector.Inject(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// ValidateHandler handles /api/schema/validate
func ValidateHandler(client *schema.Client, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SchemaJSON string `json:"schema_json"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		validator := schema.NewValidator(client, apiKey)
		result, err := validator.Validate(req.SchemaJSON)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// RollbackHandler handles /api/schema/rollback/{injection_id}
func RollbackHandler(backup *schema.BackupManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract injection ID from URL
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 5 {
			http.Error(w, "Missing injection ID", http.StatusBadRequest)
			return
		}
		injectionID := parts[4]
		
		rollbackManager := schema.NewRollbackManager(backup)
		result, err := rollbackManager.Rollback(injectionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}