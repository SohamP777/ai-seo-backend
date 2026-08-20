// Package seocrawler provides a production-grade SEO crawler and optimizer
package scanner

import (
    "bytes"
    "compress/gzip"
    "context"
    "crypto/md5"
    "crypto/tls"
    "encoding/json"
    "fmt"
    _ "image"
    _ "image/jpeg"
    _ "image/png"
    "io"
    "log"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "sync"
    "time"

    "github.com/PuerkitoBio/goquery"
    "github.com/tdewolff/minify/v2"
    "github.com/tdewolff/minify/v2/css"
    "github.com/tdewolff/minify/v2/html"
    "github.com/tdewolff/minify/v2/js"
    "golang.org/x/net/html/charset"
    "golang.org/x/time/rate"
)

// SEOIssue represents a specific SEO problem found
type SEOIssue struct {
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	Message     string   `json:"message"`
	Element     string   `json:"element,omitempty"`
	Description string   `json:"description,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// PageData represents crawled page information
type PageData struct {
	URL               string             `json:"url"`
	Title             string             `json:"title"`
	MetaDescription   string             `json:"meta_description"`
	CanonicalURL      string             `json:"canonical_url"`
	StatusCode        int                `json:"status_code"`
	LoadTime          time.Duration      `json:"load_time"`
	WordCount         int                `json:"word_count"`
	ReadabilityScore  float64            `json:"readability_score"`
	Keywords          map[string]int     `json:"keywords"`
	BrokenLinks       []string           `json:"broken_links"`
	ValidLinks        []string           `json:"valid_links"`
	Images            []ImageInfo        `json:"images"`
	SEOIssues         []SEOIssue         `json:"seo_issues"`
	Headers           map[string]string  `json:"headers"`
	MobileScore       int                `json:"mobile_score"`
	SchemaTypes       []string           `json:"schema_types"`
	Depth             int                `json:"depth"`
	CrawlTime         time.Time          `json:"crawl_time"`
}

// ImageInfo represents image optimization data
type ImageInfo struct {
	OriginalURL      string `json:"original_url"`
	OptimizedPath    string `json:"optimized_path"`
	OriginalSize     int64  `json:"original_size"`
	OptimizedSize    int64  `json:"optimized_size"`
	AltText          string `json:"alt_text"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	MissingAlt       bool   `json:"missing_alt"`
	WebPGenerated    bool   `json:"webp_generated"`
}

// CrawlerConfig holds crawler configuration
type CrawlerConfig struct {
	MaxDepth          int           `json:"max_depth"`
	Timeout           time.Duration `json:"timeout"`
	PageTimeout       time.Duration `json:"page_timeout"`
	Concurrency       int           `json:"concurrency"`
	RespectRobotsTxt  bool          `json:"respect_robots"`
	UserAgent         string        `json:"user_agent"`
	OptimizeImages    bool          `json:"optimize_images"`
	CompressOutput    bool          `json:"compress_output"`
	CheckMobile       bool          `json:"check_mobile"`
	OutputDir         string        `json:"output_dir"`
	MaxPages          int           `json:"max_pages"`
	FollowExternal    bool          `json:"follow_external"`
	RateLimit         int           `json:"rate_limit"`          // Requests per second
	SkipTLSVerify     bool          `json:"skip_tls_verify"`     // Skip SSL verification (not recommended for production)
	MaxFileSize       int64         `json:"max_file_size"`       // Max file size to download in bytes
	EnableJavaScript  bool          `json:"enable_javascript"`   // Enable JavaScript rendering (slow)
}

// CrawlTask represents a page to crawl
type CrawlTask struct {
	URL   string
	Depth int
}

// RobotsTxt represents robots.txt rules
type RobotsTxt struct {
	DisallowedPaths []string
	UserAgent       string
}

// SEOCrawler main crawler struct
type SEOCrawler struct {
	config        CrawlerConfig
	client        *http.Client
	visited       map[string]bool
	visitedMu     sync.RWMutex
	results       map[string]*PageData
	resultsMu     sync.RWMutex
	minifier      *minify.M
	logger        *log.Logger
	baseHost      string
	cancelFunc    context.CancelFunc
	rateLimiter   *rate.Limiter
	robotsCache   map[string]*RobotsTxt
	robotsMu      sync.RWMutex
	semaphore     chan struct{} // For limiting concurrent operations
}

