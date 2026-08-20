package fixer

import (
	"net/http"
	"time"
	"log"
)

// ============ CONFIGURATION ============

type Config struct {
	// Common fields
	SiteURL         string        `json:"site_url"`
	Platform        string        `json:"platform"`
	Port            string        `json:"port"`
	BackupDir       string        `json:"backup_dir"`
	RequestTimeout  time.Duration `json:"request_timeout"`
	MaxRetries      int           `json:"max_retries"`
	RetryDelay      time.Duration `json:"retry_delay"`
	DryRun          bool          `json:"dry_run"`
	Timeout         time.Duration `json:"timeout"`
	MaxConcurrent   int           `json:"max_concurrent"`
	UserAgent       string        `json:"user_agent"`
	
	// API credentials
	GoogleAPIKey    string `json:"google_api_key"`
	APIToken        string `json:"api_token"`
	AccessToken     string `json:"access_token"`
	
	// WordPress credentials
	WordPressUsername string `json:"wordpress_username"`
	WordPressPassword string `json:"wordpress_password"`
	WPURL             string `json:"wp_url"`
	WPUser            string `json:"wp_user"`
	WPPass            string `json:"wp_pass"`
	
	// Shopify credentials
	ShopifyToken    string `json:"shopify_token"`
	ShopifyShop     string `json:"shopify_shop"`
	ShopifyURL      string `json:"shopify_url"`
	
	// Database
	DatabaseDSN     string `json:"database_dsn"`
	
	// Crawling
	MaxCrawlPages    int `json:"max_crawl_pages"`
	CrawlConcurrency int `json:"crawl_concurrency"`
	 MaxPages    int `json:"max_pages"`    // ← ADD
    Concurrency int `json:"concurrency"`  
}

// ============ BACKUP ============

type BackupManager struct {
	ID          string    `json:"id"`
	SiteURL     string    `json:"site_url"`
	Platform    string    `json:"platform"`
	Type        string    `json:"type"` // full, partial, snapshot
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
	StoragePath string    `json:"storage_path"`
	BackupDir string
}

// ============ CREDENTIALS ============

type WordPressCredentials struct {
	SiteURL  string `json:"site_url"`
	Username string `json:"username"`
	Password string `json:"password"`
	
	// WordPress connection details
	SSHKey   string `json:"ssh_key"`
	Port     int    `json:"port"`
	Host     string `json:"host"`
	
	// Database connection details
	DBName   string `json:"db_name"`
	DBUser   string `json:"db_user"`
	DBPass   string `json:"db_pass"`
}

type ShopifyCredentials struct {
	StoreURL    string `json:"store_url"`
	AccessToken string `json:"access_token"`
	APIVersion  string `json:"api_version"`
}

type CloudflareCredentials struct {
	APIToken string `json:"api_token"`
	ZoneID   string `json:"zone_id"`
	Email    string `json:"email"`
	APIKey   string `json:"api_key"`
}

// ============ CLIENTS (DATA) ============

type WordPressClient struct {
	SiteURL  string
	Username string
	Password string
	Client   *http.Client
}

func (w *WordPressClient) Do(req *http.Request) (*http.Response, error) {
    return w.Client.Do(req)
}

type ShopifyClient struct {
	StoreURL    string
	AccessToken string
	Client      *http.Client
}

type CloudflareClient struct {
	APIToken  string       `json:"api_token"`
	ZoneID    string       `json:"zone_id"`
	Email     string       `json:"email"`
	APIKey    string       `json:"api_key"`
	Client    *http.Client `json:"-"`
	BaseURL   string       `json:"-"`
}

// PlatformClient is a generic interface for platform-specific clients
type PlatformClient interface {
	GetSiteURL() string
	GetPlatform() string
}

// ============ FIX RESULTS ============

