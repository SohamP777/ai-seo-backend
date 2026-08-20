package analyzer

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"
    "sync"
    "time"
)

// Client for Chrome UX Report API
type Client struct {
    httpClient *http.Client
    apiKey     string
    config     *VitalsConfig
    cache      map[string]*cachedResult
    cacheMu    sync.RWMutex
}

type cachedResult struct {
    data      *CoreWebVitals
    expiresAt time.Time
}

// NewClient creates a new CrUX API client
func NewClient(apiKey string, config *VitalsConfig) *Client {
    if config == nil {
        config = DefaultConfig()
    }
    
    return &Client{
        httpClient: &http.Client{
            Timeout: config.Timeout,
        },
        apiKey:  apiKey,
        config:  config,
        cache:   make(map[string]*cachedResult),
    }
}

// DefaultConfig returns default configuration
func DefaultConfig() *VitalsConfig {
    return &VitalsConfig{
        Timeout:         30 * time.Second,
        CacheTTL:        24 * time.Hour,
        EnableFieldData: true,
        EnableLabData:   false,
        FormFactor:      "ALL",
    }
}

// GetVitals fetches Core Web Vitals for a URL
func (c *Client) GetVitals(ctx context.Context, targetURL string) (*CoreWebVitals, error) {
    // Check cache first
    if cached := c.getFromCache(targetURL); cached != nil {
        return cached, nil
    }
    
    // Parse and normalize URL
    parsedURL, err := url.Parse(targetURL)
    if err != nil {
        return nil, fmt.Errorf("invalid URL: %w", err)
    }
    
    // Ensure URL has scheme
    if parsedURL.Scheme == "" {
        parsedURL.Scheme = "https"
    }
    
    normalizedURL := parsedURL.String()
    
    // Fetch data from CrUX API
    var mobileData, desktopData *CoreWebVitals
    var wg sync.WaitGroup
    var mobileErr, desktopErr error
    
    if c.config.FormFactor == "ALL" || c.config.FormFactor == "PHONE" {
        wg.Add(1)
        go func() {
            defer wg.Done()
            mobileData, mobileErr = c.fetchFormFactorData(ctx, normalizedURL, "PHONE")
        }()
    }
    
    if c.config.FormFactor == "ALL" || c.config.FormFactor == "DESKTOP" {
        wg.Add(1)
        go func() {
            defer wg.Done()
            desktopData, desktopErr = c.fetchFormFactorData(ctx, normalizedURL, "DESKTOP")
        }()
    }
    
    wg.Wait()
    
    if mobileErr != nil && desktopErr != nil {
        return nil, fmt.Errorf("failed to fetch data: mobile: %v, desktop: %v", mobileErr, desktopErr)
    }
    
    // Create main vitals object
    vitals := &CoreWebVitals{
        URL:         normalizedURL,
        RequestedAt: time.Now(),
        Mobile:      mobileData,
        Desktop:     desktopData,
    }
    
    // Set primary metrics (use mobile if available, otherwise desktop)
    if mobileData != nil {
        vitals.LCP = mobileData.LCP
        vitals.FID = mobileData.FID
        vitals.CLS = mobileData.CLS
        vitals.FCP = mobileData.FCP
        vitals.TTFB = mobileData.TTFB
    } else if desktopData != nil {
        vitals.LCP = desktopData.LCP
        vitals.FID = desktopData.FID
        vitals.CLS = desktopData.CLS
        vitals.FCP = desktopData.FCP
        vitals.TTFB = desktopData.TTFB
    }
    
    // Calculate overall category
    vitals.OverallCategory = c.calculateOverallCategory(vitals)
    
    // Generate recommendations
    vitals.Recommendations = c.generateRecommendations(vitals)
    
    // Generate specific issues
    vitals.Issues = c.identifyIssues(vitals)
    
    // Cache the result
    c.addToCache(normalizedURL, vitals)
    
    return vitals, nil
}

