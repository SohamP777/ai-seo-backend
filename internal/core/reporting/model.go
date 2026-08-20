package reporting

import (
	"fmt"
	"time"
)

// ScanResult represents the complete result of an SEO scan
type ScanResult struct {
	ID              string      `json:"id" bson:"_id,omitempty"`
	URL             string      `json:"url" bson:"url"`
	StatusCode      int         `json:"status_code" bson:"status_code"`
	Title           string      `json:"title" bson:"title"`
	MetaDescription string      `json:"meta_description" bson:"meta_description"`
	MetaKeywords    string      `json:"meta_keywords" bson:"meta_keywords"`
	CanonicalURL    string      `json:"canonical_url" bson:"canonical_url"`
	RobotsTxt       string      `json:"robots_txt" bson:"robots_txt"`
	SitemapXML      string      `json:"sitemap_xml" bson:"sitemap_xml"`
	LoadTime        float64     `json:"load_time" bson:"load_time"`
	PageSize        int64       `json:"page_size" bson:"page_size"` // in bytes
	WordCount       int         `json:"word_count" bson:"word_count"`
	ImageCount      int         `json:"image_count" bson:"image_count"`
	LinkCount       int         `json:"link_count" bson:"link_count"`
	Issues          []Issue     `json:"issues" bson:"issues"`
	Pages           []*PageData `json:"pages" bson:"pages"`
	Competitors     []string    `json:"competitors" bson:"competitors"`
	GeneratedAt     time.Time   `json:"generated_at" bson:"generated_at"`
	CompletedAt     time.Time   `json:"completed_at" bson:"completed_at"`
	Duration        float64     `json:"duration" bson:"duration"` // in seconds
	Status          string      `json:"status" bson:"status"`     // pending, running, completed, failed
	Error           string      `json:"error" bson:"error"`
	UserID          string      `json:"user_id" bson:"user_id"`
	ProjectID       string      `json:"project_id" bson:"project_id"`
}

// PageData represents data for a single crawled page
type PageData struct {
	URL             string     `json:"url" bson:"url"`
	StatusCode      int        `json:"status_code" bson:"status_code"`
	Title           string     `json:"title" bson:"title"`
	MetaDescription string     `json:"meta_description" bson:"meta_description"`
	MetaKeywords    string     `json:"meta_keywords" bson:"meta_keywords"`
	Headings        *Headings  `json:"headings" bson:"headings"` // Use this only
	Links           []Link     `json:"links" bson:"links"`
	Images          []Image    `json:"images" bson:"images"`
	LoadTime        float64    `json:"load_time" bson:"load_time"`
	PageSize        int64      `json:"page_size" bson:"page_size"`
	WordCount       int        `json:"word_count" bson:"word_count"`
	ImageCount      int        `json:"image_count" bson:"image_count"`
	LinkCount       int        `json:"link_count" bson:"link_count"`
	Issues          []Issue    `json:"issues" bson:"issues"`
	Depth           int        `json:"depth" bson:"depth"`
	ParentURL       string     `json:"parent_url" bson:"parent_url"`
	Content         string     `json:"content" bson:"content"`           // raw HTML content
	TextContent     string     `json:"text_content" bson:"text_content"` // extracted text
	Canonical       string     `json:"canonical" bson:"canonical"`
	Robots          string     `json:"robots" bson:"robots"` // robots meta tag
	LastModified    time.Time  `json:"last_modified" bson:"last_modified"`
	ContentType     string     `json:"content_type" bson:"content_type"`
	IsIndexable     bool       `json:"is_indexable" bson:"is_indexable"`
	CrawledAt       time.Time  `json:"crawled_at" bson:"crawled_at"`
}

// Headings represents all heading tags on a page
type Headings struct {
	H1 []string `json:"h1" bson:"h1"`
	H2 []string `json:"h2" bson:"h2"`
	H3 []string `json:"h3" bson:"h3"`
	H4 []string `json:"h4" bson:"h4"`
	H5 []string `json:"h5" bson:"h5"`
	H6 []string `json:"h6" bson:"h6"`
}

// Link represents a hyperlink on a page
type Link struct {
	URL        string `json:"url" bson:"url"`                 // the href value
	Text       string `json:"text" bson:"text"`               // link text
	Title      string `json:"title" bson:"title"`             // title attribute
	Rel        string `json:"rel" bson:"rel"`                 // rel attribute (nofollow, sponsored, etc.)
	Target     string `json:"target" bson:"target"`           // _blank, _self, etc.
	IsInternal bool   `json:"is_internal" bson:"is_internal"`
	IsFollow   bool   `json:"is_follow" bson:"is_follow"`
	StatusCode int    `json:"status_code" bson:"status_code"` // HTTP status code if checked
	IsBroken   bool   `json:"is_broken" bson:"is_broken"`
	Element    string `json:"element" bson:"element"`         // a, area, etc.
	XPath      string `json:"xpath" bson:"xpath"`             // XPath location
}

