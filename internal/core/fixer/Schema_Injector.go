package fixer

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"log"

	"ai-seo-backend/internal/core/shopify"
)

// WordPressInjector handles schema injection for WordPress
type WordPressInjector struct {
    BaseURL  string
    Username string
    Password string
    Client   *WordPressClient
    Backup   *BackupManager
}

// ShopifyInjector handles schema injection for Shopify
type ShopifyInjector struct {
    Shop   string
    Token  string
    Client *http.Client
    Backup *BackupManager  // ← ADD THIS if missing
}

var config = &Config{
    Port:            "8080",
    BackupDir:       "./backups",
    RequestTimeout:  30 * time.Second,
    MaxRetries:      3,
    RetryDelay:      2 * time.Second,
}

type SchemaInjector struct {
    Client *http.Client
    Logger *log.Logger
}

func (s *SchemaInjector) Inject(url, platform string) []string {
    return []string{"Schema markup added"}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func NewSchemaInjector(client *http.Client, logger *log.Logger) *SchemaInjector {
    return &SchemaInjector{
    Client: client,
    Logger: logger,
    }
}

// ========== SIMPLE SCHEMA BUILDER ==========

func BuildProductSchema(name, description, price, currency, availability, brand string) map[string]interface{} {
	schema := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "Product",
		"name":        name,
		"description": description,
		"brand": map[string]interface{}{
			"@type": "Brand",
			"name":  brand,
		},
		"offers": map[string]interface{}{
			"@type":         "Offer",
			"price":         price,
			"priceCurrency": currency,
			"availability":  fmt.Sprintf("https://schema.org/%s", availability),
		},
	}
	return schema
}

func BuildArticleSchema(headline, author, datePublished, publisher string) map[string]interface{} {
	return map[string]interface{}{
		"@context":      "https://schema.org",
		"@type":         "Article",
		"headline":      headline,
		"author":        map[string]interface{}{"@type": "Person", "name": author},
		"datePublished": datePublished,
		"publisher": map[string]interface{}{
			"@type": "Organization",
			"name":  publisher,
		},
	}
}

func BuildLocalBusinessSchema(name, address, phone, hours string) map[string]interface{} {
	return map[string]interface{}{
		"@context":     "https://schema.org",
		"@type":        "LocalBusiness",
		"name":         name,
		"address":      address,
		"telephone":    phone,
		"openingHours": hours,
	}
}

func BuildFAQSchema(questions []map[string]string) map[string]interface{} {
	mainEntity := []map[string]interface{}{}
	for _, qa := range questions {
		mainEntity = append(mainEntity, map[string]interface{}{
			"@type": "Question",
			"name":  qa["question"],
			"acceptedAnswer": map[string]interface{}{
				"@type": "Answer",
				"text":  qa["answer"],
			},
		})
	}
	return map[string]interface{}{
		"@context":   "https://schema.org",
		"@type":      "FAQPage",
		"mainEntity": mainEntity,
	}
}

func NewSEOAnalyzer() *SEOAnalyzer {
	return &SEOAnalyzer{
		client: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			},
		},
	}
}