func (c *Client) fetchFormFactorData(ctx context.Context, urlStr, formFactor string) (*CoreWebVitals, error) {
    apiURL := fmt.Sprintf("https://chromeuxreport.googleapis.com/v1/records:queryRecord?key=%s", c.apiKey)
    
    payload := map[string]interface{}{
        "url":        urlStr,
        "formFactor": formFactor,
    }
    
    jsonData, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(jsonData)))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("API request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API returned status: %s", resp.Status)
    }
    
    var cruxResp ChromeUXResponse
    if err := json.NewDecoder(resp.Body).Decode(&cruxResp); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }
    
    // Create CoreWebVitals object for this form factor
    vitals := &CoreWebVitals{
        URL:         urlStr,
        RequestedAt: time.Now(),
    }
    
    // Parse metrics
    for name, metric := range cruxResp.Record.Metrics {
        // Calculate category percentages from histogram
        var good, needsImprovement, poor float64
        
        for _, h := range metric.Histogram {
            switch name {
            case "LARGEST_CONTENTFUL_PAINT_MS":
                // Good: <2500ms, Needs Improvement: 2500-4000ms, Poor: >4000ms
                if h.End <= 2500 {
                    good += h.Density
                } else if h.Start < 4000 {
                    needsImprovement += h.Density
                } else {
                    poor += h.Density
                }
            case "FIRST_INPUT_DELAY_MS":
                // Good: <100ms, Needs Improvement: 100-300ms, Poor: >300ms
                if h.End <= 100 {
                    good += h.Density
                } else if h.Start < 300 {
                    needsImprovement += h.Density
                } else {
                    poor += h.Density
                }
            case "CUMULATIVE_LAYOUT_SHIFT_SCORE":
                // Good: <0.1, Needs Improvement: 0.1-0.25, Poor: >0.25
                if h.End <= 0.1 {
                    good += h.Density
                } else if h.Start < 0.25 {
                    needsImprovement += h.Density
                } else {
                    poor += h.Density
                }
            case "FIRST_CONTENTFUL_PAINT_MS":
                if h.End <= 1800 {
                    good += h.Density
                } else if h.Start < 3000 {
                    needsImprovement += h.Density
                } else {
                    poor += h.Density
                }
            case "TIME_TO_FIRST_BYTE_MS":
                if h.End <= 800 {
                    good += h.Density
                } else if h.Start < 1800 {
                    needsImprovement += h.Density
                } else {
                    poor += h.Density
                }
            }
        }
        
        m := &Metric{
            Percentile:       metric.Percentiles.P75,
            Good:             good * 100,
            NeedsImprovement: needsImprovement * 100,
            Poor:             poor * 100,
        }
        
        // Set category based on p75
        switch name {
        case "LARGEST_CONTENTFUL_PAINT_MS":
            if m.Percentile < 2500 {
                m.Category = "Good"
            } else if m.Percentile < 4000 {
                m.Category = "Needs Improvement"
            } else {
                m.Category = "Poor"
            }
            m.Unit = "ms"
            vitals.LCP = m
            
        case "FIRST_INPUT_DELAY_MS":
            if m.Percentile < 100 {
                m.Category = "Good"
            } else if m.Percentile < 300 {
                m.Category = "Needs Improvement"
            } else {
                m.Category = "Poor"
            }
            m.Unit = "ms"
            vitals.FID = m
            
        case "CUMULATIVE_LAYOUT_SHIFT_SCORE":
            if m.Percentile < 0.1 {
                m.Category = "Good"
            } else if m.Percentile < 0.25 {
                m.Category = "Needs Improvement"
            } else {
                m.Category = "Poor"
            }
            m.Unit = "score"
            vitals.CLS = m
            
        case "FIRST_CONTENTFUL_PAINT_MS":
            if m.Percentile < 1800 {
                m.Category = "Good"
            } else if m.Percentile < 3000 {
                m.Category = "Needs Improvement"
            } else {
                m.Category = "Poor"
            }
            m.Unit = "ms"
            vitals.FCP = m
            
        case "TIME_TO_FIRST_BYTE_MS":
            if m.Percentile < 800 {
                m.Category = "Good"
            } else if m.Percentile < 1800 {
                m.Category = "Needs Improvement"
            } else {
                m.Category = "Poor"
            }
            m.Unit = "ms"
            vitals.TTFB = m
        }
    }
    
    return vitals, nil
}

func (c *Client) calculateOverallCategory(vitals *CoreWebVitals) string {
    if vitals.LCP == nil || vitals.FID == nil || vitals.CLS == nil {
        return "Insufficient Data"
    }
    
    categories := []string{vitals.LCP.Category, vitals.FID.Category, vitals.CLS.Category}
    
    // Count occurrences
    counts := map[string]int{
        "Good":              0,
        "Needs Improvement": 0,
        "Poor":              0,
    }
    
    for _, cat := range categories {
        counts[cat]++
    }
    
    // Determine overall
    if counts["Poor"] > 0 {
        return "Poor"
    }
    if counts["Needs Improvement"] > 0 {
        return "Needs Improvement"
    }
    if counts["Good"] == 3 {
        return "Good"
    }
    
    return "Mixed"
}

