package optimizer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Core structs
type KeywordData struct {
	Keyword    string   `json:"keyword"`
	Volume     int      `json:"volume"`
	Difficulty float64  `json:"difficulty"`
	Intent     string   `json:"intent"`
	LSI        []string `json:"lsi,omitempty"`
}

type GapAnalysis struct {
	MissingKeywords []string `json:"missing_keywords"`
	Opportunities   []string `json:"opportunities"`
}

type Cannibalization struct {
	Keyword string   `json:"keyword"`
	Pages   []string `json:"pages"`
}

type KeywordOptimizer struct {
	client *http.Client
}

// NewKeywordOptimizer creates a new keyword optimizer
func NewKeywordOptimizer() *KeywordOptimizer {
	return &KeywordOptimizer{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// ExtractKeywords gets real keywords from HTML
func (ko *KeywordOptimizer) ExtractKeywords(html string) map[string]int {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return map[string]int{}
	}

	// Get text content from important HTML elements
	var textBuilder strings.Builder
	
	// Title (highest weight)
	doc.Find("title").Each(func(i int, s *goquery.Selection) {
		textBuilder.WriteString(" " + s.Text())
	})
	
	// Meta description
	doc.Find("meta[name='description']").Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists {
			textBuilder.WriteString(" " + content)
		}
	})
	
	// Headings
	doc.Find("h1, h2, h3").Each(func(i int, s *goquery.Selection) {
		textBuilder.WriteString(" " + s.Text())
	})
	
	// Paragraphs and list items
	doc.Find("p, li").Each(func(i int, s *goquery.Selection) {
		textBuilder.WriteString(" " + s.Text())
	})
	
	text := textBuilder.String()
	
	// Count words
	words := strings.Fields(strings.ToLower(text))
	counts := make(map[string]int)
	
	// Remove common stop words
	stopwords := map[string]bool{
		"the": true, "and": true, "for": true, "you": true, "are": true,
		"this": true, "that": true, "have": true, "from": true, "was": true,
		"were": true, "will": true, "been": true, "has": true, "had": true,
		"can": true, "all": true, "any": true, "each": true, "some": true,
		"such": true, "than": true, "then": true, "them": true, "these": true,
		"they": true, "into": true, "over": true, "very": true,
	}
	
	// Regular expression to clean words
	re := regexp.MustCompile(`[^a-z]`)
	
	for _, word := range words {
		// Clean the word
		cleanWord := re.ReplaceAllString(word, "")
		
		// Skip short words and stop words
		if len(cleanWord) > 2 && !stopwords[cleanWord] {
			counts[cleanWord]++
		}
	}
	
	return counts
}

// GetSearchVolume gets estimated search volume based on keyword patterns
func (ko *KeywordOptimizer) GetSearchVolume(keyword string) int {
	// Realistic volume based on keyword patterns
	words := strings.Fields(keyword)
	wordCount := len(words)
	
	// Base volume
	volume := 500
	
	// Adjust based on keyword characteristics
	switch {
	case strings.Contains(keyword, "best"):
		volume = 1000 + (wordCount * 100)
	case strings.Contains(keyword, "how to"):
		volume = 2000 + (wordCount * 50)
	case strings.Contains(keyword, "review"):
		volume = 800 + (wordCount * 80)
	case strings.Contains(keyword, "vs"):
		volume = 600 + (wordCount * 70)
	case wordCount > 3:
		volume = 100 + (wordCount * 20) // Long tail
	case wordCount == 1:
		volume = 1000 + (wordCount * 500) // Head terms
	}
	
	// Ensure volume is positive
	if volume < 0 {
		volume = 100
	}
	
	return volume
}

// FindGaps finds keywords competitors rank for but you don't
func (ko *KeywordOptimizer) FindGaps(yourURL string, compURLs []string) (*GapAnalysis, error) {
	// Validate input
	if yourURL == "" {
		return nil, fmt.Errorf("your URL cannot be empty")
	}
	
	// Get your keywords
	yourContent, err := ko.fetchURL(yourURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch your URL: %w", err)
	}
	yourKeywords := ko.ExtractKeywords(yourContent)

	// Get competitor keywords
	compKeywords := make(map[string]int) // Track frequency across competitors
	
	for _, url := range compURLs {
		if url == "" {
			continue
		}
		
		content, err := ko.fetchURL(url)
		if err != nil {
			// Log error but continue with other competitors
			continue
		}
		
		for kw := range ko.ExtractKeywords(content) {
			compKeywords[kw]++
		}
	}

	// Find gaps and opportunities
	var gaps []string
	var opportunities []string
	opportunityCount := 0
	
	for kw, frequency := range compKeywords {
		// If you don't have this keyword and multiple competitors use it
		if yourKeywords[kw] == 0 {
			gaps = append(gaps, kw)
			
			// Top opportunities (keywords used by multiple competitors)
			if opportunityCount < 10 && frequency >= 2 {
				volume := ko.GetSearchVolume(kw)
				opportunities = append(opportunities, fmt.Sprintf("Target: %s (volume: %d, used by %d competitors)", 
					kw, volume, frequency))
				opportunityCount++
			}
		}
	}

	return &GapAnalysis{
		MissingKeywords: gaps,
		Opportunities:   opportunities,
	}, nil
}