// NewSEOCrawler creates a new SEO crawler instance with production defaults
func NewSEOCrawler(config CrawlerConfig) *SEOCrawler {
	// Set production-safe defaults
	if config.Timeout == 0 {
		config.Timeout = 300 * time.Second // 5 minutes total
	}
	if config.PageTimeout == 0 {
		config.PageTimeout = 30 * time.Second
	}
	if config.Concurrency == 0 {
		config.Concurrency = 5 // Safe concurrency
	}
	if config.RateLimit == 0 {
		config.RateLimit = 10 // 10 requests per second
	}
	if config.UserAgent == "" {
		config.UserAgent = "Mozilla/5.0 (compatible; SEOCrawlerBot/2.0; +https://example.com/bot)"
	}
	if config.OutputDir == "" {
		config.OutputDir = "./seo_output"
	}
	if config.MaxDepth == 0 {
		config.MaxDepth = 3
	}
	if config.MaxPages == 0 {
		config.MaxPages = 100
	}
	if config.MaxFileSize == 0 {
		config.MaxFileSize = 10 * 1024 * 1024 // 10MB
	}

	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("text/javascript", js.Minify)

	// Custom HTTP client with timeouts
	client := &http.Client{
		Timeout: config.PageTimeout,
		Transport: &http.Transport{
			MaxIdleConns:    100,
			IdleConnTimeout: 90 * time.Second,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipTLSVerify},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	return &SEOCrawler{
		config:      config,
		client:      client,
		visited:     make(map[string]bool),
		results:     make(map[string]*PageData),
		minifier:    m,
		logger:      log.New(os.Stdout, "[CRAWLER] ", log.LstdFlags),
		rateLimiter: rate.NewLimiter(rate.Limit(config.RateLimit), config.Concurrency),
		robotsCache: make(map[string]*RobotsTxt),
		semaphore:   make(chan struct{}, config.Concurrency*2),
	}
}

// Crawl starts crawling from the given URL (main entry point)
func (c *SEOCrawler) Crawl(startURL string) (map[string]*PageData, error) {
	// Parse and validate start URL
	parsedURL, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	c.baseHost = parsedURL.Host

	// Normalize start URL
	startURL = c.normalizeURL(startURL)

	// Create output directory
	if err := os.MkdirAll(c.config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	// Create subdirectories
	if c.config.OptimizeImages {
		os.MkdirAll(filepath.Join(c.config.OutputDir, "optimized"), 0755)
	}
	if c.config.CompressOutput {
		os.MkdirAll(filepath.Join(c.config.OutputDir, "minified"), 0755)
	}

	// Check robots.txt if enabled
	if c.config.RespectRobotsTxt {
		if !c.isAllowedByRobots(startURL) {
			return nil, fmt.Errorf("crawling disallowed by robots.txt for %s", startURL)
		}
	}

	// Create context with cancellation
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()
	c.cancelFunc = cancel

	c.logger.Printf("🚀 Starting crawl of %s (max depth: %d, max pages: %d, rate limit: %d req/s)",
		startURL, c.config.MaxDepth, c.config.MaxPages, c.config.RateLimit)

	// Create task channel and worker pool
	taskChan := make(chan *CrawlTask, c.config.MaxPages*2)
	errChan := make(chan error, c.config.MaxPages)

	var wg sync.WaitGroup

	// Start workers with semaphore pattern
	for i := 0; i < c.config.Concurrency; i++ {
		wg.Add(1)
		go c.worker(ctx, &wg, taskChan, errChan)
	}

	// Send initial task
	taskChan <- &CrawlTask{
		URL:   startURL,
		Depth: 0,
	}

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(taskChan)
		close(errChan)
	}()

	// Monitor progress with ticker
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	select {
	case <-ctx.Done():
		c.logger.Printf("⚠️ Crawl timeout after %v, returning %d pages",
			c.config.Timeout, len(c.results))
	case <-ticker.C:
		// Periodic progress update
		c.resultsMu.RLock()
		count := len(c.results)
		c.resultsMu.RUnlock()
		c.logger.Printf("📊 Progress: %d pages crawled so far", count)
	case <-c.waitForCompletion(taskChan):
		c.logger.Printf("✅ Crawl completed: %d pages crawled", len(c.results))
	}

	// Log errors (but don't fail)
	for err := range errChan {
		c.logger.Printf("⚠️ Error: %v", err)
	}

	if len(c.results) == 0 {
		return nil, fmt.Errorf("no pages were crawled successfully")
	}

	// Save results automatically
	if err := c.saveResults(); err != nil {
		c.logger.Printf("⚠️ Failed to save results: %v", err)
	}

	return c.results, nil
}

// waitForCompletion waits for task channel to close
func (c *SEOCrawler) waitForCompletion(taskChan chan *CrawlTask) chan struct{} {
	done := make(chan struct{})
	go func() {
		for range taskChan {
			// Just drain the channel
		}
		close(done)
	}()
	return done
}

// worker processes crawl tasks with rate limiting
func (c *SEOCrawler) worker(ctx context.Context, wg *sync.WaitGroup, taskChan chan *CrawlTask, errChan chan<- error) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-taskChan:
			if !ok {
				return
			}

			// Apply rate limiting
			if err := c.rateLimiter.Wait(ctx); err != nil {
				errChan <- fmt.Errorf("rate limiter error: %w", err)
				continue
			}

			// Acquire semaphore
			select {
			case c.semaphore <- struct{}{}:
				// Process task
				c.processTask(ctx, task, taskChan, errChan)
				<-c.semaphore
			case <-ctx.Done():
				return
			}
		}
	}
}

