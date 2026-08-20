package analyzer

import "time"

// Core Web Vitals data from Chrome UX Report
type CoreWebVitals struct {
    URL              string         `json:"url"`
    RequestedAt      time.Time      `json:"requested_at"`
    LCP              *Metric        `json:"lcp,omitempty"`               // Largest Contentful Paint
    FID              *Metric        `json:"fid,omitempty"`               // First Input Delay
    CLS              *Metric        `json:"cls,omitempty"`               // Cumulative Layout Shift
    FCP              *Metric        `json:"fcp,omitempty"`               // First Contentful Paint
    TTFB             *Metric        `json:"ttfb,omitempty"`              // Time to First Byte
    INP              *Metric        `json:"inp,omitempty"`               // Interaction to Next Paint (upcoming)
    OverallCategory  string         `json:"overall_category"`             // Good, Needs Improvement, Poor
    Mobile           *CoreWebVitals `json:"mobile,omitempty"`             // Mobile-specific data
    Desktop          *CoreWebVitals `json:"desktop,omitempty"`            // Desktop-specific data
    Recommendations  []Recommendation `json:"recommendations,omitempty"`
    Issues           []VitalIssue   `json:"issues,omitempty"`
}

type Metric struct {
    Percentile       float64 `json:"percentile"`         // 75th percentile value
    Category         string  `json:"category"`           // Good, Needs Improvement, Poor
    Unit             string  `json:"unit"`                // "ms", "score"
    Good             float64 `json:"good_percentage"`     // % of good experiences
    NeedsImprovement float64 `json:"needs_improvement_percentage"` // % of needs improvement
    Poor             float64 `json:"poor_percentage"`     // % of poor experiences
}

type Recommendation struct {
    Vital        string `json:"vital"`         // lcp, fid, cls
    Priority     string `json:"priority"`      // high, medium, low
    Title        string `json:"title"`
    Description  string `json:"description"`
    CodeSnippet  string `json:"code_snippet,omitempty"`
    Impact       string `json:"impact"`        // Expected improvement
}

type VitalIssue struct {
    Vital       string  `json:"vital"`
    Element     string  `json:"element,omitempty"` // The element causing issue
    Value       float64 `json:"value"`
    Threshold   float64 `json:"threshold"`
    Description string  `json:"description"`
    Fix         string  `json:"fix"`
}

type PageVitalsReport struct {
    URL          string          `json:"url"`
    Vitals       *CoreWebVitals  `json:"vitals"`
    PageSource   string          `json:"-"` // HTML source for analysis
    LCPElement   *LCPSource      `json:"lcp_element,omitempty"`
    LayoutShifts []LayoutShift   `json:"layout_shifts,omitempty"`
    LongTasks    []LongTask      `json:"long_tasks,omitempty"`
}

type LCPSource struct {
    TagName  string `json:"tag_name"`
    Text     string `json:"text,omitempty"`
    Src      string `json:"src,omitempty"`
    Size     int    `json:"size"`
    LoadTime int    `json:"load_time_ms"`
    IsImage  bool   `json:"is_image"`
}

type LayoutShift struct {
    Element string  `json:"element"`
    Score   float64 `json:"score"`
    OldRect Rect    `json:"old_rect"`
    NewRect Rect    `json:"new_rect"`
}

type Rect struct {
    X      int `json:"x"`
    Y      int `json:"y"`
    Width  int `json:"width"`
    Height int `json:"height"`
}

type LongTask struct {
    Duration int    `json:"duration_ms"`
    Start    int    `json:"start_ms"`
    Name     string `json:"name,omitempty"`
}

// API Response from Chrome UX Report
type ChromeUXResponse struct {
    URL    string `json:"url"`
    Record struct {
        Key struct {
            URL string `json:"url"`
        } `json:"key"`
        Metrics map[string]struct {
            Percentiles struct {
                P75 float64 `json:"p75"`
            } `json:"percentiles"`
            Histogram []struct {
                Start   float64 `json:"start"`
                End     float64 `json:"end"`
                Density float64 `json:"density"`
            } `json:"histogram"`
        } `json:"metrics"`
    } `json:"record"`
}

// Configuration
type VitalsConfig struct {
    APIKey          string        `json:"api_key"`           // Google API Key
    Timeout         time.Duration `json:"timeout"`
    CacheTTL        time.Duration `json:"cache_ttl"`         // Cache results to avoid rate limits
    EnableFieldData bool          `json:"enable_field_data"` // Use CrUX field data
    EnableLabData   bool          `json:"enable_lab_data"`   // Run Lighthouse in headless Chrome
    FormFactor      string        `json:"form_factor"`       // "PHONE", "DESKTOP", or "ALL"
}