type FixResult struct {
	Success     bool      `json:"success"`
	Action      string    `json:"action"`
	Target      string    `json:"target"`
	Before      string    `json:"before,omitempty"`
	After       string    `json:"after,omitempty"`
	Message     string    `json:"message"`
	Error       string    `json:"error,omitempty"`
	BackupID    string    `json:"backup_id,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	
	// Additional fields
	IssueType   string `json:"issue_type,omitempty"`
	Fixed       bool   `json:"fixed,omitempty"`
	Count       int    `json:"count,omitempty"`
	Improvement string `json:"improvement,omitempty"`
}

type FixReport struct {
    TotalFixes         int
    SuccessfulFixes    int
    FailedFixes        int
    Results            []FixResult
    BackupID           string
    Duration           time.Duration
    
    // ADD THESE IMAGE-SPECIFIC FIELDS:
    TotalImages        int
    ImagesOptimized    int
    FailedImages       int
    OriginalTotalBytes int64
    NewTotalBytes      int64
    BytesSaved         int64
    PercentSaved       float64
    Errors             []string
	WebPConverted    int     `json:"webp_converted"`     // ← ADD
    AltTextsAdded    int     `json:"alt_texts_added"`    // ← ADD
    DimensionsAdded  int     `json:"dimensions_added"`   // ← ADD
    LazyLoadAdded    int     `json:"lazy_load_added"`    // ← ADD
	 EstimatedLCPImprovementMs  int     `json:"estimated_lcp_improvement_ms"`   // ← ADD
    EstimatedCLSImprovement    float64 `json:"estimated_cls_improvement"`      // ← ADD
}


// ============ SEO ISSUES ============

type SEOIssue struct {
	Type        string   `json:"type"`
	Severity    string   `json:"severity"` // critical, high, medium, low
	Element     string   `json:"element"`
	Description string   `json:"description"`
	Fix         string   `json:"fix"`
	Fixed bool `json:"fixed"`
	AutoFixable bool     `json:"auto_fixable"`
	FixAction    string                 `json:"fix_action"`    // ← ADD
    MetricBefore map[string]interface{} `json:"metric_before"` // ← ADD
    MetricAfter  map[string]interface{} `json:"metric_after"`  // ← ADD
	
	// Additional fields
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
	Location    string `json:"location,omitempty"`
	Fixable     bool   `json:"fixable,omitempty"`
	 URL           string `json:"url"`             // ← ADD
    StatusCode    int    `json:"status_code"`     // ← ADD
    Message       string `json:"message"`         // ← ADD
    Priority      string `json:"priority"`        // ← CHANGE FROM int TO string
    FixSuggestion string `json:"fix_suggestion"`  // ← ADD
	
	
}

// ============ IMAGE INFO ============

type ImageInfo struct {
    URL           string
    PageURL       string
    OriginalSize  int64
    NewSize       int64   // ← ADD THIS
    Format        string
    Width         int
    Height        int
    HasAlt        bool
    AltText       string
    WebPConverted bool    // ← ADD THIS
}

// ============ REDIRECT & BROKEN LINKS ============

type Redirect struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	FromURL   string    `json:"from_url,omitempty"` // Deprecated: use From
	ToURL     string    `json:"to_url,omitempty"`   // Deprecated: use To
	Type      string    `json:"type"`               // 301, 302, 307
	Status    string    `json:"status"`             // active, pending, disabled
	CreatedAt time.Time `json:"created_at"`
	HitCount  int       `json:"hit_count"`
}

type BrokenLink struct {
	URL             string  `json:"url"`               // The page URL where broken link was found
	SourceURL       string  `json:"source_url"`
	TargetURL       string  `json:"target_url"`
	StatusCode      int     `json:"status_code"`
	SuggestedTarget string  `json:"suggested_target"`
	ConfidenceScore float64 `json:"confidence_score"`
}

// ============ SCHEMA ============

type SchemaMarkup struct {
	Context     string                 `json:"@context"`
	Type        string                 `json:"@type"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Image       string                 `json:"image,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// ============ PERFORMANCE DATA ============

type CoreWebVitals struct {
	LCP   int     `json:"lcp"`   // Largest Contentful Paint (ms)
	FID   int     `json:"fid"`   // First Input Delay (ms)
	CLS   float64 `json:"cls"`   // Cumulative Layout Shift
	FCP   int     `json:"fcp"`   // First Contentful Paint (ms)
	TTFB  int     `json:"ttfb"`  // Time To First Byte (ms)
	Score int     `json:"score"`
}

type PerformanceData struct {
	Score              int     `json:"score"`
	CompressionEnabled bool    `json:"compression_enabled"`
	CachingEnabled     bool    `json:"caching_enabled"`
	MinifyEnabled      bool    `json:"minify_enabled"`
	LazyLoadEnabled    bool    `json:"lazy_load_enabled"`
	PageSpeedMobile    int     `json:"pagespeed_mobile"`
	PageSpeedDesktop   int     `json:"pagespeed_desktop"`
	CoreWebVitals      *CoreWebVitals `json:"core_web_vitals,omitempty"`
	TTFB        time.Duration
    LoadTime    time.Duration
    TotalSize   int64
    CacheStatus string
	Compression  string    // ← ADD
    TLSVersion   string    // ← ADD
    RequestsCount int       // ← ADD
}

type PerformanceReport struct {
    // Before metrics
    ScoreBefore    int
    LCPBefore      int
    FIDBefore      int
    CLSBefore      float64
    TTFBBefore     int
    
    // After metrics
    ScoreAfter     int
    LCPAfter       int
    FIDAfter       int
    CLSAfter       float64
    TTFBAfter      int
    
    // Improvement metrics
    ScoreImprovement   int
    EstimatedRankBoost string
    ExecutionTime      float64
    Success            bool
    Error              string
    
    // Lists (ADD THESE)
    IssuesFound      []string `json:"issues_found"`
    IssuesFixed      []string `json:"issues_fixed"`
    Recommendations  []string `json:"recommendations"`
    
    // Other fields
    ImprovementPercent   float64
    CoreWebVitalsBefore  *CoreWebVitals
    CoreWebVitalsAfter   *CoreWebVitals
    PageSpeedBefore      int
    PageSpeedAfter       int
}

// ============ SEO SCORE & SNAPSHOT ============

type SEOScore struct {
    TotalScore                  int      `json:"total_score"`        // ← ADD THIS
    TotalIssues                 int      `json:"total_issues"`
    Grade                       string   `json:"grade"`
    IssuesFound                 []string `json:"issues_found"`
    FixedIssues                 []string `json:"fixed_issues"`
    BrokenLinks                 int      `json:"broken_links"`
    RedirectChains              int      `json:"redirect_chains"`
    Score                       int      `json:"score"`
    Improvement                 string   `json:"improvement"`
    ExpectedRankingImprovement  string   `json:"expected_ranking_improvement"`
}

type SEOSnapshot struct {
    URL               string    `json:"url"`
    OrganicTraffic    int       `json:"organic_traffic"`
    BacklinksCount    int       `json:"backlinks_count"`
    DomainAuthority   int       `json:"domain_authority"`
    IndexedPages      int       `json:"indexed_pages"`
    CoreWebVitals     struct {
        LCP float64 `json:"lcp"`
        INP float64 `json:"inp"`
        CLS float64 `json:"cls"`
    } `json:"core_web_vitals"`
    SchemaMarkupCount   int `json:"schema_markup_count"`
    MetaIssuesCount     int `json:"meta_issues_count"`
    SEOScore            int `json:"seo_score"`
	 Timestamp   time.Time `json:"timestamp"` 
}

// ============ ANALYSIS RESULT ============

type AnalysisResult struct {
	 WebsiteURL     string   `json:"website_url"`      // ← ADD
    PagesChecked   int      `json:"pages_checked"`    // ← ADD
	SiteURL     string       `json:"site_url"`
	Platform    string       `json:"platform"`
	Timestamp   time.Time    `json:"timestamp"`
	TotalPages  int          `json:"total_pages"`
	IssuesFound []SEOIssue   `json:"issues_found"`
	Score       SEOScore     `json:"score"`
	 Issues         []SEOIssue  
	Performance PerformanceData `json:"performance"`
	Recommendations []string 
}

// ============ PAGE DATA ============

type PageData struct {
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	StatusCode  int               `json:"status_code"`
	ContentType string            `json:"content_type"`
	WordCount   int               `json:"word_count"`
	Images      []ImageInfo       `json:"images"`
	Links       []string          `json:"links"`
	BrokenLinks []BrokenLink      `json:"broken_links"`
	Schema      []SchemaMarkup    `json:"schema"`
	Canonical   string            `json:"canonical"`
	H1          string            `json:"h1"`
	H2          []string          `json:"h2"`
	MetaRobots  string            `json:"meta_robots"`
	 LoadTime      time.Duration   // ← ADD
    FinalURL      string          // ← ADD
    RedirectChain []string        // ← ADD
}

// ============ SEO REPORT ============

type SEOReport struct {
	SiteURL         string        `json:"site_url"`
	Platform        string        `json:"platform"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	FixesApplied    []FixResult   `json:"fixes_applied"`
	ScoreBefore     int           `json:"score_before"`
	ScoreAfter      int           `json:"score_after"`
	Improvement     string        `json:"improvement"`
	Errors          []string      `json:"errors"`
	Recommendations []string      `json:"recommendations"`
	 Timestamp time.Time `json:"timestamp"` 
}