func (c *Client) generateRecommendations(vitals *CoreWebVitals) []Recommendation {
    var recommendations []Recommendation
    
    // LCP Recommendations
    if vitals.LCP != nil && vitals.LCP.Category != "Good" {
        recommendations = append(recommendations, c.getLCPRecommendations(vitals)...)
    }
    
    // FID Recommendations
    if vitals.FID != nil && vitals.FID.Category != "Good" {
        recommendations = append(recommendations, c.getFIDRecommendations(vitals)...)
    }
    
    // CLS Recommendations
    if vitals.CLS != nil && vitals.CLS.Category != "Good" {
        recommendations = append(recommendations, c.getCLSRecommendations(vitals)...)
    }
    
    return recommendations
}

func (c *Client) getLCPRecommendations(vitals *CoreWebVitals) []Recommendation {
    var recs []Recommendation
    
    // Check TTFB if available
    if vitals.TTFB != nil && vitals.TTFB.Percentile > 600 {
        recs = append(recs, Recommendation{
            Vital:    "lcp",
            Priority: "high",
            Title:    "Improve Server Response Time (TTFB)",
            Description: "Your Time to First Byte is slow. This indicates server-side performance issues.",
            Impact:   "Could reduce LCP by 30-50%",
            CodeSnippet: `# Enable caching
# Nginx example
add_header Cache-Control "public, max-age=31536000";

# Use a CDN
# Optimize database queries
# Implement Redis/Memcached caching`,
        })
    }
    
    // General LCP recommendations
    recs = append(recs, Recommendation{
        Vital:    "lcp",
        Priority: "high",
        Title:    "Optimize Largest Contentful Paint Element",
        Description: "The LCP element is likely an image or text block. Optimize its loading.",
        Impact:   "Can reduce LCP by 20-40%",
        CodeSnippet: `<!-- For images: add width/height and loading="lazy" for below-fold -->
<img src="hero.jpg" width="1200" height="800" loading="eager" fetchpriority="high">

<!-- Preload critical image -->
<link rel="preload" as="image" href="hero.jpg">

<!-- Optimize images: use WebP format -->
<picture>
  <source srcset="hero.webp" type="image/webp">
  <img src="hero.jpg" width="1200" height="800">
</picture>`,
    })
    
    recs = append(recs, Recommendation{
        Vital:    "lcp",
        Priority: "medium",
        Title:    "Eliminate Render-Blocking Resources",
        Description: "Remove or defer CSS/JS that blocks rendering.",
        Impact:   "Can reduce LCP by 15-25%",
        CodeSnippet: `<!-- Defer non-critical CSS -->
<link rel="preload" href="styles.css" as="style" onload="this.onload=null;this.rel='stylesheet'">
<noscript><link rel="stylesheet" href="styles.css"></noscript>

<!-- Defer JavaScript -->
<script defer src="non-critical.js"></script>
<script async src="analytics.js"></script>`,
    })
    
    return recs
}

func (c *Client) getFIDRecommendations(vitals *CoreWebVitals) []Recommendation {
    return []Recommendation{
        {
            Vital:    "fid",
            Priority: "high",
            Title:    "Reduce JavaScript Execution Time",
            Description: "Long JavaScript tasks block the main thread and increase FID.",
            Impact:   "Can reduce FID by 50-70%",
            CodeSnippet: `// Break up long tasks
// Instead of:
// processLargeArray(items);

// Do this:
function processInChunks(items, chunkSize = 100) {
  let index = 0;
  
  function processChunk() {
    const chunk = items.slice(index, index + chunkSize);
    processChunkSync(chunk);
    index += chunkSize;
    
    if (index < items.length) {
      setTimeout(processChunk, 0); // Yield to main thread
    }
  }
  
  processChunk();
}`,
        },
        {
            Vital:    "fid",
            Priority: "high",
            Title:    "Optimize Third-Party Scripts",
            Description: "Third-party scripts can block interactivity. Load them asynchronously.",
            Impact:   "Can reduce FID by 30-50%",
            CodeSnippet: `<!-- Load third-party scripts asynchronously -->
<script async src="https://third-party.com/widget.js"></script>

<!-- Or defer them -->
<script defer src="https://analytics.com/script.js"></script>

<!-- Use resource hints -->
<link rel="preconnect" href="https://third-party.com">
<link rel="dns-prefetch" href="https://analytics.com">`,
        },
        {
            Vital:    "fid",
            Priority: "medium",
            Title:    "Reduce DOM Size",
            Description: "Large DOM trees increase processing time and memory usage.",
            Impact:   "Can reduce FID by 10-20%",
            CodeSnippet: `<!-- Avoid deeply nested structures -->
<!-- Instead of: -->
<div><div><div><div><span>Text</span></div></div></div></div>

<!-- Do: -->
<span>Text</span>

<!-- Use semantic HTML -->
<!-- Limit DOM nodes to < 1500 -->
<!-- Limit depth to < 32 levels -->
<!-- Limit children/parent to < 60 -->`,
        },
    }
}