// Image represents an image on a page
type Image struct {
	URL          string `json:"url" bson:"url"`                       // src attribute
	Alt          string `json:"alt" bson:"alt"`                       // alt text
	Title        string `json:"title" bson:"title"`                   // title attribute
	Width        int    `json:"width" bson:"width"`                   // width attribute
	Height       int    `json:"height" bson:"height"`                 // height attribute
	FileSize     int64  `json:"file_size" bson:"file_size"`           // in bytes
	FileType     string `json:"file_type" bson:"file_type"`           // jpg, png, gif, etc.
	HasAlt       bool   `json:"has_alt" bson:"has_alt"`
	IsLazyLoaded bool   `json:"is_lazy_loaded" bson:"is_lazy_loaded"` // lazy loaded
	StatusCode   int    `json:"status_code" bson:"status_code"`       // HTTP status code if checked
	IsBroken     bool   `json:"is_broken" bson:"is_broken"`
	Element      string `json:"element" bson:"element"`               // img, picture, etc.
	Loading      string `json:"loading" bson:"loading"`               // lazy, eager, auto
}

// Issue represents an SEO issue found on a page or site
type Issue struct {
	ID             string    `json:"id" bson:"id"`
	Type           string    `json:"type" bson:"type"`                       // meta_description, title_tag, heading, etc.
	Severity       string    `json:"severity" bson:"severity"`               // critical, warning, info
	Category       string    `json:"category" bson:"category"`               // content, technical, performance, mobile, etc.
	Description    string    `json:"description" bson:"description"`         // human readable description
	Element        string    `json:"element" bson:"element"`                 // HTML element with the issue
	Value          string    `json:"value" bson:"value"`                     // current value
	Expected       string    `json:"expected" bson:"expected"`               // expected value
	Recommendation string    `json:"recommendation" bson:"recommendation"`   // how to fix it
	Line           int       `json:"line" bson:"line"`                       // line number in HTML
	Column         int       `json:"column" bson:"column"`                   // column number in HTML
	PageURL        string    `json:"page_url" bson:"page_url"`               // URL where issue was found
	Selector       string    `json:"selector" bson:"selector"`               // CSS selector
	XPath          string    `json:"xpath" bson:"xpath"`                     // XPath location
	Impact         float64   `json:"impact" bson:"impact"`                   // SEO impact score (0-1)
	Effort         float64   `json:"effort" bson:"effort"`                   // effort to fix (0-1)
	Priority       float64   `json:"priority" bson:"priority"`               // priority score (impact/effort)
	Tags           []string  `json:"tags" bson:"tags"`                       // additional tags
	DetectedAt     time.Time `json:"detected_at" bson:"detected_at"`
	Fixed          bool      `json:"fixed" bson:"fixed"`
	FixedAt        time.Time `json:"fixed_at" bson:"fixed_at"`
}

// CompetitorData represents SEO data for a competitor website
type CompetitorData struct {
	Domain         string           `json:"domain" bson:"domain"`
	Pages          []*PageData      `json:"pages" bson:"pages"`
	Metrics        CompetitorMetrics `json:"metrics" bson:"metrics"`
	TopKeywords    []KeywordData    `json:"top_keywords" bson:"top_keywords"`
	Backlinks      int              `json:"backlinks" bson:"backlinks"`
	DomainAuthority int             `json:"domain_authority" bson:"domain_authority"`
	PageAuthority  int              `json:"page_authority" bson:"page_authority"`
	SpamScore      int              `json:"spam_score" bson:"spam_score"`
	AnalyzedAt     time.Time        `json:"analyzed_at" bson:"analyzed_at"`
}

// CompetitorMetrics contains metrics for competitor comparison
type CompetitorMetrics struct {
	TotalPages     int     `json:"total_pages" bson:"total_pages"`
	AvgLoadTime    float64 `json:"avg_load_time" bson:"avg_load_time"`
	AvgWordCount   float64 `json:"avg_word_count" bson:"avg_word_count"`
	TotalIssues    int     `json:"total_issues" bson:"total_issues"`
	CriticalIssues int     `json:"critical_issues" bson:"critical_issues"`
	AvgTitleLength float64 `json:"avg_title_length" bson:"avg_title_length"`
	AvgDescLength  float64 `json:"avg_desc_length" bson:"avg_desc_length"`
	TotalBacklinks int     `json:"total_backlinks" bson:"total_backlinks"`
	TotalKeywords  int     `json:"total_keywords" bson:"total_keywords"`
}