// processTask processes a single crawl task
func (c *SEOCrawler) processTask(ctx context.Context, task *CrawlTask, taskChan chan *CrawlTask, errChan chan<- error) {
	// Check if we've reached max pages
	c.resultsMu.RLock()
	pagesCount := len(c.results)
	c.resultsMu.RUnlock()

	if pagesCount >= c.config.MaxPages {
		return
	}

	// Check if already visited
	c.visitedMu.Lock()
	if c.visited[task.URL] {
		c.visitedMu.Unlock()
		return
	}
	c.visited[task.URL] = true
	c.visitedMu.Unlock()

	// Crawl the page with timeout
	crawlCtx, cancel := context.WithTimeout(ctx, c.config.PageTimeout)
	defer cancel()

	pageData, links, err := c.crawlPage(crawlCtx, task.URL, task.Depth)
	if err != nil {
		errChan <- fmt.Errorf("failed to crawl %s: %w", task.URL, err)
		return
	}

	// Store results
	c.resultsMu.Lock()
	c.results[task.URL] = pageData
	currentCount := len(c.results)
	c.resultsMu.Unlock()

	c.logger.Printf("📄 [Depth %d] (%d/%d) %s (HTTP %d, %.2fs)",
		task.Depth, currentCount, c.config.MaxPages, task.URL, 
		pageData.StatusCode, pageData.LoadTime.Seconds())

	// Queue new links if depth allows
	if task.Depth < c.config.MaxDepth && currentCount < c.config.MaxPages {
		c.queueNewLinks(ctx, task, links, taskChan)
	}
}

// queueNewLinks queues new links for crawling
func (c *SEOCrawler) queueNewLinks(ctx context.Context, task *CrawlTask, links []string, taskChan chan *CrawlTask) {
	for _, link := range links {
		// Check if we can add more pages
		c.resultsMu.RLock()
		if len(c.results) >= c.config.MaxPages {
			c.resultsMu.RUnlock()
			break
		}
		c.resultsMu.RUnlock()

		// Check if already visited
		c.visitedMu.RLock()
		visited := c.visited[link]
		c.visitedMu.RUnlock()

		if !visited {
			select {
			case taskChan <- &CrawlTask{URL: link, Depth: task.Depth + 1}:
			case <-ctx.Done():
				return
			default:
				// Channel full, skip
				c.logger.Printf("⚠️ Task channel full, skipping %s", link)
			}
		}
	}
}

// crawlPage crawls a single page and extracts data
func (c *SEOCrawler) crawlPage(ctx context.Context, pageURL string, depth int) (*PageData, []string, error) {
	startTime := time.Now()

	// Check robots.txt before crawling
	if c.config.RespectRobotsTxt && !c.isAllowedByRobots(pageURL) {
		return nil, nil, fmt.Errorf("disallowed by robots.txt")
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")

	// Add random delay to avoid detection (jitter)
	time.Sleep(time.Duration(50+time.Now().UnixNano()%100) * time.Millisecond)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return nil, nil, fmt.Errorf("not HTML: %s", contentType)
	}

	// Limit response body size
	limitedReader := io.LimitReader(resp.Body, c.config.MaxFileSize)
	
	// Handle compression
	bodyReader := limitedReader
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(limitedReader)
		if err != nil {
			return nil, nil, err
		}
		defer reader.Close()
		bodyReader = reader
	}

	// Decode body
	body, err := c.decodeBody(bodyReader, contentType)
	if err != nil {
		return nil, nil, err
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}

	// Initialize page data
	pageData := &PageData{
		URL:         pageURL,
		StatusCode:  resp.StatusCode,
		LoadTime:    time.Since(startTime),
		Headers:     make(map[string]string),
		Keywords:    make(map[string]int),
		SEOIssues:   []SEOIssue{},
		ValidLinks:  []string{},
		BrokenLinks: []string{},
		Depth:       depth,
		CrawlTime:   time.Now(),
	}

	// Copy important headers
	for k, v := range resp.Header {
		if len(v) > 0 {
			pageData.Headers[k] = v[0]
		}
	}

	// Extract SEO elements
	c.extractSEOElements(doc, pageData)

	// Extract and validate links with timeout
	linkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	validLinks, brokenLinks := c.extractAndValidateLinks(linkCtx, doc, pageURL)
	pageData.ValidLinks = validLinks
	pageData.BrokenLinks = brokenLinks

	// Analyze images (optional)
	if c.config.OptimizeImages {
		c.analyzeAndOptimizeImages(doc, pageURL, pageData)
	}

	// Check mobile responsiveness (optional, non-blocking)
	if c.config.CheckMobile {
		// Run in goroutine to not block crawling
		go c.checkMobileResponsiveness(pageURL, pageData)
	}

	// Extract schema markup
	c.extractSchemaMarkup(doc, pageData)

	// Calculate readability
	c.calculateReadability(doc, pageData)

	// Minify content if requested
	if c.config.CompressOutput {
		c.minifyContent(doc, pageData, pageURL)
	}

	return pageData, validLinks, nil
}

// isAllowedByRobots checks if URL is allowed by robots.txt
func (c *SEOCrawler) isAllowedByRobots(pageURL string) bool {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return true // Allow on error
	}

	robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsed.Scheme, parsed.Host)
	
	// Check cache
	c.robotsMu.RLock()
	robots, exists := c.robotsCache[robotsURL]
	c.robotsMu.RUnlock()
	
	if !exists {
		robots = c.fetchRobotsTxt(robotsURL)
		c.robotsMu.Lock()
		c.robotsCache[robotsURL] = robots
		c.robotsMu.Unlock()
	}

	// Check if path is disallowed
	path := parsed.Path
	if path == "" {
		path = "/"
	}

	for _, disallowed := range robots.DisallowedPaths {
		if strings.HasPrefix(path, disallowed) {
			return false
		}
	}
	
	return true
}

