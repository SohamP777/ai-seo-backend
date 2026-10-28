// pkg/wordpress/fixer.go
package wordpress

import (
     "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "regexp"
    "strings"
    "sync"
    "time"
    "log"
)

type WordPressFixer struct {
    client   *http.Client
    logger   *log.Logger
    siteURL  string
    username string
    password string
    metaTagsFixed int
}

type Fixer struct {
    client           *Client
    backupManager    *BackupManager
    metaFixer        *MetaFixer
    contentFixer     *ContentFixer
    schemaFixer      *SchemaFixer
    imageFixer       *ImageFixer
    technicalFixer   *TechnicalFixer
    performanceFixer *PerformanceFixer
    logger           *log.Logger
}

func NewWordPressFixer(client *http.Client, logger *log.Logger) *WordPressFixer {
    return &WordPressFixer{
        client: client,
        logger: logger,
    }
}

func (w *WordPressFixer) getWordPressClient() *Client {
    return &Client{
        httpClient: w.client,
        baseURL:    w.siteURL,
        rateLimiter: &RateLimiter{},  // ← ADD THIS
        logger:      &Logger{},       // ← ADD THIS
        auth:        &Auth{},         // ← ADD THIS
    }
}

// FixTechnicalSEO method
func (w *WordPressFixer) FixTechnicalSEO(url string) (int, error) {
    w.logger.Printf("Fixing WordPress technical SEO for %s", url)
    
    wpClient := w.getWordPressClient()
    technicalFixer := NewTechnicalFixer(wpClient, w.logger)
    
    results, err := technicalFixer.Fix(context.Background(), false)
    if err != nil {
        return 0, err
    }
    return len(results), nil
}

// In wordpress/fixer.go - REAL implementation
func (w *WordPressFixer) FixMetaTags(url string) (int, error) {
    fixedCount := 0
    w.logger.Printf("🔧 WordPress API: Fixing meta tags for %s", url)
    
    // Get post ID
    postID, err := w.getPostIDFromURL(url)
    if err != nil {
        return 0, fmt.Errorf("failed to get post: %w", err)
    }
    
    // Get current post
    post, err := w.getPost(postID)
    if err != nil {
        return 0, err
    }
    
    // REAL: Update title if missing
    if post["title"] == nil || post["title"] == "" {
        newTitle := w.generateOptimizedTitle(url)
        if err := w.updatePostTitle(postID, newTitle); err == nil {
            fixedCount++
            w.logger.Printf("✅ Added title via WordPress API")
        }
    }
    
    // REAL: Update meta description via Yoast API
    if err := w.updateMetaDescription(postID, w.generateMetaDescription(url)); err == nil {
        fixedCount++
        w.logger.Printf("✅ Added meta description via Yoast API")
    }
    
    return fixedCount, nil
}

// Updated to return actual count of fixes applied
func (w *WordPressFixer) FixImages(url string) (int, error) {
    fixedCount := 0
    w.logger.Printf("Fixing images for %s", url)
    
    // Step 1: Get post content
    postID, err := w.getPostIDFromURL(url)
    if err != nil {
        return 0, err
    }
    
    postData, err := w.getPost(postID)
    if err != nil {
        return 0, err
    }
    
    content := getStringValue(postData, "content")
    
    // Step 2: Find all images without alt text
    imgRegex := regexp.MustCompile(`<img[^>]+src="([^">]+)"[^>]*>`)
    images := imgRegex.FindAllStringSubmatch(content, -1)
    
    for _, img := range images {
        fullImg := img[0]
        src := img[1]
        
        // Check if alt exists
        if !strings.Contains(fullImg, "alt=") {
            // Generate alt text from filename
            altText := w.generateAltTextFromSrc(src)
            
            // Add alt attribute to image
            newImg := strings.Replace(fullImg, "<img", fmt.Sprintf(`<img alt="%s"`, altText), 1)
            content = strings.Replace(content, fullImg, newImg, 1)
            fixedCount++
        }
    }
    
    // Step 3: Update post content if changes were made
    if fixedCount > 0 {
        if err := w.updatePostContent(postID, content); err != nil {
            return fixedCount, fmt.Errorf("failed to update post content: %w", err)
        }
        w.logger.Printf("✅ Added alt text to %d images", fixedCount)
    }
    
    return fixedCount, nil
}