// KeywordData represents keyword ranking data
type KeywordData struct {
	Keyword          string    `json:"keyword" bson:"keyword"`
	Position         int       `json:"position" bson:"position"`
	PreviousPosition int       `json:"previous_position" bson:"previous_position"`
	SearchVolume     int       `json:"search_volume" bson:"search_volume"`
	Difficulty       int       `json:"difficulty" bson:"difficulty"`
	CPC              float64   `json:"cpc" bson:"cpc"`                 // Cost Per Click
	URL              string    `json:"url" bson:"url"`                 // ranking URL
	Title            string    `json:"title" bson:"title"`             // page title
	Description      string    `json:"description" bson:"description"` // meta description
	Traffic          float64   `json:"traffic" bson:"traffic"`         // estimated monthly traffic
	TrafficValue     float64   `json:"traffic_value" bson:"traffic_value"` // estimated traffic value
	Impressions      int       `json:"impressions" bson:"impressions"`
	Clicks           int       `json:"clicks" bson:"clicks"`
	CTR              float64   `json:"ctr" bson:"ctr"`                 // Click Through Rate
	CheckedAt        time.Time `json:"checked_at" bson:"checked_at"`
}

// TrendData represents historical SEO data for trend analysis
type TrendData struct {
	Domain     string           `json:"domain" bson:"domain"`
	DataPoints []TrendDataPoint `json:"data_points" bson:"data_points"`
	Metric     string           `json:"metric" bson:"metric"` // traffic, rankings, issues, etc.
	Period     string           `json:"period" bson:"period"` // daily, weekly, monthly
	StartDate  time.Time        `json:"start_date" bson:"start_date"`
	EndDate    time.Time        `json:"end_date" bson:"end_date"`
	Change     float64          `json:"change" bson:"change"` // percentage change
	Trend      string           `json:"trend" bson:"trend"`   // up, down, stable
}

// TrendDataPoint represents a single data point in a trend
type TrendDataPoint struct {
	Date  time.Time `json:"date" bson:"date"`
	Value float64   `json:"value" bson:"value"`
	Count int       `json:"count" bson:"count"` // number of data points
}

// ReportFilter represents filters for report data
type ReportFilter struct {
	DateRange     DateRange `json:"date_range" bson:"date_range"`
	IssueSeverity []string  `json:"issue_severity" bson:"issue_severity"`
	IssueCategory []string  `json:"issue_category" bson:"issue_category"`
	Pages         []string  `json:"pages" bson:"pages"`               // specific pages to include
	ExcludePages  []string  `json:"exclude_pages" bson:"exclude_pages"` // pages to exclude
	MinWordCount  int       `json:"min_word_count" bson:"min_word_count"`
	MaxLoadTime   float64   `json:"max_load_time" bson:"max_load_time"`
}

// DateRange represents a date range filter
type DateRange struct {
	Start time.Time `json:"start" bson:"start"`
	End   time.Time `json:"end" bson:"end"`
}

// Branding represents report branding configuration
type Branding struct {
	CompanyName    string `json:"company_name" bson:"company_name"`
	CompanyLogo    string `json:"company_logo" bson:"company_logo"` // URL or base64
	PrimaryColor   string `json:"primary_color" bson:"primary_color"`
	SecondaryColor string `json:"secondary_color" bson:"secondary_color"`
	FooterText     string `json:"footer_text" bson:"footer_text"`
	HeaderText     string `json:"header_text" bson:"header_text"`
	FontFamily     string `json:"font_family" bson:"font_family"`
}

// GeneratedReport represents a generated report
type GeneratedReport struct {
	ID           string                 `json:"id" bson:"_id,omitempty"`
	ConfigID     string                 `json:"config_id" bson:"config_id"`
	Type         string                 `json:"type" bson:"type"`
	Format       string                 `json:"format" bson:"format"`
	Domain       string                 `json:"domain" bson:"domain"`
	FilePath     string                 `json:"file_path" bson:"file_path"` // path to generated file
	FileSize     int64                  `json:"file_size" bson:"file_size"` // in bytes
	URL          string                 `json:"url" bson:"url"`             // downloadable URL
	Metadata     map[string]interface{} `json:"metadata" bson:"metadata"`
	GeneratedAt  time.Time              `json:"generated_at" bson:"generated_at"`
	ExpiresAt    time.Time              `json:"expires_at" bson:"expires_at"`
	Downloads    int                    `json:"downloads" bson:"downloads"`
	LastDownload time.Time              `json:"last_download" bson:"last_download"`
}

