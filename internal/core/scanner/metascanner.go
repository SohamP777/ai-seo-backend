// Package metascanner provides comprehensive SEO meta tag analysis and optimization
package scanner

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// MetaTag represents a single meta tag with its properties
type MetaTag struct {
	Name           string   `json:"name"`
	Property       string   `json:"property,omitempty"`
	Content        string   `json:"content"`
	HTTPEquiv      string   `json:"http_equiv,omitempty"`
	Charset        string   `json:"charset,omitempty"`
	IsValid        bool     `json:"is_valid"`
	Issues         []string `json:"issues,omitempty"`
	Recommendation string   `json:"recommendation,omitempty"`
}

// ImageMeta represents image metadata for SEO optimization
type ImageMeta struct {
	URL           string   `json:"url"`
	AltText       string   `json:"alt_text"`
	Title         string   `json:"title"`
	Width         int      `json:"width"`
	Height        int      `json:"height"`
	FileSize      int64    `json:"file_size"`
	OptimizedSize int64    `json:"optimized_size"`
	Format        string   `json:"format"`
	WebPGenerated bool     `json:"webp_generated"`
	MissingAlt    bool     `json:"missing_alt"`
	LazyLoaded    bool     `json:"lazy_loaded"`
	Issues        []string `json:"issues"`
}

// LinkMeta represents link metadata for SEO
type LinkMeta struct {
	HREF       string   `json:"href"`
	Rel        string   `json:"rel"`
	Type       string   `json:"type"`
	Title      string   `json:"title"`
	IsBroken   bool     `json:"is_broken"`
	StatusCode int      `json:"status_code,omitempty"`
	IsInternal bool     `json:"is_internal"`
	Follow     bool     `json:"follow"`
	Issues     []string `json:"issues"`
}

// SchemaMeta represents schema.org markup
type SchemaMeta struct {
	Type       string                 `json:"type"`
	Context    string                 `json:"context"`
	Data       map[string]interface{} `json:"data"`
	Raw        string                 `json:"raw"`
	IsValid    bool                   `json:"is_valid"`
	Validation []string               `json:"validation,omitempty"`
}

// PageSpeedMetrics represents page speed optimization data
type PageSpeedMetrics struct {
	LoadTime          time.Duration `json:"load_time"`
	HTMLSize          int64         `json:"html_size"`
	HTMLMinified      int64         `json:"html_minified"`
	CSSSize           int64         `json:"css_size"`
	CSSMinified       int64         `json:"css_minified"`
	JSSize            int64         `json:"js_size"`
	JSMinified        int64         `json:"js_minified"`
	TotalRequests     int           `json:"total_requests"`
	CompressionEnabled bool         `json:"compression_enabled"`
	CacheHeaders      bool          `json:"cache_headers"`
	Score             int           `json:"score"`
}

// ContentMetrics represents content optimization metrics
type ContentMetrics struct {
	WordCount        int             `json:"word_count"`
	ReadabilityScore float64         `json:"readability_score"`
	KeywordDensity   map[string]float64 `json:"keyword_density"`
	PrimaryKeyword   string          `json:"primary_keyword"`
	HeadingStructure map[string]int  `json:"heading_structure"`
	HasH1            bool            `json:"has_h1"`
	MultipleH1       bool            `json:"multiple_h1"`
	Issues           []string        `json:"issues"`
}

// MobileMetrics represents mobile responsiveness metrics
type MobileMetrics struct {
	ViewportConfigured bool     `json:"viewport_configured"`
	TapTargets         []string `json:"tap_targets"`
	FontSizes          []string `json:"font_sizes"`
	MediaQueries       int      `json:"media_queries"`
	Score              int      `json:"score"`
	Issues             []string `json:"issues"`
}

// ScanResult represents the complete meta scan result
type ScanResult struct {
	URL             string            `json:"url"`
	Timestamp       time.Time         `json:"timestamp"`
	Title           string            `json:"title"`
	HTML            string            `json:"html,omitempty"`
	MetaTags        []MetaTag         `json:"meta_tags"`
	OpenGraph       map[string]string `json:"open_graph"`
	TwitterCard     map[string]string `json:"twitter_card"`
	Images          []ImageMeta       `json:"images"`
	Links           []LinkMeta        `json:"links"`
	CanonicalURL    string            `json:"canonical_url"`
	RobotsMeta      string            `json:"robots_meta"`
	SchemaMarkup    []SchemaMeta      `json:"schema_markup"`
	PageSpeed       PageSpeedMetrics  `json:"page_speed"`
	Content         ContentMetrics    `json:"content"`
	Mobile          MobileMetrics     `json:"mobile"`
	StatusCode      int               `json:"status_code"`
	Issues          []string          `json:"issues"`
	Recommendations []string          `json:"recommendations"`
	Score           int               `json:"score"`
}