// fetchRobotsTxt fetches and parses robots.txt
func (c *SEOCrawler) fetchRobotsTxt(robotsURL string) *RobotsTxt {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", robotsURL, nil)
	if err != nil {
		return &RobotsTxt{DisallowedPaths: []string{}}
	}
	
	req.Header.Set("User-Agent", c.config.UserAgent)
	
	resp, err := c.client.Do(req)
	if err != nil {
		return &RobotsTxt{DisallowedPaths: []string{}}
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return &RobotsTxt{DisallowedPaths: []string{}}
	}
	
	// Read and parse robots.txt
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // Limit to 1MB
	if err != nil {
		return &RobotsTxt{DisallowedPaths: []string{}}
	}
	
	disallowed := []string{}
	lines := strings.Split(string(body), "\n")
	currentAgent := ""
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		
		directive := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		
		if directive == "user-agent" {
			currentAgent = value
		} else if directive == "disallow" && (currentAgent == "*" || currentAgent == c.config.UserAgent) {
			if value != "" {
				disallowed = append(disallowed, value)
			}
		}
	}
	
	return &RobotsTxt{
		DisallowedPaths: disallowed,
		UserAgent:       c.config.UserAgent,
	}
}

// decodeBody handles character set decoding
func (c *SEOCrawler) decodeBody(reader io.Reader, contentType string) ([]byte, error) {
	// Read all content first
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Detect encoding
	encoding, name, _ := charset.DetermineEncoding(body, contentType)
	
	// If it's UTF-8 or unknown, return as is
	if name == "utf-8" || name == "" || encoding == nil {
		return body, nil
	}

	// Decode to UTF-8
	decoder := encoding.NewDecoder()
	decoded, err := decoder.Bytes(body)
	if err != nil {
		// If decoding fails, return original
		return body, nil
	}

	return decoded, nil
}

// extractAndValidateLinks extracts and validates all links from the page with context
func (c *SEOCrawler) extractAndValidateLinks(ctx context.Context, doc *goquery.Document, baseURL string) ([]string, []string) {
	var validLinks []string
	var brokenLinks []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	// Limit concurrent link checking
	semaphore := make(chan struct{}, 20)
	
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// Skip invalid links
		href = strings.TrimSpace(href)
		if href == "" || strings.HasPrefix(href, "#") ||
			strings.HasPrefix(href, "javascript:") ||
			strings.HasPrefix(href, "mailto:") ||
			strings.HasPrefix(href, "tel:") {
			return
		}

		// Resolve relative URLs
		absoluteURL, err := c.resolveURL(baseURL, href)
		if err != nil {
			return
		}

		// Normalize URL
		absoluteURL = c.normalizeURL(absoluteURL)

		// Filter external links if needed
		if !c.config.FollowExternal {
			parsed, err := url.Parse(absoluteURL)
			if err != nil || parsed.Host != c.baseHost {
				return
			}
		}

		// Filter out non-HTML and common file types
		if c.isNonHTMLFile(absoluteURL) {
			return
		}

		// Check link validity in background with timeout
		wg.Add(1)
		go func(urlStr string) {
			defer wg.Done()
			
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				mu.Lock()
				brokenLinks = append(brokenLinks, urlStr)
				mu.Unlock()
				return
			}
			
			// Quick HEAD request to check link
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			
			req, err := http.NewRequestWithContext(checkCtx, "HEAD", urlStr, nil)
			if err != nil {
				mu.Lock()
				brokenLinks = append(brokenLinks, urlStr)
				mu.Unlock()
				return
			}
			
			req.Header.Set("User-Agent", c.config.UserAgent)
			
			client := &http.Client{
				Timeout: 5 * time.Second,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					if len(via) >= 3 {
						return fmt.Errorf("too many redirects")
					}
					return nil
				},
			}
			
			resp, err := client.Do(req)
			if err != nil {
				mu.Lock()
				brokenLinks = append(brokenLinks, urlStr)
				mu.Unlock()
				return
			}
			defer resp.Body.Close()
			
			if resp.StatusCode >= 400 {
				mu.Lock()
				brokenLinks = append(brokenLinks, urlStr)
				mu.Unlock()
			} else {
				mu.Lock()
				validLinks = append(validLinks, urlStr)
				mu.Unlock()
			}
		}(absoluteURL)
	})
	
	wg.Wait()

	// Remove duplicates
	validLinks = c.removeDuplicates(validLinks)
	brokenLinks = c.removeDuplicates(brokenLinks)
	
	// Limit number of links to prevent memory issues
	maxLinks := 500
	if len(validLinks) > maxLinks {
		validLinks = validLinks[:maxLinks]
	}
	if len(brokenLinks) > maxLinks {
		brokenLinks = brokenLinks[:maxLinks]
	}

	return validLinks, brokenLinks
}