// EmailLog represents a sent email log
type EmailLog struct {
	ID        string    `json:"id" bson:"_id,omitempty"`
	ReportID  string    `json:"report_id" bson:"report_id"`
	To        []string  `json:"to" bson:"to"`
	CC        []string  `json:"cc" bson:"cc"`
	BCC       []string  `json:"bcc" bson:"bcc"`
	Subject   string    `json:"subject" bson:"subject"`
	Status    string    `json:"status" bson:"status"` // sent, failed, pending
	Error     string    `json:"error" bson:"error"`
	Attempts  int       `json:"attempts" bson:"attempts"`
	SentAt    time.Time `json:"sent_at" bson:"sent_at"`
	OpenedAt  time.Time `json:"opened_at" bson:"opened_at"`
	ClickedAt time.Time `json:"clicked_at" bson:"clicked_at"`
}

// Project represents a SEO project
type Project struct {
	ID          string          `json:"id" bson:"_id,omitempty"`
	Name        string          `json:"name" bson:"name"`
	Domain      string          `json:"domain" bson:"domain"`
	Description string          `json:"description" bson:"description"`
	UserID      string          `json:"user_id" bson:"user_id"`
	TeamID      string          `json:"team_id" bson:"team_id"`
	Settings    ProjectSettings `json:"settings" bson:"settings"`
	CreatedAt   time.Time       `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" bson:"updated_at"`
	LastScanAt  time.Time       `json:"last_scan_at" bson:"last_scan_at"`
	IsActive    bool            `json:"is_active" bson:"is_active"`
}

// ProjectSettings represents project-specific settings
type ProjectSettings struct {
	CrawlDepth        int      `json:"crawl_depth" bson:"crawl_depth"`
	MaxPages          int      `json:"max_pages" bson:"max_pages"`
	RespectRobotsTxt  bool     `json:"respect_robots_txt" bson:"respect_robots_txt"`
	UserAgent         string   `json:"user_agent" bson:"user_agent"`
	IncludePatterns   []string `json:"include_patterns" bson:"include_patterns"`
	ExcludePatterns   []string `json:"exclude_patterns" bson:"exclude_patterns"`
	CheckBrokenLinks  bool     `json:"check_broken_links" bson:"check_broken_links"`
	CheckImages       bool     `json:"check_images" bson:"check_images"`
	ExtractKeywords   bool     `json:"extract_keywords" bson:"extract_keywords"`
	AnalyzeCompetitors bool    `json:"analyze_competitors" bson:"analyze_competitors"`
	Schedule          string   `json:"schedule" bson:"schedule"` // cron expression
	Notifications     []string `json:"notifications" bson:"notifications"` // email, slack, webhook
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    ResponseMeta `json:"meta,omitempty"`
}

// ResponseMeta contains metadata for API responses
type ResponseMeta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// Error constants for consistent error handling
const (
	ErrorTypeValidation   = "validation_error"
	ErrorTypeNotFound     = "not_found"
	ErrorTypeUnauthorized = "unauthorized"
	ErrorTypeForbidden    = "forbidden"
	ErrorTypeInternal     = "internal_error"
	ErrorTypeBadRequest   = "bad_request"
)

// Issue severity constants
const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
)

// Issue category constants
const (
	CategoryContent       = "content"
	CategoryTechnical     = "technical"
	CategoryPerformance   = "performance"
	CategoryMobile        = "mobile"
	CategorySecurity      = "security"
	CategoryAccessibility = "accessibility"
	CategoryLinks         = "links"
	CategoryImages        = "images"
)

// Report type constants
const (
	ReportTypeSummary    = "summary"
	ReportTypeDetailed   = "detailed"
	ReportTypeExecutive  = "executive"
	ReportTypeCompetitor = "competitor"
	ReportTypeTrend      = "trend"
)

// Report format constants
const (
	ReportFormatHTML = "html"
	ReportFormatPDF  = "pdf"
	ReportFormatJSON = "json"
	ReportFormatCSV  = "csv"
	ReportFormatMD   = "md"
)

// Scan status constants
const (
	ScanStatusPending   = "pending"
	ScanStatusRunning   = "running"
	ScanStatusCompleted = "completed"
	ScanStatusFailed    = "failed"
	ScanStatusCancelled = "cancelled"
)

// Helper function to create a new ScanResult
func NewScanResult(url string) *ScanResult {
	return &ScanResult{
		URL:         url,
		Pages:       []*PageData{},
		Issues:      []Issue{},
		Competitors: []string{},
		GeneratedAt: time.Now(),
		Status:      ScanStatusPending,
	}
}

// Helper function to create a new Issue
func NewIssue(issueType, severity, description string) Issue {
	return Issue{
		ID:          GenerateID("issue"),
		Type:        issueType,
		Severity:    severity,
		Description: description,
		DetectedAt:  time.Now(),
		Fixed:       false,
	}
}

// Helper function to generate a unique ID
func GenerateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}