// GetLSI finds related keywords using Google Suggest API
func (ko *KeywordOptimizer) GetLSI(keyword string) []string {
	if keyword == "" {
		return []string{}
	}
	
	// Use Google Suggest API
	apiURL := fmt.Sprintf("http://suggestqueries.google.com/complete/search?client=firefox&q=%s", 
		url.QueryEscape(keyword))
	
	// Create request with timeout
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return ko.fallbackLSI(keyword)
	}
	
	// Set headers to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	
	resp, err := ko.client.Do(req)
	if err != nil {
		return ko.fallbackLSI(keyword)
	}
	defer resp.Body.Close()

	// Parse response
	var suggestions []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&suggestions); err != nil {
		return ko.fallbackLSI(keyword)
	}

	if len(suggestions) > 1 {
		if terms, ok := suggestions[1].([]interface{}); ok {
			result := make([]string, 0, len(terms))
			for _, t := range terms {
				if term, ok := t.(string); ok {
					// Remove the original keyword from suggestions if present
					if !strings.EqualFold(term, keyword) {
						result = append(result, term)
					}
				}
			}
			
			// Limit to top 10 suggestions
			if len(result) > 10 {
				result = result[:10]
			}
			
			return result
		}
	}
	
	return ko.fallbackLSI(keyword)
}

// DetectCannibalization finds pages targeting same keywords
func (ko *KeywordOptimizer) DetectCannibalization(siteURL string, maxPages int) ([]Cannibalization, error) {
	if siteURL == "" {
		return nil, fmt.Errorf("site URL cannot be empty")
	}
	
	// Get site pages (in production, you'd crawl the sitemap)
	pages := []string{
		siteURL,
		siteURL + "/blog",
		siteURL + "/products",
		siteURL + "/services",
		siteURL + "/about",
		siteURL + "/contact",
	}
	
	// Limit pages if specified
	if maxPages > 0 && len(pages) > maxPages {
		pages = pages[:maxPages]
	}
	
	keywordPages := make(map[string]map[string]int) // keyword -> page -> count
	
	for _, page := range pages {
		content, err := ko.fetchURL(page)
		if err != nil {
			continue // Skip pages that can't be fetched
		}
		
		keywords := ko.ExtractKeywords(content)
		for kw, count := range keywords {
			if count > 1 { // Keyword appears at least twice
				if keywordPages[kw] == nil {
					keywordPages[kw] = make(map[string]int)
				}
				keywordPages[kw][page] = count
			}
		}
	}

	// Find cannibalization issues
	var issues []Cannibalization
	for kw, pageCounts := range keywordPages {
		if len(pageCounts) > 1 {
			// Convert map keys to slice
			pages := make([]string, 0, len(pageCounts))
			for page := range pageCounts {
				pages = append(pages, page)
			}
			
			issues = append(issues, Cannibalization{
				Keyword: kw,
				Pages:   pages,
			})
		}
	}
	
	return issues, nil
}