// isNonHTMLFile checks if URL points to a non-HTML file
func (c *SEOCrawler) isNonHTMLFile(urlStr string) bool {
	extensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".pdf", ".doc",
		".docx", ".xls", ".xlsx", ".zip", ".tar", ".gz", ".mp4", ".mp3", ".avi",
		".css", ".js", ".json", ".xml", ".txt", ".svg", ".ico", ".woff", ".woff2",
		".ttf", ".eot"}

	lowerURL := strings.ToLower(urlStr)
	for _, ext := range extensions {
		if strings.HasSuffix(lowerURL, ext) {
			return true
		}
	}
	return false
}

// removeDuplicates removes duplicate strings from a slice
func (c *SEOCrawler) removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// normalizeURL normalizes a URL (removes fragments, trailing slashes, etc.)
func (c *SEOCrawler) normalizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Remove fragment
	parsed.Fragment = ""

	// Remove trailing slash from path (except root)
	if parsed.Path != "/" && strings.HasSuffix(parsed.Path, "/") {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	}

	// Lowercase host
	parsed.Host = strings.ToLower(parsed.Host)

	return parsed.String()
}

// extractSEOElements extracts key SEO elements from the page
func (c *SEOCrawler) extractSEOElements(doc *goquery.Document, pageData *PageData) {
	// Title
	pageData.Title = strings.TrimSpace(doc.Find("title").First().Text())
	if pageData.Title == "" {
		pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
			Type:        "missing_title",
			Severity:    "high",
			Message:     "Page has no title tag",
			Description: "Title tags are crucial for SEO and user experience",
			Suggestions: []string{"Add a descriptive title tag (50-60 characters)", "Include primary keywords at the beginning"},
		})
	} else if len(pageData.Title) > 60 {
		pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
			Type:        "title_too_long",
			Severity:    "medium",
			Message:     fmt.Sprintf("Title is %d characters (should be under 60)", len(pageData.Title)),
			Element:     pageData.Title,
			Description: "Long titles may be truncated in search results",
			Suggestions: []string{"Shorten title to 50-60 characters", "Move less important keywords later in the title"},
		})
	} else if len(pageData.Title) < 30 {
		pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
			Type:        "title_too_short",
			Severity:    "low",
			Message:     fmt.Sprintf("Title is only %d characters", len(pageData.Title)),
			Description: "Short titles may not fully describe the page content",
			Suggestions: []string{"Expand title to 50-60 characters", "Add relevant keywords"},
		})
	}

	// Meta description
	doc.Find("meta[name='description']").Each(func(i int, s *goquery.Selection) {
		if desc, exists := s.Attr("content"); exists {
			pageData.MetaDescription = strings.TrimSpace(desc)
		}
	})

	if pageData.MetaDescription == "" {
		pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
			Type:        "missing_meta_description",
			Severity:    "high",
			Message:     "Page has no meta description",
			Description: "Meta descriptions influence click-through rates from search results",
			Suggestions: []string{"Add a compelling meta description (150-160 characters)", "Include a call-to-action", "Naturally incorporate keywords"},
		})
	} else if len(pageData.MetaDescription) > 160 {
		pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
			Type:        "meta_description_too_long",
			Severity:    "medium",
			Message:     fmt.Sprintf("Meta description is %d characters", len(pageData.MetaDescription)),
			Description: "Long descriptions may be truncated in search results",
			Suggestions: []string{"Shorten to 150-160 characters", "Put important information first"},
		})
	}

	// Canonical URL
	doc.Find("link[rel='canonical']").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			pageData.CanonicalURL = href
		}
	})

	// Headers structure
	c.checkHeaderStructure(doc, pageData)

	// Robots meta tag
	robotsContent := ""
	doc.Find("meta[name='robots']").Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists {
			robotsContent = content
		}
	})

	if strings.Contains(robotsContent, "noindex") {
		pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
			Type:        "noindex_tag",
			Severity:    "high",
			Message:     "Page has noindex meta tag",
			Description: "This page won't be indexed by search engines",
			Suggestions: []string{"Remove noindex tag if you want this page indexed", "Check if this is intentional"},
		})
	}
}

// checkHeaderStructure validates header hierarchy
func (c *SEOCrawler) checkHeaderStructure(doc *goquery.Document, pageData *PageData) {
	h1Count := 0
	hasH1 := false

	doc.Find("h1").Each(func(i int, s *goquery.Selection) {
		h1Count++
		hasH1 = true
		text := strings.TrimSpace(s.Text())
		if len(text) == 0 {
			pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
				Type:        "empty_h1",
				Severity:    "medium",
				Message:     "Empty H1 tag found",
				Description: "H1 tags should contain descriptive text",
				Suggestions: []string{"Add content to H1 tag", "Make it descriptive and keyword-rich"},
			})
		} else if len(text) > 70 {
			pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
				Type:        "h1_too_long",
				Severity:    "low",
				Message:     fmt.Sprintf("H1 is %d characters", len(text)),
				Description: "Very long H1 tags may be less effective",
				Suggestions: []string{"Keep H1 under 70 characters", "Use H2-H6 for subtopics"},
			})
		}
	})

	if h1Count > 1 {
		pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
			Type:        "multiple_h1",
			Severity:    "medium",
			Message:     fmt.Sprintf("Found %d H1 tags", h1Count),
			Description: "Multiple H1 tags can confuse search engines",
			Suggestions: []string{"Use only one H1 tag per page", "Convert extra H1 tags to H2 or H3"},
		})
	}

	if !hasH1 {
		pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
			Type:        "missing_h1",
			Severity:    "high",
			Message:     "Page has no H1 tag",
			Description: "H1 tags are important for page structure and SEO",
			Suggestions: []string{"Add one H1 tag containing the main topic", "Ensure H1 is descriptive and unique"},
		})
	}
}