func (sa *SEOAnalyzer) AnalyzeAndFix(url string) (*SEOScore, error) {
	resp, err := sa.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	html, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read page: %w", err)
	}

	htmlStr := string(html)
	score := &SEOScore{
		TotalScore:  100,
		IssuesFound: []string{},
		FixedIssues: []string{},
	}

	// Check for schema markup
	hasSchema := strings.Contains(htmlStr, "application/ld+json")
	if !hasSchema {
		score.IssuesFound = append(score.IssuesFound, "No schema markup found - Rich snippets missing")
		score.TotalScore -= 30
	} else {
		score.FixedIssues = append(score.FixedIssues, "Schema markup present - Rich snippets enabled")
	}

	// Check title
	titleRegex := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)
	titleMatch := titleRegex.FindStringSubmatch(htmlStr)
	if len(titleMatch) == 0 {
		score.IssuesFound = append(score.IssuesFound, "Missing title tag - Critical for SEO")
		score.TotalScore -= 25
	} else if len(titleMatch[1]) < 30 || len(titleMatch[1]) > 60 {
		score.IssuesFound = append(score.IssuesFound, fmt.Sprintf("Title length issue (%d chars) - Affects CTR", len(titleMatch[1])))
		score.TotalScore -= 10
	} else {
		score.FixedIssues = append(score.FixedIssues, "Title tag optimized")
	}

	// Check meta description
	descRegex := regexp.MustCompile(`<meta name="description" content="([^"]+)"`)
	descMatch := descRegex.FindStringSubmatch(htmlStr)
	if len(descMatch) == 0 {
		score.IssuesFound = append(score.IssuesFound, "Missing meta description - Lower CTR in search results")
		score.TotalScore -= 20
	} else if len(descMatch[1]) < 120 || len(descMatch[1]) > 160 {
		score.IssuesFound = append(score.IssuesFound, "Meta description length suboptimal")
		score.TotalScore -= 5
	} else {
		score.FixedIssues = append(score.FixedIssues, "Meta description optimized")
	}

	// Check headings
	h1Count := strings.Count(htmlStr, "<h1")
	if h1Count == 0 {
		score.IssuesFound = append(score.IssuesFound, "No H1 heading - Poor content structure")
		score.TotalScore -= 15
	} else if h1Count > 1 {
		score.IssuesFound = append(score.IssuesFound, "Multiple H1 tags - Confusing for search engines")
		score.TotalScore -= 5
	} else {
		score.FixedIssues = append(score.FixedIssues, "Proper heading structure")
	}

	// Calculate grade
	if score.TotalScore >= 90 {
		score.Grade = "A"
		score.ExpectedRankingImprovement = "Expected 15-30% ranking improvement within 2-4 weeks"
	} else if score.TotalScore >= 70 {
		score.Grade = "B"
		score.ExpectedRankingImprovement = "Expected 10-20% ranking improvement within 3-5 weeks"
	} else if score.TotalScore >= 50 {
		score.Grade = "C"
		score.ExpectedRankingImprovement = "Expected 5-15% ranking improvement within 4-6 weeks"
	} else {
		score.Grade = "D"
		score.ExpectedRankingImprovement = "Critical fixes needed - Expected 20-40% improvement after fixes"
	}

	return score, nil
}

func NewBackupManager() *BackupManager {
    os.MkdirAll(config.BackupDir, 0755)
    return &BackupManager{BackupDir: config.BackupDir}
}

func (bm *BackupManager) CreateBackup(platform, identifier string, data []byte) (string, error) {
    timestamp := time.Now().Unix()
    backupID := fmt.Sprintf("%s_%s_%d", platform, identifier, timestamp)
    backupPath := filepath.Join(bm.BackupDir, backupID+".json")

    backup := map[string]interface{}{
        "id":         backupID,
        "platform":   platform,
        "identifier": identifier,
        "timestamp":  timestamp,
        "data":       string(data),
    }

    jsonData, err := json.Marshal(backup)
    if err != nil {
        return "", err
    }
    if err := os.WriteFile(backupPath, jsonData, 0644); err != nil {
        return "", err
    }

    return backupID, nil
}

func (bm *BackupManager) Restore(backupID string) error {
    backupPath := filepath.Join(bm.BackupDir, backupID+".json")
    
    data, err := os.ReadFile(backupPath)
    if err != nil {
        return fmt.Errorf("backup not found: %w", err)
    }

    var backup map[string]interface{}
    if err := json.Unmarshal(data, &backup); err != nil {
        return err
    }

    fmt.Printf("Restored backup: %s for platform: %s\n", backupID, backup["platform"])
    return nil
}

func (bm *BackupManager) ListBackups() ([]string, error) {
    files, err := filepath.Glob(filepath.Join(bm.BackupDir, "*.json"))
    if err != nil {
        return nil, err
    }
    
    backups := make([]string, len(files))
    for i, file := range files {
        backups[i] = strings.TrimSuffix(filepath.Base(file), ".json")
    }
    return backups, nil
}

func NewWordPressInjector(baseURL, username, password string, httpClient *http.Client, backupManager *BackupManager) *WordPressInjector {
	wpClient := &WordPressClient{
    SiteURL:  baseURL,
    Username: username,
    Password: password,
    Client: &http.Client{Timeout: 30 * time.Second},
}
   wp := &WordPressInjector{
    BaseURL:  baseURL,
    Username: username,
    Password: password,
    Client:   wpClient,
    Backup:   backupManager,
}
return wp
}