// ============ GUIDE ============

type Guide struct {
	ID             string      `json:"id"`
	Title          string      `json:"title"`
	IssueType      string      `json:"issue_type"`
	Platform       string      `json:"platform"`
	Difficulty     string      `json:"difficulty"`     // beginner, intermediate, advanced
	EstimatedTime  string      `json:"estimated_time"`
	Summary        string      `json:"summary"`
	Steps          []GuideStep `json:"steps"`
	Verification   string      `json:"verification"`
}

type GuideStep struct {
	Number       int      `json:"number"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Navigation   string   `json:"navigation"`
	Actions      []string `json:"actions"`
	CodeSnippet  string   `json:"code_snippet,omitempty"`
	Warning      string   `json:"warning,omitempty"`
	Tip          string   `json:"tip,omitempty"`
}
type Zone struct {
    ID     string `json:"id"`
    Name   string `json:"name"`
    Status string `json:"status"`
}

type OptimizationResult struct {
    Success bool
    Message string
	Duration    time.Duration     `json:"duration"`
    BeforePerf  *PerformanceData  `json:"before_perf"`
    AfterPerf   *PerformanceData  `json:"after_perf"`
	ErrorMessage string `json:"error_message"`
	Improvement  float64 `json:"improvement"`   // ← ADD
    IssuesFixed  int     `json:"issues_fixed"`  // ← ADD
    TotalIssues  int     `json:"total_issues"`  // ← ADD
}
type SEOAnalyzer struct {
    client *http.Client
    logger *log.Logger
}

type Validator struct {
    client *http.Client
}

type APIServer struct {
    analyzer  *SEOAnalyzer
    validator *Validator
    backup    *BackupManager
}


type RedirectRule struct {
    FromPath string `json:"from_path"`
    ToURL    string `json:"to_url"`
    Type     string `json:"type"`
    Reason   string `json:"reason"`
}

type SEOFix struct {
    ID          string
    Type        string
    Target      string
    BeforeValue string
    AfterValue  string
}

type RestoreResult struct {
    Success        bool          `json:"success"`
    BackupID       string        `json:"backup_id"`        // ← ADD
    Duration       time.Duration `json:"duration"`         // ← ADD
    SEODelta       int           `json:"seo_delta"`        // ← ADD
    SEOImprovement bool          `json:"seo_improvement"`  // ← ADD
    Message        string        `json:"message"`
    Errors         []string      `json:"errors"`           // ← ADD
}

type SEOProgress struct {
    SiteURL        string              `json:"site_url"`
    Days           int                 `json:"days"`             // ← ADD
    CurrentScore   int                 `json:"current_score"`    // ← ADD
    Improvement    int                 `json:"improvement"`
    TrafficGrowth  int                 `json:"traffic_growth"`
    BacklinkGrowth int                 `json:"backlink_growth"`
    Points         []SEOProgressPoint  `json:"points"`           // ← ADD
}

type SEOProgressPoint struct {
    Date      time.Time `json:"date"`
    SEOScore  int       `json:"seo_score"`
    Traffic   int       `json:"traffic"`
    Backlinks int       `json:"backlinks"`
	BacklinksCount int       `json:"backlinks_count"`
}

type Backup struct {
    ID          string
    Type        string      `json:"type"`        // ← ADD
    Description string      `json:"description"` // ← ADD
    SEOBefore   *SEOSnapshot `json:"seo_before"`  // ← ADD
    Size        int64       `json:"size"`        // ← ADD
    SiteURL     string
    Platform    string
    CreatedAt   time.Time
    Data        []byte
}

type PageSpeedData struct {
    Score  int     `json:"score"`
    LCP    int     `json:"lcp"`
    FID    int     `json:"fid"`
    CLS    float64 `json:"cls"`
    TTFB   int     `json:"ttfb"`
    Issues []string `json:"issues"`
}