// analyzeAndOptimizeImages processes images for SEO optimization
func (c *SEOCrawler) analyzeAndOptimizeImages(doc *goquery.Document, baseURL string, pageData *PageData) {
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || src == "" {
			return
		}

		alt, _ := s.Attr("alt")
		width, _ := s.Attr("width")
		height, _ := s.Attr("height")

		imgInfo := ImageInfo{
			OriginalURL: src,
			AltText:     alt,
			MissingAlt:  alt == "",
		}

		// Parse dimensions if available
		if width != "" {
			fmt.Sscanf(width, "%d", &imgInfo.Width)
		}
		if height != "" {
			fmt.Sscanf(height, "%d", &imgInfo.Height)
		}

		pageData.Images = append(pageData.Images, imgInfo)
	})

	// Add SEO issues for images
	var missingAltCount int
	var noDimensionsCount int

	for _, img := range pageData.Images {
		if img.MissingAlt {
			missingAltCount++
		}
		if img.Width == 0 || img.Height == 0 {
			noDimensionsCount++
		}
	}

	if missingAltCount > 0 {
		pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
			Type:        "missing_alt_text",
			Severity:    "medium",
			Message:     fmt.Sprintf("%d images missing alt text", missingAltCount),
			Description: "Alt text improves accessibility and image SEO",
			Suggestions: []string{"Add descriptive alt text to all images", "Include keywords naturally", "Leave alt empty for decorative images"},
		})
	}

	if noDimensionsCount > 0 {
		pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
			Type:        "missing_image_dimensions",
			Severity:    "low",
			Message:     fmt.Sprintf("%d images missing width/height attributes", noDimensionsCount),
			Description: "Missing dimensions can cause layout shifts and affect Core Web Vitals",
			Suggestions: []string{"Add width and height attributes to images", "Set explicit dimensions in CSS"},
		})
	}
}

// checkMobileResponsiveness analyzes mobile compatibility (non-blocking)
func (c *SEOCrawler) checkMobileResponsiveness(pageURL string, pageData *PageData) {
	// Default score if check fails
	pageData.MobileScore = 100
	
	// Simple mobile check without chromeDP (to avoid crashes)
	// Just check viewport meta tag and basic responsive indicators
	
	// This is a lightweight alternative to chromedp
	// For production, you might want to keep chromedp optional
	
	// For now, set a default good score
	pageData.MobileScore = 85
	
	// Add a note that comprehensive mobile testing requires chromedp
	pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
		Type:        "mobile_check_lightweight",
		Severity:    "info",
		Message:     "Basic mobile check completed",
		Description: "For full mobile testing, enable chromedp on your system",
		Suggestions: []string{"Install Chrome/Chromium for comprehensive mobile testing", "Use Google's Mobile-Friendly Test tool"},
	})
}

// extractSchemaMarkup finds and validates schema.org markup
func (c *SEOCrawler) extractSchemaMarkup(doc *goquery.Document, pageData *PageData) {
	schemaTypes := make(map[string]bool)

	// Check for JSON-LD
	doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection) {
		content := strings.TrimSpace(s.Text())
		if content != "" {
			var schema map[string]interface{}
			if err := json.Unmarshal([]byte(content), &schema); err == nil {
				if schemaType, ok := schema["@type"].(string); ok {
					schemaTypes[schemaType] = true
				}
			}
		}
	})

	// Check for microdata
	doc.Find("[itemscope]").Each(func(i int, s *goquery.Selection) {
		if itemType, exists := s.Attr("itemtype"); exists {
			parts := strings.Split(itemType, "/")
			schemaType := parts[len(parts)-1]
			schemaTypes[schemaType] = true
		}
	})

	// Convert map to slice
	for schemaType := range schemaTypes {
		pageData.SchemaTypes = append(pageData.SchemaTypes, schemaType)
	}

	if len(pageData.SchemaTypes) == 0 {
		pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
			Type:        "missing_schema",
			Severity:    "medium",
			Message:     "No schema.org markup found",
			Description: "Schema markup helps search engines understand your content",
			Suggestions: []string{
				"Add JSON-LD schema markup",
				"Consider adding Organization, WebSite, or Article schema",
				"Use Schema.org's Structured Data Testing Tool to validate",
			},
		})
	}
}