// ScannerConfig holds configuration for the meta scanner
type ScannerConfig struct {
	Timeout          time.Duration `json:"timeout"`
	UserAgent        string        `json:"user_agent"`
	FollowRedirects  bool          `json:"follow_redirects"`
	MaxRedirects     int           `json:"max_redirects"`
	CheckBrokenLinks bool          `json:"check_broken_links"`
	OptimizeImages   bool          `json:"optimize_images"`
	MinifyContent    bool          `json:"minify_content"`
	CheckMobile      bool          `json:"check_mobile"`
	OutputDir        string        `json:"output_dir"`
	EnableJavaScript bool          `json:"enable_javascript"`
	Concurrency      int           `json:"concurrency"`
}

// MetaScanner main scanner struct
type MetaScanner struct {
	config   ScannerConfig
	client   *http.Client
	minifier *minify.M
	baseURL  *url.URL
	results  *ScanResult
	mu       sync.RWMutex
}

// NewMetaScanner creates a new meta scanner instance
func NewMetaScanner(config ScannerConfig) *MetaScanner {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.UserAgent == "" {
		config.UserAgent = "MetaScanner/1.0 (+https://github.com/metascanner)"
	}
	if config.OutputDir == "" {
		config.OutputDir = "./scan_results"
	}
	if config.Concurrency == 0 {
		config.Concurrency = 5
	}

	client := &http.Client{
		Timeout: config.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !config.FollowRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) >= config.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", config.MaxRedirects)
			}
			return nil
		},
	}

	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("text/javascript", js.Minify)

	return &MetaScanner{
		config:   config,
		client:   client,
		minifier: m,
		results:  &ScanResult{},
	}
}

// Scan performs a comprehensive meta scan of the given URL
func (ms *MetaScanner) Scan(targetURL string) (*ScanResult, error) {
	ms.results = &ScanResult{
		URL:            targetURL,
		Timestamp:      time.Now(),
		MetaTags:       []MetaTag{},
		OpenGraph:      make(map[string]string),
		TwitterCard:    make(map[string]string),
		Images:         []ImageMeta{},
		Links:          []LinkMeta{},
		SchemaMarkup:   []SchemaMeta{},
		Issues:         []string{},
		Recommendations: []string{},
	}

	// Parse base URL
	base, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	ms.baseURL = base

	// Create output directory
	if err := os.MkdirAll(ms.config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	// Fetch and parse the page
	if ms.config.EnableJavaScript {
		if err := ms.scanWithJavaScript(targetURL); err != nil {
			return nil, err
		}
	} else {
		if err := ms.scanWithHTTP(targetURL); err != nil {
			return nil, err
		}
	}

	// Calculate overall score
	ms.calculateScore()

	// Generate recommendations
	ms.generateRecommendations()

	return ms.results, nil
}

// scanWithJavaScript scans the page using Chrome headless browser
func (ms *MetaScanner) scanWithJavaScript(targetURL string) error {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, ms.config.Timeout)
	defer cancel()

	var statusCode int64
	var htmlContent string

	err := chromedp.Run(ctx,
		chromedp.Navigate(targetURL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			chromedp.ListenTarget(ctx, func(ev interface{}) {
				switch ev := ev.(type) {
				case *network.EventResponseReceived:
					if ev.Type == network.ResourceTypeDocument {
						statusCode = ev.Response.Status
					}
				}
			})
			return nil
		}),
		chromedp.Sleep(2*time.Second), // Wait for dynamic content
		chromedp.OuterHTML("html", &htmlContent, chromedp.ByQuery),
	)

	if err != nil {
		return fmt.Errorf("chrome scan failed: %w", err)
	}

	ms.results.StatusCode = int(statusCode)
	ms.results.HTML = htmlContent

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return err
	}

	// Perform analyses
	ms.analyzeTitle(doc)
	ms.analyzeMetaTags(doc)
	ms.analyzeOpenGraph(doc)
	ms.analyzeTwitterCard(doc)
	ms.analyzeCanonical(doc)
	ms.analyzeRobots(doc)
	ms.analyzeSchemaMarkup(doc)
	ms.analyzeImages(doc)
	ms.analyzeLinks(doc)
	ms.analyzeContent(doc)
	ms.analyzePageSpeed(doc, []byte(htmlContent))
	ms.analyzeMobile(doc)

	return nil
}

