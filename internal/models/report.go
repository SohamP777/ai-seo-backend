// Package models contains data structures for report generation
package models

import (
	"time"
)

// ReportType defines available report types
type ReportType string

const (
	ReportSiteAudit     ReportType = "site_audit"
	ReportKeywordTrack  ReportType = "keyword_tracking"
	ReportBrokenLinks   ReportType = "broken_links"
	ReportCompetitor    ReportType = "competitor_analysis"
	ReportPerformance   ReportType = "performance"
	ReportCustom        ReportType = "custom"
)

// ReportFormat defines output formats
type ReportFormat string

const (
	FormatPDF       ReportFormat = "pdf"
	FormatHTML      ReportFormat = "html"
	FormatJSON      ReportFormat = "json"
	FormatCSV       ReportFormat = "csv"
	FormatExcel     ReportFormat = "xlsx"
	FormatMarkdown  ReportFormat = "md"
)

// ReportFrequency defines how often reports are generated
type ReportFrequency string

const (
	FrequencyDaily   ReportFrequency = "daily"
	FrequencyWeekly  ReportFrequency = "weekly"
	FrequencyMonthly ReportFrequency = "monthly"
	FrequencyQuarterly ReportFrequency = "quarterly"
	FrequencyOnce    ReportFrequency = "once"
)

// ReportStatus defines report generation status
type ReportStatus string

const (
	ReportPending    ReportStatus = "pending"
	ReportGenerating ReportStatus = "generating"
	ReportCompleted  ReportStatus = "completed"
	ReportFailed     ReportStatus = "failed"
	ReportCancelled  ReportStatus = "cancelled"
)

// Report represents a generated report
type Report struct {
	ID              string                 `json:"id" db:"id"`
	UserID          string                 `json:"user_id" db:"user_id"`
	DomainID        string                 `json:"domain_id" db:"domain_id"`
	Type            ReportType              `json:"type" db:"type"`
	Format          ReportFormat            `json:"format" db:"format"`
	Status          ReportStatus            `json:"status" db:"status"`
	
	// Content
	Title           string                 `json:"title" db:"title"`
	Description     string                 `json:"description" db:"description"`
	Summary         string                 `json:"summary" db:"summary"`
	Data            map[string]interface{} `json:"data,omitempty" db:"data"`
	
	// Metrics
	Metrics         map[string]interface{} `json:"metrics" db:"metrics"`
	Issues          []ReportIssue          `json:"issues" db:"issues"`
	Recommendations []string               `json:"recommendations" db:"recommendations"`
	Score           int                    `json:"score" db:"score"` // 0-100
	
	// Files
	FilePath        string                 `json:"file_path" db:"file_path"`
	FileSize        int64                   `json:"file_size_bytes" db:"file_size"`
	FileURL         string                 `json:"file_url,omitempty" db:"file_url"`
	
	// Schedule
	IsScheduled     bool                   `json:"is_scheduled" db:"is_scheduled"`
	Frequency       ReportFrequency         `json:"frequency,omitempty" db:"frequency"`
	ScheduledAt     *time.Time              `json:"scheduled_at,omitempty" db:"scheduled_at"`
	NextRunAt       *time.Time              `json:"next_run_at,omitempty" db:"next_run_at"`
	
	// Dates
	GeneratedAt     time.Time               `json:"generated_at" db:"generated_at"`
	PeriodStart     *time.Time               `json:"period_start,omitempty" db:"period_start"`
	PeriodEnd       *time.Time               `json:"period_end,omitempty" db:"period_end"`
	CreatedAt       time.Time                `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at" db:"updated_at"`
	ExpiresAt       *time.Time                `json:"expires_at,omitempty" db:"expires_at"`
}