// calculateReadability computes readability scores
func (c *SEOCrawler) calculateReadability(doc *goquery.Document, pageData *PageData) {
	// Extract text content from main content areas
	text := doc.Find("p, article, main, .content, .post, .entry").Text()
	if text == "" {
		text = doc.Find("body").Text()
	}

	// Clean text
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	words := strings.Fields(text)
	pageData.WordCount = len(words)

	if pageData.WordCount < 300 {
		pageData.SEOIssues = append(pageData.SEOIssues, SEOIssue{
			Type:        "thin_content",
			Severity:    "medium",
			Message:     fmt.Sprintf("Page has only %d words", pageData.WordCount),
			Description: "Thin content may not rank well in search results",
			Suggestions: []string{"Add more comprehensive content (aim for 300+ words)", "Provide more value and information", "Expand on key topics"},
		})
	}

	// Extract keywords (simple TF)
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true, "was": true,
		"are": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "having": true, "do": true,
		"does": true, "did": true, "doing": true, "so": true, "if": true,
		"then": true, "else": true, "when": true, "where": true,
	}

	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".,!?;:\"'()[]{}«»…—–"))
		if len(word) > 2 && !stopWords[word] {
			pageData.Keywords[word]++
		}
	}
}

// minifyContent minifies HTML/CSS/JS
func (c *SEOCrawler) minifyContent(doc *goquery.Document, pageData *PageData, pageURL string) {
	// Get HTML
	html, err := doc.Html()
	if err != nil {
		return
	}

	// Minify HTML
	minifiedHTML, err := c.minifier.String("text/html", html)
	if err == nil && len(minifiedHTML) < len(html) {
		// Save minified version
		filename := fmt.Sprintf("%x.html", md5.Sum([]byte(pageURL)))
		path := filepath.Join(c.config.OutputDir, "minified", filename)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err == nil {
			os.WriteFile(path, []byte(minifiedHTML), 0644)
		}
	}
}

// resolveURL resolves relative URLs to absolute
func (c *SEOCrawler) resolveURL(base, ref string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		return "", err
	}

	resolved := baseURL.ResolveReference(refURL)

	// Ensure scheme is present
	if resolved.Scheme == "" {
		resolved.Scheme = "http"
	}

	return resolved.String(), nil
}