// scanWithHTTP scans the page using HTTP client
func (ms *MetaScanner) scanWithHTTP(targetURL string) error {
	// Fetch the page
	resp, err := ms.fetchPage(targetURL)
	if err != nil {
		return fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	ms.results.StatusCode = resp.StatusCode

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// Decode body
	body, err := ms.decodeBody(resp)
	if err != nil {
		return fmt.Errorf("failed to decode body: %w", err)
	}

	ms.results.HTML = string(body)

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Perform analyses
	ms.analyzeTitle(doc)
	ms.analyzeMetaTags(doc)
	ms.analyzeOpenGraph(doc)
	ms.analyzeTwitterCard(doc)
	ms.analyzeCanonical(doc)
	ms.analyzeRobots(doc)
	ms.analyzeSchemaMarkup(doc)
	ms.analyzeImages(doc)
	ms.analyzeLinks(doc)
	ms.analyzeContent(doc)
	ms.analyzePageSpeed(doc, body)
	ms.analyzeMobile(doc)

	return nil
}

// fetchPage fetches a page with proper headers
func (ms *MetaScanner) fetchPage(pageURL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", ms.config.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	resp, err := ms.client.Do(req)
	if err != nil {
		return nil, err
	}

	// Handle compression
	if resp.Header.Get("Content-Encoding") == "gzip" {
		ms.results.PageSpeed.CompressionEnabled = true
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body = reader
	}

	// Check cache headers
	if resp.Header.Get("Cache-Control") != "" || resp.Header.Get("Expires") != "" {
		ms.results.PageSpeed.CacheHeaders = true
	}

	return resp, nil
}

// decodeBody handles character set decoding
func (ms *MetaScanner) decodeBody(resp *http.Response) ([]byte, error) {
	contentType := resp.Header.Get("Content-Type")

	// Detect encoding
	e, name, _ := charset.DetermineEncoding(nil, contentType)
	if name == "utf-8" || name == "" {
		e = unicode.UTF8
	}

	// Decode body
	reader := transform.NewReader(resp.Body, e.NewDecoder())
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// analyzeTitle analyzes the page title
func (ms *MetaScanner) analyzeTitle(doc *goquery.Document) {
	title := strings.TrimSpace(doc.Find("title").First().Text())
	ms.results.Title = title

	// Validate title
	if title == "" {
		ms.addIssue("high", "Missing page title", "Add a descriptive title tag (50-60 characters)")
	} else {
		length := len(title)
		if length < 30 {
			ms.addIssue("medium", fmt.Sprintf("Title too short: %d characters", length),
				"Aim for 50-60 characters for optimal display in search results")
		} else if length > 60 {
			ms.addIssue("medium", fmt.Sprintf("Title too long: %d characters", length),
				"Titles over 60 characters may be truncated in search results")
		}
	}
}

// analyzeMetaTags analyzes all meta tags
func (ms *MetaScanner) analyzeMetaTags(doc *goquery.Document) {
	// Track found meta tags
	foundDesc := false
	foundViewport := false

	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		meta := MetaTag{
			IsValid: true,
			Issues:  []string{},
		}

		// Get attributes
		if name, exists := s.Attr("name"); exists {
			meta.Name = name
		}
		if property, exists := s.Attr("property"); exists {
			meta.Property = property
		}
		if content, exists := s.Attr("content"); exists {
			meta.Content = content
		}
		if httpEquiv, exists := s.Attr("http-equiv"); exists {
			meta.HTTPEquiv = httpEquiv
		}
		if charset, exists := s.Attr("charset"); exists {
			meta.Charset = charset
		}

		// Validate based on type
		switch {
		case meta.Name == "description":
			foundDesc = true
			if len(meta.Content) < 50 {
				meta.IsValid = false
				meta.Issues = append(meta.Issues, "Description too short (minimum 50 characters)")
				meta.Recommendation = "Write a compelling description between 150-160 characters"
			} else if len(meta.Content) > 160 {
				meta.IsValid = false
				meta.Issues = append(meta.Issues, "Description too long (maximum 160 characters)")
				meta.Recommendation = "Trim description to 150-160 characters"
			}

		case meta.Name == "keywords":
			keywords := strings.Split(meta.Content, ",")
			if len(keywords) > 10 {
				meta.IsValid = false
				meta.Issues = append(meta.Issues, "Too many keywords (maximum 10 recommended)")
				meta.Recommendation = "Limit keywords to 5-10 most relevant terms"
			}

		case meta.Name == "robots":
			ms.results.RobotsMeta = meta.Content
			// Check for blocking directives
			if strings.Contains(meta.Content, "noindex") {
				ms.addIssue("high", "Page has noindex directive",
					"Remove noindex if you want this page to appear in search results")
			}

		case meta.Name == "viewport":
			foundViewport = true
			if !strings.Contains(meta.Content, "width=device-width") {
				meta.IsValid = false
				meta.Issues = append(meta.Issues, "Viewport missing width=device-width")
				meta.Recommendation = "Add width=device-width, initial-scale=1"
			}

		case strings.HasPrefix(meta.Property, "og:"):
			ms.results.OpenGraph[meta.Property] = meta.Content

		case strings.HasPrefix(meta.Name, "twitter:"):
			ms.results.TwitterCard[meta.Name] = meta.Content
		}

		ms.results.MetaTags = append(ms.results.MetaTags, meta)
	})

	// Add issues for missing important meta tags
	if !foundDesc {
		ms.addIssue("high", "Missing meta description",
			"Add a meta description to improve click-through rates from search results")
	}
	if !foundViewport {
		ms.addIssue("medium", "Missing viewport meta tag",
			"Add viewport meta tag for proper mobile rendering")
	}
}

