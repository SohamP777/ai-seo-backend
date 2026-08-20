// api/handlers.go
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"ai-seo-backend/internal/core/fixer"
)

type CDNHandler struct {
    fixer           *fixer.CloudflareFixer
    rollbackManager *fixer.RollbackManager
}

// AnalyzeRequest represents the analyze API request
type AnalyzeRequest struct {
	ZoneID string `json:"zone_id"`
}

// AnalyzeResponse represents the analyze API response
type AnalyzeResponse struct {
	ZoneID         string   `json:"zone_id"`
	ZoneName       string   `json:"zone_name"`
	CurrentScore   int      `json:"current_score"`
	Recommendations []string `json:"recommendations"`
}

// ConnectRequest represents the connect API request
type ConnectRequest struct {
	APIToken string `json:"api_token"`
	APIKey   string `json:"api_key"`
	Email    string `json:"email"`
}

// ConnectResponse represents the connect API response - FIXED: use Zone type
type ConnectResponse struct {
    Zones []fixer.Zone `json:"zones"`  // Changed from CloudflareClient to Zone
}

// FixRequest represents the fix API request
type FixRequest struct {
	ZoneID          string `json:"zone_id"`
	DryRun          bool   `json:"dry_run"`
	EnableSpeed     bool   `json:"enable_speed"`
	EnableCache     bool   `json:"enable_cache"`
	EnableSSL       bool   `json:"enable_ssl"`
	EnableSecurity  bool   `json:"enable_security"`
	CreateBackup    bool   `json:"create_backup"`
	EnablePageRules bool   `json:"enable_page_rules"`
	SetupDNS        bool   `json:"setup_dns"`
	OriginServerIP  string `json:"origin_server_ip"`
}

type FixOptions struct {
    DryRun          bool
    EnableSpeed     bool
    EnableCache     bool
    EnableSSL       bool
    EnableSecurity  bool
    CreateBackup    bool
    EnablePageRules bool
    SetupDNS        bool
    OriginServerIP  string  // ADDED missing field
}

// RollbackRequest represents the rollback API request
type RollbackRequest struct {
	ZoneID   string `json:"zone_id"`
	BackupID string `json:"backup_id"`
}

func (h *CDNHandler) HandleAnalyze(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var req AnalyzeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    zone, err := h.fixer.GetZone(r.Context(), req.ZoneID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    response := AnalyzeResponse{
        ZoneID:         zone.ID,
        ZoneName:       zone.Name,
        CurrentScore:   65,
        Recommendations: []string{
            "Enable Rocket Loader for faster JavaScript loading",
            "Enable Polish for image optimization",
            "Set browser cache TTL to 1 month",
            "Enable HTTP/3 for faster connections",
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// HandleConnect connects to Cloudflare account
func (h *CDNHandler) HandleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	var client *fixer.CloudflareClient
	
	if req.APIToken != "" {
		client = fixer.NewCloudflareClientWithToken(req.APIToken)
	} else if req.APIKey != "" && req.Email != "" {
		client = fixer.NewCloudflareClientWithKey(req.APIKey, req.Email)
	} else {
		http.Error(w, "Either API token or API key with email required", http.StatusBadRequest)
		return
	}
	
	zones, err := client.ListZones(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := ConnectResponse{
        Zones: zones,
    }
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleFix applies CDN optimizations - FIXED: Fix returns 1 value
func (h *CDNHandler) HandleFix(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req FixRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	opts := &fixer.FixOptions{
		DryRun:          req.DryRun,
		EnableSpeed:     req.EnableSpeed,
		EnableCache:     req.EnableCache,
		EnableSSL:       req.EnableSSL,
		EnableSecurity:  req.EnableSecurity,
		CreateBackup:    req.CreateBackup,
		EnablePageRules: req.EnablePageRules,
		SetupDNS:        req.SetupDNS,
		OriginServerIP:  req.OriginServerIP,
	}
	
	// Fix returns 1 value (error), not 2
	err := h.fixer.Fix(r.Context(), req.ZoneID, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"success": true,
		"message": "CDN optimizations applied successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleRollback restores previous configuration - FIXED: Rollback returns 1 value
func (h *CDNHandler) HandleRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req RollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Rollback returns 1 value (error)
	err := h.rollbackManager.Rollback(r.Context(), req.BackupID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"success": true,
		"message": "Rollback completed successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleStatus checks optimization status
func (h *CDNHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	zoneID := r.URL.Query().Get("zone_id")
	if zoneID == "" {
		http.Error(w, "zone_id parameter required", http.StatusBadRequest)
		return
	}
	
	// Get zone details
	zone, err := h.fixer.GetZone(r.Context(), zoneID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Validate configuration - FIXED: Validate returns 2 values, takes map
	validator := fixer.NewValidator()
	
	// Create validation data
	validationData := map[string]interface{}{
		"zone_id":   zoneID,
		"zone_name": zone.Name,
	}
	
	valid, validationErrors := validator.Validate(validationData)
	
	status := map[string]interface{}{
		"zone_id":          zoneID,
		"zone_name":        zone.Name,
		"status":           "optimized",
		"valid":            valid,
		"validation_errors": validationErrors,
		"timestamp":        time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}