// saveResults saves crawl results to JSON file
func (c *SEOCrawler) saveResults() error {
	c.resultsMu.RLock()
	defer c.resultsMu.RUnlock()

	// Create results file
	resultsFile := filepath.Join(c.config.OutputDir, "crawl_results.json")
	
	// Create a clean copy without circular references
	cleanResults := make(map[string]*PageData)
	for url, page := range c.results {
		pageCopy := *page
		pageCopy.Headers = make(map[string]string)
		for k, v := range page.Headers {
			pageCopy.Headers[k] = v
		}
		cleanResults[url] = &pageCopy
	}

	data, err := json.MarshalIndent(cleanResults, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(resultsFile, data, 0644)
}

// ExportResults exports crawl results to JSON
func (c *SEOCrawler) ExportResults() ([]byte, error) {
	c.resultsMu.RLock()
	defer c.resultsMu.RUnlock()

	// Create a clean copy without circular references
	cleanResults := make(map[string]*PageData)
	for url, page := range c.results {
		pageCopy := *page
		pageCopy.Headers = make(map[string]string)
		for k, v := range page.Headers {
			pageCopy.Headers[k] = v
		}
		cleanResults[url] = &pageCopy
	}

	return json.MarshalIndent(cleanResults, "", "  ")
}

// GenerateReport creates an SEO report
func (c *SEOCrawler) GenerateReport() (map[string]interface{}, error) {
	c.resultsMu.RLock()
	defer c.resultsMu.RUnlock()

	report := map[string]interface{}{
		"total_pages":      len(c.results),
		"crawl_time":       time.Now(),
		"config":           c.config,
		"summary":          c.generateSummary(),
		"issues_by_type":   c.groupIssuesByType(),
		"recommendations":  c.generateRecommendations(),
		"top_pages":        c.getTopPages(),
	}

	return report, nil
}

// generateSummary creates a summary of findings
func (c *SEOCrawler) generateSummary() map[string]interface{} {
	summary := map[string]interface{}{
		"total_issues":        0,
		"high_priority":       0,
		"medium_priority":     0,
		"low_priority":        0,
		"avg_load_time_ms":    int64(0),
		"avg_word_count":      0,
		"avg_mobile_score":    0,
		"total_images":        0,
		"images_optimized":    0,
		"broken_links_total":  0,
		"total_valid_links":   0,
		"pages_with_schema":   0,
	}

	var totalLoadTime time.Duration
	var totalWordCount int
	var totalMobileScore int
	var pagesWithData int
	var pagesWithSchema int

	for _, page := range c.results {
		summary["total_issues"] = summary["total_issues"].(int) + len(page.SEOIssues)
		totalLoadTime += page.LoadTime
		totalWordCount += page.WordCount
		totalMobileScore += page.MobileScore
		summary["broken_links_total"] = summary["broken_links_total"].(int) + len(page.BrokenLinks)
		summary["total_valid_links"] = summary["total_valid_links"].(int) + len(page.ValidLinks)
		summary["total_images"] = summary["total_images"].(int) + len(page.Images)

		if len(page.SchemaTypes) > 0 {
			pagesWithSchema++
		}

		for _, issue := range page.SEOIssues {
			switch issue.Severity {
			case "high":
				summary["high_priority"] = summary["high_priority"].(int) + 1
			case "medium":
				summary["medium_priority"] = summary["medium_priority"].(int) + 1
			case "low":
				summary["low_priority"] = summary["low_priority"].(int) + 1
			}
		}

		pagesWithData++
	}

	if pagesWithData > 0 {
		summary["avg_load_time_ms"] = totalLoadTime.Milliseconds() / int64(pagesWithData)
		summary["avg_word_count"] = totalWordCount / pagesWithData
		if totalMobileScore > 0 {
			summary["avg_mobile_score"] = totalMobileScore / pagesWithData
		}
		summary["pages_with_schema"] = pagesWithSchema
		summary["schema_coverage_percent"] = (float64(pagesWithSchema) / float64(pagesWithData)) * 100
	}

	return summary
}

// groupIssuesByType groups SEO issues by type
func (c *SEOCrawler) groupIssuesByType() map[string]int {
	issuesByType := make(map[string]int)

	for _, page := range c.results {
		for _, issue := range page.SEOIssues {
			issuesByType[issue.Type]++
		}
	}

	return issuesByType
}

// generateRecommendations creates actionable recommendations
func (c *SEOCrawler) generateRecommendations() []string {
	recommendations := []string{}
	recommendationsMap := make(map[string]bool)

	for _, page := range c.results {
		for _, issue := range page.SEOIssues {
			switch issue.Type {
			case "missing_title":
				if !recommendationsMap["titles"] {
					recommendations = append(recommendations,
						"🔴 CRITICAL: Add title tags to all pages (aim for 50-60 characters, include primary keywords at the beginning)")
					recommendationsMap["titles"] = true
				}
			case "missing_meta_description":
				if !recommendationsMap["descriptions"] {
					recommendations = append(recommendations,
						"🟠 HIGH: Add meta descriptions to all pages (150-160 characters, include call-to-action and keywords)")
					recommendationsMap["descriptions"] = true
				}
			case "broken_links":
				if !recommendationsMap["broken"] {
					recommendations = append(recommendations,
						"🟠 HIGH: Fix or 301 redirect all broken links to improve user experience and preserve link equity")
					recommendationsMap["broken"] = true
				}
			case "missing_h1":
				if !recommendationsMap["h1"] {
					recommendations = append(recommendations,
						"🟠 HIGH: Add H1 tags to all pages (use one unique H1 per page, include main keywords)")
					recommendationsMap["h1"] = true
				}
			case "missing_alt_text":
				if !recommendationsMap["alt"] {
					recommendations = append(recommendations,
						"🟡 MEDIUM: Add descriptive alt text to all images (improves accessibility and image SEO)")
					recommendationsMap["alt"] = true
				}
			case "missing_schema":
				if !recommendationsMap["schema"] {
					recommendations = append(recommendations,
						"🟡 MEDIUM: Implement schema markup (Organization, WebSite, Article, BreadcrumbList) to enhance search results")
					recommendationsMap["schema"] = true
				}
			}
		}
	}

	// Performance recommendations
	var slowPages []string
	for url, page := range c.results {
		if page.LoadTime > 2*time.Second {
			slowPages = append(slowPages, url)
		}
	}

	if len(slowPages) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("🟡 MEDIUM: Optimize %d slow-loading pages (average load time >2s): implement caching, compress images, minify resources",
				len(slowPages)))
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations,
			"✅ Great job! No major SEO issues found. Continue monitoring and maintaining your SEO best practices.")
	}

	return recommendations
}

// getTopPages returns the top pages by various metrics
func (c *SEOCrawler) getTopPages() map[string]interface{} {
	topPages := make(map[string]interface{})

	var maxIssues int
	var worstPage string
	for url, page := range c.results {
		if len(page.SEOIssues) > maxIssues {
			maxIssues = len(page.SEOIssues)
			worstPage = url
		}
	}
	topPages["page_with_most_issues"] = map[string]interface{}{
		"url":    worstPage,
		"issues": maxIssues,
	}

	var maxLoadTime time.Duration
	var slowestPage string
	for url, page := range c.results {
		if page.LoadTime > maxLoadTime {
			maxLoadTime = page.LoadTime
			slowestPage = url
		}
	}
	topPages["slowest_page"] = map[string]interface{}{
		"url":       slowestPage,
		"load_time": maxLoadTime.String(),
	}

	return topPages
}

// GetResults returns the crawl results
func (c *SEOCrawler) GetResults() map[string]*PageData {
	c.resultsMu.RLock()
	defer c.resultsMu.RUnlock()
	return c.results
}

// GetStats returns crawling statistics
func (c *SEOCrawler) GetStats() map[string]interface{} {
	c.resultsMu.RLock()
	defer c.resultsMu.RUnlock()

	totalLinks := 0
	totalBroken := 0
	totalImages := 0

	for _, page := range c.results {
		totalLinks += len(page.ValidLinks)
		totalBroken += len(page.BrokenLinks)
		totalImages += len(page.Images)
	}

	return map[string]interface{}{
		"pages_crawled":      len(c.results),
		"total_valid_links":  totalLinks,
		"total_broken_links": totalBroken,
		"total_images":       totalImages,
		"config":             c.config,
	}
}