// ReportIssue represents an issue found in a report
type ReportIssue struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`        // missing_title, broken_link, etc.
	Severity    string   `json:"severity"`    // high, medium, low
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Element     string   `json:"element,omitempty"` // HTML element
	URL         string   `json:"url,omitempty"`     // Affected URL
	Value       string   `json:"value,omitempty"`   // Current value
	Expected    string   `json:"expected,omitempty"` // Expected value
	Fix         string   `json:"fix,omitempty"`     // How to fix
	Count       int      `json:"count,omitempty"`   // Number of occurrences
	Pages       []string `json:"pages,omitempty"`   // Affected pages
}

// ScheduledReport represents a recurring report schedule
type ScheduledReport struct {
	ID              string          `json:"id" db:"id"`
	UserID          string          `json:"user_id" db:"user_id"`
	DomainID        string          `json:"domain_id" db:"domain_id"`
	Name            string          `json:"name" db:"name"`
	Type            ReportType       `json:"type" db:"type"`
	Format          []ReportFormat   `json:"format" db:"format"` // Multiple formats
	
	// Schedule
	Frequency       ReportFrequency  `json:"frequency" db:"frequency"`
	CronExpression  string           `json:"cron_expression" db:"cron_expression"` // Custom cron
	TimeOfDay       string           `json:"time_of_day" db:"time_of_day"` // "09:00"
	DayOfWeek       int              `json:"day_of_week" db:"day_of_week"` // 1-7 for weekly
	DayOfMonth      int              `json:"day_of_month" db:"day_of_month"` // 1-31 for monthly
	
	// Delivery
	EmailRecipients []string         `json:"email_recipients" db:"email_recipients"`
	SlackWebhook    string           `json:"slack_webhook,omitempty" db:"slack_webhook"`
	WebhookURL      string           `json:"webhook_url,omitempty" db:"webhook_url"`
	
	// Filters
	IncludePages    []string         `json:"include_pages,omitempty" db:"include_pages"` // Specific pages
	ExcludePages    []string         `json:"exclude_pages,omitempty" db:"exclude_pages"`
	MinSeverity     string           `json:"min_severity" db:"min_severity"` // Only include issues >= this severity
	
	// Status
	IsActive        bool             `json:"is_active" db:"is_active"`
	LastRunAt       *time.Time       `json:"last_run_at,omitempty" db:"last_run_at"`
	LastRunStatus   ReportStatus      `json:"last_run_status,omitempty" db:"last_run_status"`
	NextRunAt       *time.Time       `json:"next_run_at,omitempty" db:"next_run_at"`
	TotalRuns       int               `json:"total_runs" db:"total_runs"`
	
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}

// ReportTemplate represents a custom report template
type ReportTemplate struct {
	ID          string                 `json:"id" db:"id"`
	UserID      string                 `json:"user_id" db:"user_id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	
	// Template content
	HeaderHTML  string                 `json:"header_html" db:"header_html"`
	FooterHTML  string                 `json:"footer_html" db:"footer_html"`
	CSS         string                 `json:"css" db:"css"`
	
	// Sections to include
	Sections    []ReportSection        `json:"sections" db:"sections"`
	
	// Branding
	LogoURL     string                 `json:"logo_url" db:"logo_url"`
	PrimaryColor string                `json:"primary_color" db:"primary_color"`
	SecondaryColor string              `json:"secondary_color" db:"secondary_color"`
	FontFamily  string                 `json:"font_family" db:"font_family"`
	
	IsDefault   bool                   `json:"is_default" db:"is_default"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// ReportSection represents a section in a custom report
type ReportSection struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Type        string   `json:"type"` // summary, issues, keywords, links, images, custom
	Order       int      `json:"order"`
	IsVisible   bool     `json:"is_visible"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// ReportExportRequest represents a request to export a report
type ReportExportRequest struct {
	ReportID     string       `json:"report_id"`
	Format       ReportFormat `json:"format"`
	IncludeData  bool         `json:"include_data"`
	IncludeCharts bool        `json:"include_charts"`
	PageSize     string       `json:"page_size"` // A4, Letter
	Orientation  string       `json:"orientation"` // portrait, landscape
}