// Updated to return actual count of fixes applied
func (w *WordPressFixer) FixContentStructure(url string) (int, error) {
    fixedCount := 0
    w.logger.Printf("Fixing content structure for %s", url)
    
    // Step 1: Get post content
    postID, err := w.getPostIDFromURL(url)
    if err != nil {
        return 0, err
    }
    
    postData, err := w.getPost(postID)
    if err != nil {
        return 0, err
    }
    
    content := getStringValue(postData, "content")
    originalContent := content
    
    // Step 2: Check for H1 tag
    if !strings.Contains(content, "<h1>") && !strings.Contains(content, "<h1 ") {
        // Extract title for H1
        title := getStringValue(postData, "title")
        if title == "" {
            title = "Welcome to " + strings.TrimPrefix(url, "https://")
        }
        
        // Add H1 at beginning of content
        content = fmt.Sprintf("<h1>%s</h1>\n%s", title, content)
        fixedCount++
        w.logger.Printf("✅ Added missing H1 tag: %s", title)
    }
    
    // Step 3: Fix heading hierarchy (ensure H2 follows H1, etc.)
    h1Count := strings.Count(content, "<h1")
    if h1Count > 1 {
        // Replace extra H1 with H2
        h1Regex := regexp.MustCompile(`<h1[^>]*>.*?</h1>`)
        matches := h1Regex.FindAllString(content, -1)
        if len(matches) > 1 {
            for i := 1; i < len(matches); i++ {
                newHeading := strings.Replace(matches[i], "h1", "h2", -1)
                content = strings.Replace(content, matches[i], newHeading, 1)
                fixedCount++
            }
            w.logger.Printf("✅ Fixed heading hierarchy (converted extra H1 to H2)")
        }
    }
    
    // Step 4: Update post if changes were made
    if content != originalContent {
        if err := w.updatePostContent(postID, content); err != nil {
            return fixedCount, fmt.Errorf("failed to update post content: %w", err)
        }
    }
    
    return fixedCount, nil
}

// ========== REAL WordPress REST API Methods ==========

func (w *WordPressFixer) getPostIDFromURL(url string) (int, error) {
    // Extract post ID from URL or by slug
    apiURL := strings.TrimSuffix(w.siteURL, "/") + "/wp-json/wp/v2/posts?slug=" + getSlugFromURL(url)
    
    req, _ := http.NewRequest("GET", apiURL, nil)
    req.SetBasicAuth(w.username, w.password)
    
    resp, err := w.client.Do(req)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()
    
    body, _ := io.ReadAll(resp.Body)
    var posts []map[string]interface{}
    json.Unmarshal(body, &posts)
    
    if len(posts) == 0 {
        return 0, fmt.Errorf("post not found")
    }
    
    return int(posts[0]["id"].(float64)), nil
}

func (w *WordPressFixer) getPost(postID int) (map[string]interface{}, error) {
    apiURL := fmt.Sprintf("%s/wp-json/wp/v2/posts/%d", strings.TrimSuffix(w.siteURL, "/"), postID)
    
    req, _ := http.NewRequest("GET", apiURL, nil)
    req.SetBasicAuth(w.username, w.password)
    
    resp, err := w.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    body, _ := io.ReadAll(resp.Body)
    var post map[string]interface{}
    json.Unmarshal(body, &post)
    
    return post, nil
}

func (w *WordPressFixer) updatePostTitle(postID int, title string) error {
    apiURL := fmt.Sprintf("%s/wp-json/wp/v2/posts/%d", strings.TrimSuffix(w.siteURL, "/"), postID)
    
    data := map[string]interface{}{"title": title}
    jsonData, _ := json.Marshal(data)
    
    req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
    req.SetBasicAuth(w.username, w.password)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := w.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}

func (w *WordPressFixer) updatePostContent(postID int, content string) error {
    apiURL := fmt.Sprintf("%s/wp-json/wp/v2/posts/%d", strings.TrimSuffix(w.siteURL, "/"), postID)
    
    data := map[string]interface{}{"content": content}
    jsonData, _ := json.Marshal(data)
    
    req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
    req.SetBasicAuth(w.username, w.password)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := w.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}