// OptimizeContent gives density suggestions
func (ko *KeywordOptimizer) OptimizeContent(content, target string) []string {
	if content == "" || target == "" {
		return []string{"Please provide both content and target keyword"}
	}
	
	words := strings.Fields(strings.ToLower(content))
	if len(words) == 0 {
		return []string{"No content to analyze"}
	}
	
	// Count target keyword occurrences
	targetWords := strings.Fields(strings.ToLower(target))
	targetCount := 0
	
	for i := 0; i <= len(words)-len(targetWords); i++ {
		match := true
		for j, word := range targetWords {
			if words[i+j] != word {
				match = false
				break
			}
		}
		if match {
			targetCount++
		}
	}
	
	// Calculate density
	density := float64(targetCount) / float64(len(words)) * 100
	
	var suggestions []string
	
	// Density suggestions
	if density < 1.0 {
		suggestions = append(suggestions, 
			fmt.Sprintf("⚠️ Increase '%s' usage (currently %.1f%%, target 1-2%%)", target, density))
		suggestions = append(suggestions, "• Add to H1 tag")
		suggestions = append(suggestions, "• Include in first paragraph")
		suggestions = append(suggestions, "• Use in headings and subheadings")
		suggestions = append(suggestions, "• Add to image alt text")
	} else if density > 3.0 {
		suggestions = append(suggestions,
			fmt.Sprintf("⚠️ Reduce '%s' usage to avoid over-optimization (currently %.1f%%)", target, density))
		suggestions = append(suggestions, "• Use synonyms instead")
		suggestions = append(suggestions, "• Replace some occurrences with related terms")
	} else {
		suggestions = append(suggestions,
			fmt.Sprintf("✅ Good keyword density: %.1f%% (target 1-2%%)", density))
	}
	
	// Add LSI suggestions
	lsi := ko.GetLSI(target)
	if len(lsi) > 0 {
		relatedTerms := lsi
		if len(lsi) > 5 {
			relatedTerms = lsi[:5]
		}
		suggestions = append(suggestions, 
			fmt.Sprintf("📝 Consider using related terms: %s", strings.Join(relatedTerms, ", ")))
	}
	
	return suggestions
}

// AnalyzeKeyword performs comprehensive keyword analysis
func (ko *KeywordOptimizer) AnalyzeKeyword(keyword string) (*KeywordData, error) {
	if keyword == "" {
		return nil, fmt.Errorf("keyword cannot be empty")
	}
	
	// Get search volume
	volume := ko.GetSearchVolume(keyword)
	
	// Calculate difficulty (simplified algorithm)
	difficulty := ko.calculateDifficulty(keyword)
	
	// Detect intent
	intent := ko.detectIntent(keyword)
	
	// Get LSI keywords
	lsi := ko.GetLSI(keyword)
	
	return &KeywordData{
		Keyword:    keyword,
		Volume:     volume,
		Difficulty: difficulty,
		Intent:     intent,
		LSI:        lsi,
	}, nil
}

// BatchAnalyzeKeywords analyzes multiple keywords
func (ko *KeywordOptimizer) BatchAnalyzeKeywords(keywords []string) []*KeywordData {
	results := make([]*KeywordData, 0, len(keywords))
	
	for _, keyword := range keywords {
		if data, err := ko.AnalyzeKeyword(keyword); err == nil {
			results = append(results, data)
		}
	}
	
	return results
}

// Helper: fetch URL with retry and error handling
func (ko *KeywordOptimizer) fetchURL(urlStr string) (string, error) {
	// Validate URL
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}
	
	// Create request
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers to avoid blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; KeywordOptimizer/1.0; +http://example.com/bot)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	
	// Execute with retry
	var resp *http.Response
	var lastErr error
	
	for retries := 0; retries < 3; retries++ {
		resp, err = ko.client.Do(req)
		if err == nil {
			break
		}
		lastErr = err
		time.Sleep(time.Duration(retries+1) * time.Second)
	}
	
	if resp == nil {
		return "", fmt.Errorf("failed to fetch URL after retries: %w", lastErr)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}
	
	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	// Get full HTML
	html, err := doc.Html()
	if err != nil {
		return "", fmt.Errorf("failed to generate HTML: %w", err)
	}
	
	return html, nil
}

// Helper: fallback LSI when API fails
func (ko *KeywordOptimizer) fallbackLSI(keyword string) []string {
	// Simple but realistic LSI suggestions based on keyword patterns
	words := strings.Fields(keyword)
	wordCount := len(words)
	
	switch {
	case wordCount == 1:
		// Single word keyword
		return []string{
			"best " + keyword,
			keyword + " review",
			keyword + " pricing",
			keyword + " vs",
			"how to use " + keyword,
			keyword + " tutorial",
			keyword + " examples",
			"what is " + keyword,
		}
	case wordCount == 2:
		// Two-word keyword
		return []string{
			keyword + " guide",
			keyword + " tools",
			"best " + keyword,
			keyword + " review",
			keyword + " pricing",
			"how to " + keyword,
		}
	default:
		// Long-tail keyword
		return []string{
			keyword + " guide",
			keyword + " tutorial",
			keyword + " examples",
			"best " + keyword,
			keyword + " tips",
		}
	}
}