// ReportShare represents a shared report
type ReportShare struct {
	ID          string     `json:"id" db:"id"`
	ReportID    string     `json:"report_id" db:"report_id"`
	ShareToken  string     `json:"share_token" db:"share_token"`
	CreatedBy   string     `json:"created_by" db:"created_by"`
	
	// Access control
	IsPublic    bool       `json:"is_public" db:"is_public"`
	Password    string     `json:"-" db:"password"` // Password protected
	AllowedEmails []string `json:"allowed_emails,omitempty" db:"allowed_emails"`
	
	// Expiration
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	MaxViews    int        `json:"max_views" db:"max_views"`
	ViewCount   int        `json:"view_count" db:"view_count"`
	
	// Tracking
	LastViewedAt *time.Time `json:"last_viewed_at,omitempty" db:"last_viewed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// SiteAuditData represents data for a site audit report
type SiteAuditData struct {
	Domain          string                 `json:"domain"`
	TotalPages      int                    `json:"total_pages"`
	PagesCrawled    int                    `json:"pages_crawled"`
	CrawlTime       time.Duration          `json:"crawl_time_ms"`
	
	// Overview
	OverallScore    int                    `json:"overall_score"`
	Grade           string                 `json:"grade"` // A+, A, B, C, D, F
	
	// Issue breakdown
	IssuesByType    map[string]int         `json:"issues_by_type"`
	IssuesBySeverity map[string]int        `json:"issues_by_severity"`
	TopIssues       []ReportIssue          `json:"top_issues"`
	
	// Page stats
	PagesWithIssues []PageIssueSummary     `json:"pages_with_issues"`
	SlowestPages    []PagePerformance      `json:"slowest_pages"`
	
	// Content stats
	AvgWordCount    float64                `json:"avg_word_count"`
	TotalWords      int                    `json:"total_words"`
	PagesWithThinContent int               `json:"pages_with_thin_content"` // <300 words
	
	// Link stats
	TotalLinks      int                    `json:"total_links"`
	InternalLinks   int                    `json:"internal_links"`
	ExternalLinks   int                    `json:"external_links"`
	BrokenLinks     int                    `json:"broken_links"`
	
	// Image stats
	TotalImages     int                    `json:"total_images"`
	ImagesMissingAlt int                   `json:"images_missing_alt"`
	TotalImageSize  int64                  `json:"total_image_size_bytes"`
	
	// Performance
	AvgLoadTime     time.Duration          `json:"avg_load_time_ms"`
	PagesSlow       int                    `json:"pages_slow"` // >3 seconds
	
	// Mobile
	AvgMobileScore  int                    `json:"avg_mobile_score"`
	MobileFriendly  int                    `json:"mobile_friendly_pages"` // Count
}

// PageIssueSummary summarizes issues on a single page
type PageIssueSummary struct {
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	IssueCount  int      `json:"issue_count"`
	HighCount   int      `json:"high_count"`
	MediumCount int      `json:"medium_count"`
	LowCount    int      `json:"low_count"`
	Score       int      `json:"score"` // Page score 0-100
}

// PagePerformance represents performance data for a page
type PagePerformance struct {
	URL         string        `json:"url"`
	LoadTime    time.Duration `json:"load_time_ms"`
	PageSize    int64         `json:"page_size_bytes"`
	RequestCount int          `json:"request_count"`
}

// KeywordReportData represents data for a keyword tracking report
type KeywordReportData struct {
	Domain          string              `json:"domain"`
	Period          string              `json:"period"` // "Last 30 days"
	DateRange       []string            `json:"date_range"`
	
	// Overview
	TotalKeywords   int                 `json:"total_keywords"`
	AvgPosition     float64             `json:"avg_position"`
	BestPosition    int                 `json:"best_position"`
	WorstPosition   int                 `json:"worst_position"`
	
	// Changes
	Improved        int                 `json:"improved"`
	Declined        int                 `json:"declined"`
	NewInTop10      int                 `json:"new_in_top_10"`
	NewInTop3       int                 `json:"new_in_top_3"`
	
	// Keyword list
	Keywords        []KeywordRank       `json:"keywords"`
	
	// Competitors
	Competitors     []CompetitorRank    `json:"competitors"`
	
	// SERP features
	FeaturedSnippets int               `json:"featured_snippets"`
	PeopleAlsoAsk   int                 `json:"people_also_ask"`
	LocalPack       int                 `json:"local_pack"`
}

// KeywordRank represents a keyword's ranking data
type KeywordRank struct {
	Keyword      string    `json:"keyword"`
	Position     int       `json:"position"`
	Previous     int       `json:"previous"`
	Change       int       `json:"change"`
	URL          string    `json:"url"`
	SearchVolume int       `json:"search_volume"`
	Difficulty   int       `json:"difficulty"` // 0-100
	Intent       string    `json:"intent"` // informational, commercial, transactional, navigational
}

// CompetitorRank represents a competitor's ranking for keywords
type CompetitorRank struct {
	Domain       string             `json:"domain"`
	Keywords     map[string]int     `json:"keywords"` // keyword -> position
	AvgPosition  float64            `json:"avg_position"`
	Overlap      int                `json:"overlap"` // keywords both rank for
}

// BrokenLinksReportData represents data for a broken links report
type BrokenLinksReportData struct {
	Domain          string              `json:"domain"`
	TotalBroken     int                 `json:"total_broken"`
	BrokenByType    map[string]int      `json:"broken_by_type"` // 404, 500, timeout, etc.
	
	// Broken links grouped by source page
	BrokenLinks     map[string][]BrokenLink `json:"broken_links"`
	
	// External vs internal
	InternalBroken  int                 `json:"internal_broken"`
	ExternalBroken  int                 `json:"external_broken"`
	
	// Recommendations
	Recommendations []string            `json:"recommendations"`
}

// BrokenLink represents a single broken link
type BrokenLink struct {
	URL         string `json:"url"`
	SourcePage  string `json:"source_page"`
	AnchorText  string `json:"anchor_text"`
	StatusCode  int    `json:"status_code"`
	Error       string `json:"error,omitempty"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

// ==================== HELPER METHODS ====================

// GetGrade returns a letter grade based on score
func (r *Report) GetGrade() string {
	switch {
	case r.Score >= 95:
		return "A+"
	case r.Score >= 85:
		return "A"
	case r.Score >= 75:
		return "B"
	case r.Score >= 60:
		return "C"
	case r.Score >= 40:
		return "D"
	default:
		return "F"
	}
}

// CountIssuesBySeverity returns count of issues by severity
func (r *Report) CountIssuesBySeverity() map[string]int {
	counts := map[string]int{
		"high":   0,
		"medium": 0,
		"low":    0,
	}
	for _, issue := range r.Issues {
		counts[issue.Severity]++
	}
	return counts
}

// IsExpired checks if report is expired
func (r *Report) IsExpired() bool {
	if r.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*r.ExpiresAt)
}

// NextRun calculates next run time for scheduled report
func (s *ScheduledReport) NextRun() (*time.Time, error) {
	now := time.Now()
	
	switch s.Frequency {
	case FrequencyDaily:
		// Parse time of day
		t, err := time.Parse("15:04", s.TimeOfDay)
		if err != nil {
			return nil, err
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), 
			t.Hour(), t.Minute(), 0, 0, now.Location())
		if next.Before(now) {
			next = next.AddDate(0, 0, 1)
		}
		return &next, nil
		
	case FrequencyWeekly:
		// Implementation for weekly
		// ...
		
	case FrequencyMonthly:
		// Implementation for monthly
		// ...
	}
	
	return nil, nil
}