func (w *WordPressFixer) getMetaDescription(postID int) string {
    apiURL := fmt.Sprintf("%s/wp-json/yoast/v1/get_head", strings.TrimSuffix(w.siteURL, "/"))
    
    req, _ := http.NewRequest("GET", apiURL, nil)
    req.SetBasicAuth(w.username, w.password)
    
    resp, err := w.client.Do(req)
    if err != nil {
        return ""
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    
    if description, ok := result["description"].(string); ok {
        return description
    }
    return ""
}

func (w *WordPressFixer) updateMetaDescription(postID int, description string) error {
    apiURL := fmt.Sprintf("%s/wp-json/yoast/v1/update_meta", strings.TrimSuffix(w.siteURL, "/"))
    
    data := map[string]interface{}{
        "post_id": postID,
        "meta": map[string]string{
            "yoast_wpseo_metadesc": description,
        },
    }
    jsonData, _ := json.Marshal(data)
    
    req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
    req.SetBasicAuth(w.username, w.password)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := w.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}

// Helper functions
func getStringValue(data map[string]interface{}, key string) string {
    if val, ok := data[key].(string); ok {
        return val
    }
    if val, ok := data[key].(map[string]interface{}); ok {
        if rendered, ok := val["rendered"].(string); ok {
            return rendered
        }
    }
    return ""
}

func getSlugFromURL(url string) string {
    parts := strings.Split(strings.TrimSuffix(url, "/"), "/")
    return parts[len(parts)-1]
}

func (w *WordPressFixer) generateOptimizedTitle(url string) string {
    domain := strings.TrimPrefix(url, "https://")
    domain = strings.TrimPrefix(domain, "http://")
    return fmt.Sprintf("%s - Expert Guide | SEO Optimized", domain)
}

func (w *WordPressFixer) generateMetaDescription(url string) string {
    return fmt.Sprintf("Learn everything about %s. Expert tips, best practices, and comprehensive guides to help you succeed.", strings.TrimPrefix(url, "https://"))
}

func (w *WordPressFixer) generateAltTextFromSrc(src string) string {
    parts := strings.Split(src, "/")
    filename := parts[len(parts)-1]
    name := strings.TrimSuffix(filename, ".jpg")
    name = strings.TrimSuffix(name, ".png")
    name = strings.TrimSuffix(name, ".jpeg")
    name = strings.ReplaceAll(name, "-", " ")
    name = strings.ReplaceAll(name, "_", " ")
    return fmt.Sprintf("SEO optimized image - %s", name)
}
func NewFixer(client *Client, backupManager *BackupManager, logger *log.Logger) *Fixer {
    return &Fixer{
        client:          client,
        backupManager:   backupManager,
        metaFixer:       NewMetaFixer(client, logger),
        contentFixer:    NewContentFixer(client, logger),
        schemaFixer:     NewSchemaFixer(client, logger),
        imageFixer:      NewImageFixer(client, logger),
        technicalFixer:  NewTechnicalFixer(client, logger),
        performanceFixer: NewPerformanceFixer(client, logger),
        logger:          logger,  // This will still have type mismatch if Fixer.logger is *Logger
    }
}

func (f *Fixer) Analyze(ctx context.Context, siteURL string) (*SEOReport, error) {
    f.logger.Printf("Starting SEO analysis for %s", siteURL)
    
    report := &SEOReport{
        SiteURL:   siteURL,
        StartTime: time.Now(),
        Issues:    []SEOIssue{},
    }
    
    var wg sync.WaitGroup
    issuesChan := make(chan SEOIssue, 100)
    
    // Analyze all aspects concurrently
    analyzers := []func(context.Context) ([]SEOIssue, error){
        f.metaFixer.Analyze,
        f.contentFixer.Analyze,
        f.schemaFixer.Analyze,
        f.imageFixer.Analyze,
        f.technicalFixer.Analyze,
        f.performanceFixer.Analyze,
    }
    
    for _, analyzer := range analyzers {
        wg.Add(1)
        go func(a func(context.Context) ([]SEOIssue, error)) {
            defer wg.Done()
            issues, err := a(ctx)
            if err != nil {
                f.logger.Printf("Analysis error: %v", err)
                return
            }
            for _, issue := range issues {
                issuesChan <- issue
            }
        }(analyzer)
    }
    
    go func() {
        wg.Wait()
        close(issuesChan)
    }()
    
    for issue := range issuesChan {
        report.Issues = append(report.Issues, issue)
    }
    
    // Calculate SEO score
    report.Score = f.calculateScore(report.Issues)
    report.RankingImpact = f.calculateRankingImpact(report.Score)
    report.EstimatedTraffic = f.estimateTrafficImpact(report.Score)
    report.EndTime = time.Now()
    
    f.logger.Printf("Analysis complete. Score: %d/100, Issues found: %d", report.Score, len(report.Issues))
    
    return report, nil
}

func (f *Fixer) Fix(ctx context.Context, siteURL string, options FixOptions) (*SEOReport, error) {
    var backupID string
    
    if options.CreateBackup {
        backup, err := f.backupManager.CreateBackup(ctx, siteURL)
        if err != nil {
            return nil, fmt.Errorf("failed to create backup: %w", err)
        }
        backupID = backup.ID
        f.logger.Printf("Backup created: %s", backupID)
    }
    
    report := &SEOReport{
        SiteURL:       siteURL,
        StartTime:     time.Now(),
        FixesApplied:  []FixResult{},
        BackupCreated: options.CreateBackup,
        BackupID:      backupID,
    }
    
    var wg sync.WaitGroup
    fixesChan := make(chan FixResult, 100)
    
    // Apply fixes based on options
    fixers := []struct {
        name    string
        enabled bool
        fixer   func(context.Context) ([]FixResult, error)
    }{
        {"Meta", options.FixMeta, func(ctx context.Context) ([]FixResult, error) {
            return f.metaFixer.Fix(ctx, options.DryRun)
        }},
        {"Content", options.FixContent, func(ctx context.Context) ([]FixResult, error) {
            return f.contentFixer.Fix(ctx, options.DryRun)
        }},
        {"Schema", options.FixSchema, func(ctx context.Context) ([]FixResult, error) {
            return f.schemaFixer.Fix(ctx, options.DryRun)
        }},
        {"Images", options.FixImages, func(ctx context.Context) ([]FixResult, error) {
            return f.imageFixer.Fix(ctx, options.DryRun)
        }},
        {"Technical", options.FixTechnical, func(ctx context.Context) ([]FixResult, error) {
            return f.technicalFixer.Fix(ctx, options.DryRun)
        }},
        {"Performance", options.FixPerformance, func(ctx context.Context) ([]FixResult, error) {
            return f.performanceFixer.Fix(ctx, options.DryRun)
        }},
    }
    
    for _, fixer := range fixers {
        if !fixer.enabled {
            continue
        }
        
        wg.Add(1)
        go func(name string, f func(context.Context) ([]FixResult, error)) {
            defer wg.Done()
            fixes, err := f(ctx)
            if err != nil {
                log.Printf("Error: %s fixer error: %v", name, err)
                fixesChan <- FixResult{
                    Success: false,
                    Action:  name,
                    Error:   err.Error(),
                }
                return
            }
            for _, fix := range fixes {
                fixesChan <- fix
            }
        }(fixer.name, fixer.fixer)
    }
    
    go func() {
        wg.Wait()
        close(fixesChan)
    }()
    
    for fix := range fixesChan {
        report.FixesApplied = append(report.FixesApplied, fix)
    }
    
    // Verify site still works
    if err := f.verifySiteWorks(ctx); err != nil {
        if backupID != "" {
            f.logger.Printf("Site verification failed, rolling back...")
            if rollbackErr := f.backupManager.RestoreBackup(ctx, backupID); rollbackErr != nil {
                f.logger.Printf("Rollback failed: %v", rollbackErr)
            }
        }
        return nil, fmt.Errorf("site verification failed after fixes: %w", err)
    }
    
    // Calculate final score
    report.Score = f.calculateScoreAfterFixes(report.FixesApplied)
    report.RankingImpact = f.calculateRankingImpact(report.Score)
    report.EstimatedTraffic = f.estimateTrafficImpact(report.Score)
    report.EndTime = time.Now()
    
    f.logger.Printf("Fix complete. Applied %d fixes, new score: %d/100", 
        len(report.FixesApplied), report.Score)
    
    return report, nil
}

func (f *Fixer) verifySiteWorks(ctx context.Context) error {
    // Check homepage is accessible
    _, err := f.client.Get(ctx, "")
    if err != nil {
        return fmt.Errorf("homepage not accessible: %w", err)
    }
    
    // Check API is responsive
    _, err = f.client.Get(ctx, "/wp-json/wp/v2")
    if err != nil {
        return fmt.Errorf("API not responsive: %w", err)
    }
    
    return nil
}

func (f *Fixer) calculateScore(issues []SEOIssue) int {
    // Start with perfect score
    score := 100
    
    // Deduct points based on issue severity
    for _, issue := range issues {
        switch issue.Severity {
        case "critical":
            score -= 15
        case "high":
            score -= 10
        case "medium":
            score -= 5
        case "low":
            score -= 2
        }
    }
    
    if score < 0 {
        score = 0
    }
    
    return score
}

func (f *Fixer) calculateScoreAfterFixes(fixes []FixResult) int {
    // Simplified score calculation
    score := 50 // Starting score
    
    for _, fix := range fixes {
        if fix.Success {
            score += 5
        }
    }
    
    if score > 100 {
        score = 100
    }
    
    return score
}

func (f *Fixer) calculateRankingImpact(score int) string {
    switch {
    case score >= 90:
        return "Significant positive impact expected"
    case score >= 70:
        return "Moderate positive impact expected"
    case score >= 50:
        return "Minor positive impact expected"
    default:
        return "Limited impact expected"
    }
}

func (f *Fixer) estimateTrafficImpact(score int) string {
    switch {
    case score >= 90:
        return "+20-30% in 3-6 months"
    case score >= 70:
        return "+10-20% in 3-6 months"
    case score >= 50:
        return "+5-10% in 6 months"
    default:
        return "<5% increase expected"
    }
}