// analyzeOpenGraph analyzes Open Graph meta tags
func (ms *MetaScanner) analyzeOpenGraph(doc *goquery.Document) {
	required := []string{"og:title", "og:type", "og:url", "og:image"}
	missing := []string{}

	for _, tag := range required {
		if _, exists := ms.results.OpenGraph[tag]; !exists {
			missing = append(missing, tag)
		}
	}

	if len(missing) > 0 {
		ms.addIssue("medium", fmt.Sprintf("Missing Open Graph tags: %v", missing),
			"Add Open Graph tags for better social media sharing")
	}

	// Validate image
	if imgURL, exists := ms.results.OpenGraph["og:image"]; exists {
		if !strings.HasPrefix(imgURL, "http") {
			ms.addIssue("low", "Open Graph image URL is relative",
				"Use absolute URLs for Open Graph images")
		}
	}
}

// analyzeTwitterCard analyzes Twitter Card meta tags
func (ms *MetaScanner) analyzeTwitterCard(doc *goquery.Document) {
	if len(ms.results.TwitterCard) == 0 {
		ms.addIssue("low", "Missing Twitter Card tags",
			"Add Twitter Card tags for better sharing on Twitter")
		return
	}

	// Check for required Twitter Card tags
	if _, exists := ms.results.TwitterCard["twitter:card"]; !exists {
		ms.addIssue("medium", "Missing twitter:card type",
			"Specify card type: summary, summary_large_image, app, or player")
	}
}

// analyzeCanonical analyzes canonical link
func (ms *MetaScanner) analyzeCanonical(doc *goquery.Document) {
	doc.Find("link[rel='canonical']").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			ms.results.CanonicalURL = href

			// Validate canonical URL
			if !strings.HasPrefix(href, "https://") && !strings.HasPrefix(href, "http://") {
				ms.addIssue("medium", "Canonical URL is not absolute",
					"Use absolute URLs for canonical tags to avoid confusion")
			}
		}
	})

	// Check for missing canonical
	if ms.results.CanonicalURL == "" {
		ms.addIssue("medium", "Missing canonical URL",
			"Add canonical URL to prevent duplicate content issues")
	}
}

// analyzeRobots analyzes robots meta tag
func (ms *MetaScanner) analyzeRobots(doc *goquery.Document) {
	doc.Find("meta[name='robots']").Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists {
			ms.results.RobotsMeta = content

			// Check for problematic directives
			if strings.Contains(content, "noindex") {
				ms.addIssue("high", "Page has noindex directive",
					"Remove noindex if you want this page indexed")
			}
			if strings.Contains(content, "nofollow") {
				ms.addIssue("medium", "Page has nofollow directive",
					"Consider removing nofollow for important pages to pass link equity")
			}
		}
	})
}

// analyzeSchemaMarkup extracts and validates schema.org markup
func (ms *MetaScanner) analyzeSchemaMarkup(doc *goquery.Document) {
	// Check for JSON-LD
	doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection) {
		content := strings.TrimSpace(s.Text())
		if content == "" {
			return
		}

		schema := SchemaMeta{
			Raw:     content,
			IsValid: true,
			Data:    make(map[string]interface{}),
		}

		// Parse JSON
		if err := json.Unmarshal([]byte(content), &schema.Data); err != nil {
			schema.IsValid = false
			schema.Validation = append(schema.Validation, fmt.Sprintf("Invalid JSON: %v", err))
		} else {
			// Extract type
			if typ, ok := schema.Data["@type"]; ok {
				schema.Type = fmt.Sprintf("%v", typ)
			}
			if ctx, ok := schema.Data["@context"]; ok {
				schema.Context = fmt.Sprintf("%v", ctx)
			}

			// Validate common schema types
			ms.validateSchema(&schema)
		}

		ms.results.SchemaMarkup = append(ms.results.SchemaMarkup, schema)
	})

	// Check for microdata
	doc.Find("[itemscope]").Each(func(i int, s *goquery.Selection) {
		if itemType, exists := s.Attr("itemtype"); exists {
			schema := SchemaMeta{
				Type:    itemType,
				IsValid: true,
			}
			ms.results.SchemaMarkup = append(ms.results.SchemaMarkup, schema)
		}
	})

	// Recommend schema if none found
	if len(ms.results.SchemaMarkup) == 0 {
		ms.addIssue("medium", "No schema markup found",
			"Add structured data to help search engines understand your content")
	}
}

// validateSchema validates common schema types
func (ms *MetaScanner) validateSchema(schema *SchemaMeta) {
	switch schema.Type {
	case "Product":
		required := []string{"name", "offers"}
		for _, field := range required {
			if _, ok := schema.Data[field]; !ok {
				schema.IsValid = false
				schema.Validation = append(schema.Validation, fmt.Sprintf("Missing required field: %s", field))
			}
		}
	case "Article", "NewsArticle", "BlogPosting":
		required := []string{"headline", "author", "datePublished"}
		for _, field := range required {
			if _, ok := schema.Data[field]; !ok {
				schema.IsValid = false
				schema.Validation = append(schema.Validation, fmt.Sprintf("Missing required field: %s", field))
			}
		}
	case "LocalBusiness":
		required := []string{"name", "address", "telephone"}
		for _, field := range required {
			if _, ok := schema.Data[field]; !ok {
				schema.IsValid = false
				schema.Validation = append(schema.Validation, fmt.Sprintf("Missing required field: %s", field))
			}
		}
	}
}