// Helper: calculate keyword difficulty
func (ko *KeywordOptimizer) calculateDifficulty(keyword string) float64 {
	words := strings.Fields(keyword)
	wordCount := len(words)
	
	// Base difficulty
	difficulty := 50.0
	
	// Adjust based on keyword characteristics
	switch {
	case strings.Contains(keyword, "best"):
		difficulty += 15 // Commercial keywords are harder
	case strings.Contains(keyword, "review"):
		difficulty += 10
	case strings.Contains(keyword, "how to"):
		difficulty -= 10 // Informational keywords are easier
	case strings.Contains(keyword, "what is"):
		difficulty -= 15
	}
	
	// Long-tail keywords are easier
	if wordCount > 3 {
		difficulty -= 15
	}
	
	// Head terms are harder
	if wordCount == 1 {
		difficulty += 20
	}
	
	// Ensure within 0-100 range
	if difficulty < 0 {
		difficulty = 0
	} else if difficulty > 100 {
		difficulty = 100
	}
	
	return difficulty
}

// Helper: detect search intent
func (ko *KeywordOptimizer) detectIntent(keyword string) string {
	keyword = strings.ToLower(keyword)
	
	// Transactional intent
	transactional := []string{"buy", "purchase", "order", "cheap", "discount", "price", "cost", "deal"}
	for _, term := range transactional {
		if strings.Contains(keyword, term) {
			return "transactional"
		}
	}
	
	// Commercial intent
	commercial := []string{"best", "top", "review", "vs", "comparison", "alternative"}
	for _, term := range commercial {
		if strings.Contains(keyword, term) {
			return "commercial"
		}
	}
	
	// Navigational intent
	navigational := []string{"login", "sign in", "register", "dashboard", "account"}
	for _, term := range navigational {
		if strings.Contains(keyword, term) {
			return "navigational"
		}
	}
	
	// Default to informational
	return "informational"
}

// API Handler
type API struct {
	opt *KeywordOptimizer
}

func NewAPI() *API {
	return &API{opt: NewKeywordOptimizer()}
}

// AnalyzeRequest represents the analysis request body
type AnalyzeRequest struct {
	URL         string   `json:"url"`
	Competitors []string `json:"competitors"`
	Keyword     string   `json:"keyword"`
}

// AnalyzeResponse represents the analysis response
type AnalyzeResponse struct {
	URL             string                 `json:"url"`
	MainKeyword     string                 `json:"main_keyword"`
	FoundKeywords   map[string]int         `json:"found_keywords"`
	LSISuggestions  []string               `json:"lsi_suggestions"`
	OptimizationTips []string               `json:"optimization_tips"`
	CompetitorGaps  *GapAnalysis           `json:"competitor_gaps"`
	Error           string                  `json:"error,omitempty"`
}

func (api *API) AnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(AnalyzeResponse{Error: "POST method required"})
		return
	}

	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(AnalyzeResponse{Error: "Invalid JSON request"})
		return
	}

	// Validate request
	if req.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(AnalyzeResponse{Error: "URL is required"})
		return
	}

	// Get content
	content, err := api.opt.fetchURL(req.URL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(AnalyzeResponse{Error: fmt.Sprintf("Failed to fetch URL: %v", err)})
		return
	}

	// Analyze
	keywords := api.opt.ExtractKeywords(content)
	
	var lsi []string
	if req.Keyword != "" {
		lsi = api.opt.GetLSI(req.Keyword)
	}
	
	var optimize []string
	if req.Keyword != "" {
		optimize = api.opt.OptimizeContent(content, req.Keyword)
	}
	
	var gaps *GapAnalysis
	if len(req.Competitors) > 0 {
		gaps, _ = api.opt.FindGaps(req.URL, req.Competitors)
	}

	response := AnalyzeResponse{
		URL:             req.URL,
		MainKeyword:     req.Keyword,
		FoundKeywords:   keywords,
		LSISuggestions:  lsi,
		OptimizationTips: optimize,
		CompetitorGaps:  gaps,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HealthHandler returns service health status
func (api *API) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	response := map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
		"version": "1.0.0",
	}
	
	json.NewEncoder(w).Encode(response)
}

// Simple test example (not a main function)
func Example() {
	opt := NewKeywordOptimizer()
	
	// Get keywords from a page
	content := "<html><body><h1>SEO Tools</h1><p>Best SEO tools for keyword research and optimization</p></body></html>"
	kws := opt.ExtractKeywords(content)
	fmt.Printf("Found %d keywords\n", len(kws))
	
	// Get LSI suggestions
	lsi := opt.GetLSI("seo tools")
	fmt.Printf("LSI suggestions: %v\n", lsi)
	
	// Get optimization tips
	tips := opt.OptimizeContent(content, "seo tools")
	fmt.Printf("Optimization tips: %v\n", tips)
	
	// Analyze keyword
	data, _ := opt.AnalyzeKeyword("seo tools")
	fmt.Printf("Keyword data: %+v\n", data)
}