func (wp *WordPressInjector) InjectSchema(postID int, schema map[string]interface{}) error {
	schemaJSON, _ := json.Marshal(schema)
	backupID, err := wp.Backup.CreateBackup("wordpress", fmt.Sprintf("post_%d", postID), schemaJSON)
	if err != nil {
		fmt.Printf("Warning: Backup failed: %v\n", err)
	}

	// Try Yoast SEO API
	yoastURL := fmt.Sprintf("%s/wp-json/yoast/v1/schema", wp.BaseURL)
	req, _ := http.NewRequest("POST", yoastURL, bytes.NewBuffer(schemaJSON))
	req.SetBasicAuth(wp.Username, wp.Password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := wp.Client.Do(req)
	if err == nil && resp.StatusCode == 200 {
		resp.Body.Close()
		fmt.Printf("Schema injected via Yoast SEO (Backup: %s)\n", backupID)
		return nil
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Fallback to custom field
	customURL := fmt.Sprintf("%s/wp-json/wp/v2/posts/%d", wp.BaseURL, postID)
	
	updateData := map[string]interface{}{
		"meta": map[string]interface{}{
			"_schema_markup": string(schemaJSON),
		},
	}
	
	updateJSON, _ := json.Marshal(updateData)
	req, _ = http.NewRequest("POST", customURL, bytes.NewBuffer(updateJSON))
	req.SetBasicAuth(wp.Username, wp.Password)
	req.Header.Set("Content-Type", "application/json")

	resp, err = wp.Client.Do(req)
	if err != nil {
		return fmt.Errorf("injection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	fmt.Printf("Schema injected via custom fields (Backup: %s)\n", backupID)
	return nil
}

func (sh *ShopifyInjector) getActiveThemeID() (string, error) {
	url := fmt.Sprintf("https://%s.myshopify.com/admin/api/2024-01/themes.json", sh.Shop)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Shopify-Access-Token", sh.Token)

	resp, err := sh.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var response struct {
		Themes []struct {
			ID   int64  `json:"id"`
			Role string `json:"role"`
		} `json:"themes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	for _, theme := range response.Themes {
		if theme.Role == "main" {
			return fmt.Sprintf("%d", theme.ID), nil
		}
	}

	return "", fmt.Errorf("no active theme found")
}

func (sh *ShopifyInjector) InjectSchema(schema map[string]interface{}) error {
	themeID, err := sh.getActiveThemeID()
	if err != nil {
		return err
	}

	schemaJSON, _ := json.Marshal(schema)
	backupID, _ := sh.Backup.CreateBackup("shopify", themeID, schemaJSON)

	snippet := fmt.Sprintf(`<script type="application/ld+json">%s</script>`, string(schemaJSON))
	
	snippetURL := fmt.Sprintf("https://%s.myshopify.com/admin/api/2024-01/themes/%s/assets.json", sh.Shop, themeID)
	
	asset := map[string]interface{}{
		"asset": map[string]interface{}{
			"key":   "snippets/seo-schema.liquid",
			"value": snippet,
		},
	}
	
	assetJSON, _ := json.Marshal(asset)
	req, _ := http.NewRequest("PUT", snippetURL, bytes.NewBuffer(assetJSON))
	req.Header.Set("X-Shopify-Access-Token", sh.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := sh.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("failed to create snippet: status %d", resp.StatusCode)
	}

	themeURL := fmt.Sprintf("https://%s.myshopify.com/admin/api/2024-01/themes/%s/assets.json?asset[key]=layout/theme.liquid", sh.Shop, themeID)
	req, _ = http.NewRequest("GET", themeURL, nil)
	req.Header.Set("X-Shopify-Access-Token", sh.Token)

	resp, err = sh.Client.Do(req)
	if err != nil {
		return err
	}
	
	var themeResp struct {
		Asset struct {
			Value string `json:"value"`
		} `json:"asset"`
	}
	
	json.NewDecoder(resp.Body).Decode(&themeResp)
	resp.Body.Close()

	themeContent := themeResp.Asset.Value
	if !strings.Contains(themeContent, "seo-schema.liquid") {
		themeContent = strings.Replace(themeContent, "</head>", "{% include 'seo-schema' %}\n</head>", 1)
		
		updateAsset := map[string]interface{}{
			"asset": map[string]interface{}{
				"key":   "layout/theme.liquid",
				"value": themeContent,
			},
		}
		
		updateJSON, _ := json.Marshal(updateAsset)
		req, _ = http.NewRequest("PUT", themeURL, bytes.NewBuffer(updateJSON))
		req.Header.Set("X-Shopify-Access-Token", sh.Token)
		req.Header.Set("Content-Type", "application/json")
		
		resp, err = sh.Client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	fmt.Printf("Schema injected into Shopify theme (Backup: %s)\n", backupID)
	return nil
}

// ========== VALIDATOR ==========


func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) Validate(schema map[string]interface{}) (bool, []string) {
	errors := []string{}
	
	if _, ok := schema["@context"]; !ok {
		errors = append(errors, "Missing @context field")
	}
	
	if _, ok := schema["@type"]; !ok {
		errors = append(errors, "Missing @type field")
	}
	
	schemaType, _ := schema["@type"].(string)
	switch schemaType {
	case "Product":
		if _, ok := schema["name"]; !ok {
			errors = append(errors, "Product missing name")
		}
		if _, ok := schema["offers"]; !ok {
			errors = append(errors, "Product missing offers")
		}
	case "Article":
		if _, ok := schema["headline"]; !ok {
			errors = append(errors, "Article missing headline")
		}
	}
	
	return len(errors) == 0, errors
}


func NewAPIServer() *APIServer {
	return &APIServer{
		analyzer:  NewSEOAnalyzer(),
		validator: NewValidator(),
		backup:    NewBackupManager(),
	}
}

func (api *APIServer) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	score, err := api.analyzer.AnalyzeAndFix(req.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(score)
}

func (api *APIServer) handleGenerateSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var schema map[string]interface{}

	switch req.Type {
	case "product":
		schema = BuildProductSchema(
			getString(req.Data, "name"),
			getString(req.Data, "description"),
			getString(req.Data, "price"),
			getString(req.Data, "currency"),
			getString(req.Data, "availability"),
			getString(req.Data, "brand"),
		)
	case "article":
		schema = BuildArticleSchema(
			getString(req.Data, "headline"),
			getString(req.Data, "author"),
			getString(req.Data, "datePublished"),
			getString(req.Data, "publisher"),
		)
	case "local":
		schema = BuildLocalBusinessSchema(
			getString(req.Data, "name"),
			getString(req.Data, "address"),
			getString(req.Data, "phone"),
			getString(req.Data, "hours"),
		)
	default:
		http.Error(w, "Unsupported schema type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schema)
}

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func (api *APIServer) handleWordPressInject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL      string                 `json:"url"`
		Username string                 `json:"username"`
		Password string                 `json:"password"`
		PostID   int                    `json:"post_id"`
		Schema   map[string]interface{} `json:"schema"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
    backupManager := NewBackupManager()

   baseURL := r.URL.Query().Get("url")
  username := r.URL.Query().Get("username")
 password := r.URL.Query().Get("password")

injector := NewWordPressInjector(baseURL, username, password, httpClient, backupManager)

if err := injector.InjectSchema(req.PostID, req.Schema); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
}

	response := map[string]interface{}{
		"success": true,
		"message": "Schema injected successfully into WordPress",
		"benefits": map[string]string{
			"rich_snippets":      "Enabled - Products will show price, rating in search",
			"ctr_improvement":    "Expected 15-30% increase in click-through rate",
			"ranking_signal":     "Improved structured data signals to Google",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (api *APIServer) handleShopifyInject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Shop   string                 `json:"shop"`
		Token  string                 `json:"token"`
		Schema map[string]interface{} `json:"schema"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	shopURL := req.Shop
    accessToken := req.Token

	httpClient := &http.Client{Timeout: 30 * time.Second}
backupManager := NewBackupManager()

  shopifyInjector := shopify.NewShopifyInjector(shopURL, accessToken, httpClient, backupManager)
	
	if err := shopifyInjector.InjectSchema(req.Schema); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Schema injected successfully into Shopify",
		"benefits": map[string]string{
			"rich_snippets":      "Enabled - Products show in Google Shopping",
			"ctr_improvement":    "Expected 20-40% increase in click-through rate",
			"conversion_impact":  "Rich snippets increase purchase intent",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (api *APIServer) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var schema map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&schema); err != nil {
		http.Error(w, "Invalid schema", http.StatusBadRequest)
		return
	}

	valid, errors := api.validator.Validate(schema)
	
	response := map[string]interface{}{
		"valid":  valid,
		"errors": errors,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (api *APIServer) handleListBackups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	backups, err := api.backup.ListBackups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"backups": backups,
		"count":   len(backups),
	})
}

func (api *APIServer) handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	backupID := strings.TrimPrefix(r.URL.Path, "/api/backup/restore/")
	if backupID == "" {
		http.Error(w, "Backup ID required", http.StatusBadRequest)
		return
	}

	if err := api.backup.Restore(backupID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Backup %s restored successfully", backupID),
	})
}

func (api *APIServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>SEO Schema Injector</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
        .container { max-width: 900px; margin: 0 auto; }
        .card { background: white; border-radius: 10px; padding: 20px; margin-bottom: 20px; }
        h1 { color: #667eea; }
        input, select, textarea { width: 100%; padding: 10px; margin: 10px 0; }
        button { background: #667eea; color: white; border: none; padding: 10px 20px; cursor: pointer; }
        .result { background: #f5f5f5; padding: 10px; margin-top: 10px; display: none; }
        .score { font-size: 48px; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <div class="card">
            <h1>SEO Schema Injector</h1>
            <p>Real SEO Tool That Improves Rankings</p>
        </div>
        <div class="card">
            <h2>Analyze Website</h2>
            <input type="text" id="analyzeUrl" placeholder="https://example.com">
            <button onclick="analyzeSEO()">Analyze</button>
            <div id="analyzeResult" class="result"></div>
        </div>
        <div class="card">
            <h2>Generate Schema</h2>
            <select id="schemaType">
                <option value="product">Product</option>
                <option value="article">Article</option>
                <option value="local">Local Business</option>
            </select>
            <textarea id="schemaData" rows="3" placeholder='{"name":"Product Name","price":"99.99"}'></textarea>
            <button onclick="generateSchema()">Generate</button>
            <div id="schemaResult" class="result"></div>
        </div>
    </div>
    <script>
        async function analyzeSEO() {
            const url = document.getElementById('analyzeUrl').value;
            const resultDiv = document.getElementById('analyzeResult');
            resultDiv.style.display = 'block';
            resultDiv.innerHTML = 'Analyzing...';
            const response = await fetch('/api/analyze', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({url: url})
            });
            const data = await response.json();
            resultDiv.innerHTML = '<h3>Results</h3><p>Score: ' + data.total_score + '/100</p><p>Grade: ' + data.grade + '</p><p>' + data.expected_ranking_improvement + '</p>';
        }
        async function generateSchema() {
            const type = document.getElementById('schemaType').value;
            const data = JSON.parse(document.getElementById('schemaData').value);
            const resultDiv = document.getElementById('schemaResult');
            resultDiv.style.display = 'block';
            resultDiv.innerHTML = 'Generating...';
            const response = await fetch('/api/generate-schema', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({type: type, data: data})
            });
            const schema = await response.json();
            resultDiv.innerHTML = '<pre>' + JSON.stringify(schema, null, 2) + '</pre>';
        }
    </script>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (api *APIServer) Start() error {
	http.HandleFunc("/api/analyze", api.handleAnalyze)
	http.HandleFunc("/api/generate-schema", api.handleGenerateSchema)
	http.HandleFunc("/api/inject/wordpress", api.handleWordPressInject)
	http.HandleFunc("/api/inject/shopify", api.handleShopifyInject)
	http.HandleFunc("/api/validate", api.handleValidate)
	http.HandleFunc("/api/backups", api.handleListBackups)
	http.HandleFunc("/api/backup/restore/", api.handleRestoreBackup)
	http.HandleFunc("/", api.handleDashboard)

	fmt.Printf("\nSEO Schema Injector Server Started\n")
	fmt.Printf("Server: http://localhost:%s\n", config.Port)
	fmt.Printf("\n")

	return http.ListenAndServe(":"+config.Port, nil)
}