// analyzeImages analyzes and optimizes images
func (ms *MetaScanner) analyzeImages(doc *goquery.Document) {
	var wg sync.WaitGroup
	imgChan := make(chan ImageMeta, 100)

	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		wg.Add(1)
		go func(selection *goquery.Selection) {
			defer wg.Done()

			img := ImageMeta{
				Issues: []string{},
			}

			// Get image attributes
			src, exists := selection.Attr("src")
			if !exists {
				img.Issues = append(img.Issues, "Missing src attribute")
				imgChan <- img
				return
			}
			img.URL = src

			// Get alt text
			alt, exists := selection.Attr("alt")
			if exists {
				img.AltText = alt
				if alt == "" {
					img.MissingAlt = true
					img.Issues = append(img.Issues, "Empty alt text")
				}
			} else {
				img.MissingAlt = true
				img.Issues = append(img.Issues, "Missing alt text")
			}

			// Check for lazy loading
			_, img.LazyLoaded = selection.Attr("loading")
			if !img.LazyLoaded {
				img.Issues = append(img.Issues, "Consider adding lazy loading")
			}

			// Get title
			img.Title, _ = selection.Attr("title")

			// If configured, download and optimize image
			if ms.config.OptimizeImages && strings.HasPrefix(src, "http") {
				ms.optimizeImage(&img)
			}

			imgChan <- img
		}(s)
	})

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(imgChan)
	}()

	// Collect results
	for img := range imgChan {
		ms.results.Images = append(ms.results.Images, img)
	}

	// Add summary issues
	missingAlt := 0
	for _, img := range ms.results.Images {
		if img.MissingAlt {
			missingAlt++
		}
	}

	if missingAlt > 0 {
		ms.addIssue("medium", fmt.Sprintf("%d images missing alt text", missingAlt),
			"Add descriptive alt text to all images for accessibility and SEO")
	}
}

// optimizeImage downloads and optimizes an image
func (ms *MetaScanner) optimizeImage(img *ImageMeta) {
	// Resolve absolute URL
	imgURL, err := ms.resolveURL(img.URL)
	if err != nil {
		img.Issues = append(img.Issues, fmt.Sprintf("Invalid URL: %v", err))
		return
	}

	// Download image
	resp, err := http.Get(imgURL)
	if err != nil {
		img.Issues = append(img.Issues, fmt.Sprintf("Failed to download: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		img.Issues = append(img.Issues, fmt.Sprintf("HTTP error: %d", resp.StatusCode))
		return
	}

	// Read image data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		img.Issues = append(img.Issues, fmt.Sprintf("Failed to read: %v", err))
		return
	}

	img.FileSize = int64(len(data))

	// Decode image for dimensions
	reader := bytes.NewReader(data)
	config, format, err := image.DecodeConfig(reader)
	if err == nil {
		img.Width = config.Width
		img.Height = config.Height
		img.Format = format
	}

	// Optimize if beneficial
	if len(data) > 10240 { // > 10KB
		optimized, webpGenerated, err := ms.compressImage(data, format)
		if err == nil && len(optimized) < len(data) {
			img.OptimizedSize = int64(len(optimized))
			img.WebPGenerated = webpGenerated

			// Save optimized image
			filename := fmt.Sprintf("%x_optimized%s", md5.Sum([]byte(img.URL)),
				getExtension(format))
			path := filepath.Join(ms.config.OutputDir, "optimized", filename)
			os.MkdirAll(filepath.Dir(path), 0755)
			os.WriteFile(path, optimized, 0644)
		}
	}
}

// compressImage compresses an image
func (ms *MetaScanner) compressImage(data []byte, format string) ([]byte, bool, error) {
	// Decode image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, false, err
	}

	var buf bytes.Buffer
	webpGenerated := false

	// Compress based on format
	switch format {
	case "jpeg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(&buf, img)
	default:
		return data, false, nil
	}

	if err != nil {
		return data, false, err
	}

	// Check if compression helped
	if buf.Len() < len(data) {
		// Also try to generate WebP if format is suitable
		if format == "jpeg" || format == "png" {
			webpGenerated = true
		}
		return buf.Bytes(), webpGenerated, nil
	}

	return data, false, nil
}