func (c *Client) getCLSRecommendations(vitals *CoreWebVitals) []Recommendation {
    return []Recommendation{
        {
            Vital:    "cls",
            Priority: "high",
            Title:    "Set Width and Height on Images",
            Description: "Images without dimensions cause layout shifts as they load.",
            Impact:   "Can reduce CLS by 50-80%",
            CodeSnippet: `<!-- Always include width and height attributes -->
<img src="image.jpg" width="800" height="600" alt="description">

<!-- For responsive images -->
<img src="image.jpg" 
     srcset="image-400.jpg 400w, image-800.jpg 800w" 
     sizes="(max-width: 600px) 400px, 800px"
     width="800" 
     height="600"
     alt="description">

<!-- CSS aspect ratio boxes for background images -->
.aspect-ratio-box {
  aspect-ratio: 16 / 9;
  width: 100%;
  background: url('image.jpg') center/cover;
}`,
        },
        {
            Vital:    "cls",
            Priority: "high",
            Title:    "Reserve Space for Dynamic Content",
            Description: "Ads, embeds, and dynamic content should have reserved space.",
            Impact:   "Can reduce CLS by 40-60%",
            CodeSnippet: `<!-- Reserve space for ads -->
<div class="ad-container" style="min-height: 250px; width: 300px;">
  <!-- Ad code here -->
</div>

<!-- For dynamic embeds -->
<div class="embed-container" style="aspect-ratio: 16/9;">
  <iframe src="video.html" width="100%" height="100%"></iframe>
</div>

<!-- Use CSS for loading states -->
.loading-skeleton {
  background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: loading 1.5s infinite;
}`,
        },
        {
            Vital:    "cls",
            Priority: "medium",
            Title:    "Optimize Web Fonts",
            Description: "Font loading can cause layout shifts (FOIT/FOUT).",
            Impact:   "Can reduce CLS by 20-30%",
            CodeSnippet: `/* Use font-display: swap in @font-face */
@font-face {
  font-family: 'Custom Font';
  src: url('/fonts/custom.woff2') format('woff2');
  font-display: swap;
}

/* Preload critical fonts */
<link rel="preload" href="/fonts/custom.woff2" as="font" type="font/woff2" crossorigin>

/* Use system fonts initially */
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

/* Apply custom font after load */
.fonts-loaded body {
  font-family: 'Custom Font', sans-serif;
}`,
        },
    }
}

func (c *Client) identifyIssues(vitals *CoreWebVitals) []VitalIssue {
    var issues []VitalIssue
    
    if vitals.LCP != nil && vitals.LCP.Category != "Good" {
        issues = append(issues, VitalIssue{
            Vital:       "lcp",
            Value:       vitals.LCP.Percentile,
            Threshold:   2500,
            Description: "Largest Contentful Paint is slow",
            Fix:         "Optimize LCP element loading, improve server response, and eliminate render-blocking resources",
        })
    }
    
    if vitals.FID != nil && vitals.FID.Category != "Good" {
        issues = append(issues, VitalIssue{
            Vital:       "fid",
            Value:       vitals.FID.Percentile,
            Threshold:   100,
            Description: "First Input Delay is high",
            Fix:         "Break up long tasks, optimize third-party scripts, reduce main thread work",
        })
    }
    
    if vitals.CLS != nil && vitals.CLS.Category != "Good" {
        issues = append(issues, VitalIssue{
            Vital:       "cls",
            Value:       vitals.CLS.Percentile,
            Threshold:   0.1,
            Description: "Cumulative Layout Shift is too high",
            Fix:         "Add dimensions to images, reserve space for dynamic content, optimize fonts",
        })
    }
    
    if vitals.TTFB != nil && vitals.TTFB.Percentile > 600 {
        issues = append(issues, VitalIssue{
            Vital:       "ttfb",
            Value:       vitals.TTFB.Percentile,
            Threshold:   600,
            Description: "Time to First Byte is slow",
            Fix:         "Implement caching, use CDN, optimize server-side code and database queries",
        })
    }
    
    return issues
}

func (c *Client) getFromCache(url string) *CoreWebVitals {
    c.cacheMu.RLock()
    defer c.cacheMu.RUnlock()
    
    cached, exists := c.cache[url]
    if !exists || time.Now().After(cached.expiresAt) {
        return nil
    }
    
    return cached.data
}

func (c *Client) addToCache(url string, data *CoreWebVitals) {
    c.cacheMu.Lock()
    defer c.cacheMu.Unlock()
    
    c.cache[url] = &cachedResult{
        data:      data,
        expiresAt: time.Now().Add(c.config.CacheTTL),
    }
}