// analyzeLinks analyzes all links on the page
func (ms *MetaScanner) analyzeLinks(doc *goquery.Document) {
	var wg sync.WaitGroup
	linkChan := make(chan LinkMeta, 100)

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		wg.Add(1)
		go func(selection *goquery.Selection) {
			defer wg.Done()

			link := LinkMeta{
				Issues: []string{},
				Follow: true,
			}

			// Get href
			href, exists := selection.Attr("href")
			if !exists {
				link.Issues = append(link.Issues, "Missing href")
				linkChan <- link
				return
			}
			link.HREF = href

			// Get rel attributes
			if rel, exists := selection.Attr("rel"); exists {
				link.Rel = rel
				if strings.Contains(rel, "nofollow") {
					link.Follow = false
				}
			}

			// Get type and title
			link.Type, _ = selection.Attr("type")
			link.Title, _ = selection.Attr("title")

			// Check if internal
			parsed, err := url.Parse(href)
			if err == nil {
				if parsed.IsAbs() {
					link.IsInternal = parsed.Host == ms.baseURL.Host
				} else {
					link.IsInternal = true
				}
			}

			// Check if broken (if configured)
			if ms.config.CheckBrokenLinks && strings.HasPrefix(href, "http") {
				ms.checkLinkBroken(&link)
			}

			linkChan <- link
		}(s)
	})

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(linkChan)
	}()

	// Collect results
	brokenCount := 0
	for link := range linkChan {
		if link.IsBroken {
			brokenCount++
		}
		ms.results.Links = append(ms.results.Links, link)
	}

	// Add summary issues
	if brokenCount > 0 {
		ms.addIssue("high", fmt.Sprintf("%d broken links found", brokenCount),
			"Fix or remove broken links to improve user experience")
	}
}

// checkLinkBroken checks if a link is broken
func (ms *MetaScanner) checkLinkBroken(link *LinkMeta) {
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Head(link.HREF)
	if err != nil {
		link.IsBroken = true
		return
	}
	defer resp.Body.Close()

	link.StatusCode = resp.StatusCode
	link.IsBroken = resp.StatusCode >= 400
}

// analyzeContent analyzes content for SEO optimization
func (ms *MetaScanner) analyzeContent(doc *goquery.Document) {
	content := ContentMetrics{
		KeywordDensity:   make(map[string]float64),
		HeadingStructure: make(map[string]int),
		Issues:           []string{},
	}

	// Get all text content
	text := doc.Find("p, h1, h2, h3, h4, h5, h6, li, td, th").Text()
	words := strings.Fields(text)
	content.WordCount = len(words)

	// Check content length
	if content.WordCount < 300 {
		content.Issues = append(content.Issues, "Thin content (less than 300 words)")
		ms.addIssue("medium", "Thin content detected",
			"Aim for at least 300 words per page for better rankings")
	}

	// Analyze headings
	doc.Find("h1").Each(func(i int, s *goquery.Selection) {
		content.HeadingStructure["h1"]++
		text := strings.TrimSpace(s.Text())
		if text == "" {
			content.Issues = append(content.Issues, "Empty H1 tag")
		}
	})

	doc.Find("h2").Each(func(i int, s *goquery.Selection) {
		content.HeadingStructure["h2"]++
	})

	doc.Find("h3").Each(func(i int, s *goquery.Selection) {
		content.HeadingStructure["h3"]++
	})

	// Validate heading structure
	if content.HeadingStructure["h1"] == 0 {
		content.Issues = append(content.Issues, "Missing H1 tag")
		ms.addIssue("high", "Missing H1 tag", "Add one H1 tag containing the main topic")
	} else if content.HeadingStructure["h1"] > 1 {
		content.MultipleH1 = true
		content.Issues = append(content.Issues, "Multiple H1 tags found")
		ms.addIssue("medium", "Multiple H1 tags", "Use only one H1 tag per page")
	} else {
		content.HasH1 = true
	}

	// Calculate keyword density
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true, "was": true,
		"are": true, "were": true, "be": true, "been": true, "being": true,
	}

	wordFreq := make(map[string]int)
	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".,!?;:\"'()[]{}"))
		if !stopWords[word] && len(word) > 2 {
			wordFreq[word]++
		}
	}

	if content.WordCount > 0 {
		for word, count := range wordFreq {
			density := float64(count) / float64(content.WordCount) * 100
			if density >= 0.5 { // Only include words with density >= 0.5%
				content.KeywordDensity[word] = density
			}
		}

		// Find primary keyword
		var maxDensity float64
		for word, density := range content.KeywordDensity {
			if density > maxDensity {
				maxDensity = density
				content.PrimaryKeyword = word
			}
		}
	}

	// Calculate readability (Flesch-Kincaid)
	sentences := strings.Count(text, ".") + strings.Count(text, "!") + strings.Count(text, "?")
	if sentences > 0 && content.WordCount > 0 {
		avgWordsPerSentence := float64(content.WordCount) / float64(sentences)
		syllables := ms.countSyllables(text)
		avgSyllablesPerWord := float64(syllables) / float64(content.WordCount)

		// Flesch Reading Ease
		content.ReadabilityScore = 206.835 - 1.015*avgWordsPerSentence - 84.6*avgSyllablesPerWord
	}

	ms.results.Content = content
}

// countSyllables approximates syllable count for readability
func (ms *MetaScanner) countSyllables(text string) int {
	count := 0
	words := strings.Fields(text)

	for _, word := range words {
		word = strings.ToLower(word)
		syllables := 0
		prevVowel := false

		for i := 0; i < len(word); i++ {
			isVowel := strings.ContainsRune("aeiouy", rune(word[i]))
			if isVowel && !prevVowel {
				syllables++
			}
			prevVowel = isVowel
		}

		// Adjust for silent e
		if strings.HasSuffix(word, "e") {
			syllables--
		}

		// Ensure at least one syllable per word
		if syllables < 1 {
			syllables = 1
		}

		count += syllables
	}

	return count
}

// analyzePageSpeed analyzes page speed metrics
func (ms *MetaScanner) analyzePageSpeed(doc *goquery.Document, body []byte) {
	metrics := PageSpeedMetrics{
		Score: 100,
	}

	// HTML size
	metrics.HTMLSize = int64(len(body))

	// Minify HTML if configured
	if ms.config.MinifyContent {
		html, err := doc.Html()
		if err == nil {
			minified, err := ms.minifier.String("text/html", html)
			if err == nil {
				metrics.HTMLMinified = int64(len(minified))
			}
		}
	}

	// Count and size of CSS/JS
	doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
		metrics.TotalRequests++
	})

	doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		metrics.TotalRequests++
	})

	// Check for inline CSS/JS
	doc.Find("style").Each(func(i int, s *goquery.Selection) {
		css := s.Text()
		metrics.CSSSize += int64(len(css))

		if ms.config.MinifyContent {
			minified, _ := ms.minifier.String("text/css", css)
			metrics.CSSMinified += int64(len(minified))
		}
	})

	doc.Find("script:not([src])").Each(func(i int, s *goquery.Selection) {
		js := s.Text()
		metrics.JSSize += int64(len(js))

		if ms.config.MinifyContent {
			minified, _ := ms.minifier.String("text/javascript", js)
			metrics.JSMinified += int64(len(minified))
		}
	})

	// Calculate score based on various factors
	if metrics.HTMLSize > 50000 { // > 50KB
		metrics.Score -= 10
		ms.addIssue("low", "Large HTML size", "Consider minifying HTML")
	}

	if metrics.CSSSize > 50000 { // > 50KB
		metrics.Score -= 10
		ms.addIssue("medium", "Large CSS size", "Minify and combine CSS files")
	}

	if metrics.JSSize > 100000 { // > 100KB
		metrics.Score -= 10
		ms.addIssue("medium", "Large JavaScript size", "Minify and defer JavaScript")
	}

	if metrics.TotalRequests > 20 {
		metrics.Score -= 10
		ms.addIssue("medium", fmt.Sprintf("Too many requests (%d)", metrics.TotalRequests),
			"Reduce the number of CSS and JavaScript files")
	}

	if !metrics.CompressionEnabled {
		metrics.Score -= 10
		ms.addIssue("medium", "Compression not enabled", "Enable gzip compression")
	}

	if !metrics.CacheHeaders {
		metrics.Score -= 5
		ms.addIssue("low", "Cache headers not set", "Implement browser caching")
	}

	ms.results.PageSpeed = metrics
}

// analyzeMobile analyzes mobile responsiveness
func (ms *MetaScanner) analyzeMobile(doc *goquery.Document) {
	mobile := MobileMetrics{
		TapTargets: []string{},
		FontSizes:  []string{},
		Issues:     []string{},
		Score:      100,
	}

	// Check viewport
	doc.Find("meta[name='viewport']").Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists {
			mobile.ViewportConfigured = true
			if !strings.Contains(content, "width=device-width") {
				mobile.Issues = append(mobile.Issues, "Viewport missing width=device-width")
				mobile.Score -= 20
			}
		}
	})

	if !mobile.ViewportConfigured {
		mobile.Issues = append(mobile.Issues, "Viewport meta tag missing")
		mobile.Score -= 30
		ms.addIssue("high", "Missing viewport configuration",
			"Add viewport meta tag for proper mobile rendering")
	}

	// Count media queries
	doc.Find("style").Each(func(i int, s *goquery.Selection) {
		css := s.Text()
		queries := strings.Count(css, "@media")
		mobile.MediaQueries += queries
	})

	if mobile.MediaQueries == 0 {
		mobile.Issues = append(mobile.Issues, "No media queries found")
		mobile.Score -= 20
		ms.addIssue("medium", "No responsive design detected",
			"Implement responsive design with media queries")
	}

	// Check font sizes
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		if style, exists := s.Attr("style"); exists {
			if strings.Contains(style, "font-size") {
				re := regexp.MustCompile(`font-size:\s*(\d+)px`)
				if matches := re.FindStringSubmatch(style); len(matches) > 1 {
					size := parseInt(matches[1])
					if size < 12 {
						mobile.FontSizes = append(mobile.FontSizes, fmt.Sprintf("%dpx", size))
					}
				}
			}
		}
	})

	if len(mobile.FontSizes) > 0 {
		mobile.Issues = append(mobile.Issues, "Text too small on mobile")
		mobile.Score -= 10
	}

	ms.results.Mobile = mobile
}

// calculateScore calculates overall SEO score
func (ms *MetaScanner) calculateScore() {
	score := 100

	// Deduct based on issues severity
	for _, issue := range ms.results.Issues {
		if strings.Contains(issue, "[HIGH]") {
			score -= 15
		} else if strings.Contains(issue, "[MEDIUM]") {
			score -= 10
		} else {
			score -= 5
		}
	}

	// Add points for good practices
	if ms.results.CanonicalURL != "" {
		score += 5
	}
	if len(ms.results.SchemaMarkup) > 0 {
		score += 10
	}
	if ms.results.PageSpeed.Score > 80 {
		score += 5
	}
	if ms.results.Mobile.Score > 80 {
		score += 5
	}
	if ms.results.Content.WordCount > 1000 {
		score += 5
	}

	// Ensure score stays within 0-100
	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}

	ms.results.Score = score
}

// generateRecommendations generates actionable recommendations
func (ms *MetaScanner) generateRecommendations() {
	recs := []string{}

	// Title recommendations
	if ms.results.Title == "" {
		recs = append(recs, "Add a title tag (50-60 characters)")
	} else if len(ms.results.Title) < 30 {
		recs = append(recs, "Expand your title to 50-60 characters")
	}

	// Meta description recommendations
	hasDesc := false
	for _, meta := range ms.results.MetaTags {
		if meta.Name == "description" {
			hasDesc = true
			if len(meta.Content) < 50 {
				recs = append(recs, "Write a longer meta description (150-160 characters)")
			}
			break
		}
	}
	if !hasDesc {
		recs = append(recs, "Add a meta description to improve CTR")
	}

	// Image recommendations
	missingAlt := 0
	for _, img := range ms.results.Images {
		if img.MissingAlt {
			missingAlt++
		}
	}
	if missingAlt > 0 {
		recs = append(recs, fmt.Sprintf("Add alt text to %d images", missingAlt))
	}

	// Link recommendations
	brokenCount := 0
	for _, link := range ms.results.Links {
		if link.IsBroken {
			brokenCount++
		}
	}
	if brokenCount > 0 {
		recs = append(recs, fmt.Sprintf("Fix %d broken links", brokenCount))
	}

	// Schema recommendations
	if len(ms.results.SchemaMarkup) == 0 {
		recs = append(recs, "Add structured data (schema.org markup)")
	}

	// Content recommendations
	if ms.results.Content.WordCount < 300 {
		recs = append(recs, "Add more content (aim for 1000+ words)")
	}
	if !ms.results.Content.HasH1 {
		recs = append(recs, "Add an H1 heading")
	}
	if ms.results.Content.MultipleH1 {
		recs = append(recs, "Use only one H1 tag")
	}

	// Page speed recommendations
	if !ms.results.PageSpeed.CompressionEnabled {
		recs = append(recs, "Enable gzip compression")
	}
	if ms.results.PageSpeed.HTMLSize > 50000 {
		recs = append(recs, "Minify HTML")
	}
	if ms.results.PageSpeed.CSSSize > 50000 {
		recs = append(recs, "Minify and combine CSS files")
	}
	if ms.results.PageSpeed.JSSize > 100000 {
		recs = append(recs, "Minify and defer JavaScript")
	}

	// Mobile recommendations
	if !ms.results.Mobile.ViewportConfigured {
		recs = append(recs, "Configure viewport meta tag for mobile")
	}
	if ms.results.Mobile.MediaQueries == 0 {
		recs = append(recs, "Implement responsive design with media queries")
	}

	ms.results.Recommendations = recs
}

// addIssue adds an issue with severity
func (ms *MetaScanner) addIssue(severity, issue, recommendation string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	formatted := fmt.Sprintf("[%s] %s", strings.ToUpper(severity), issue)
	ms.results.Issues = append(ms.results.Issues, formatted)
	ms.results.Recommendations = append(ms.results.Recommendations, recommendation)
}

// resolveURL resolves a relative URL to absolute
func (ms *MetaScanner) resolveURL(ref string) (string, error) {
	refURL, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	return ms.baseURL.ResolveReference(refURL).String(), nil
}

// parseInt converts string to int
func parseInt(s string) int {
	var result int
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			result = result*10 + int(s[i]-'0')
		} else {
			break
		}
	}
	return result
}

// getExtension returns file extension for format
func getExtension(format string) string {
	switch format {
	case "jpeg":
		return ".jpg"
	case "png":
		return ".png"
	case "gif":
		return ".gif"
	default:
		return ".bin"
	}
}

// ExportJSON exports scan results as JSON
func (ms *MetaScanner) ExportJSON() ([]byte, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return json.MarshalIndent(ms.results, "", "  ")
}

// SaveResults saves scan results to file
func (ms *MetaScanner) SaveResults() error {
	data, err := ms.ExportJSON()
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("scan_%s.json", time.Now().Format("20060102_150405"))
	path := filepath.Join(ms.config.OutputDir, filename)

	return os.WriteFile(path, data, 0644)
}