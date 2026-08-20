// pkg/wordpress/models.go
package wordpress

import (
    "time"
)

type WordPressSite struct {
    ID          string    `json:"id"`
    URL         string    `json:"url"`
    Username    string    `json:"username"`
    Password    string    `json:"password,omitempty"`
    IsConnected bool      `json:"is_connected"`
    Version     string    `json:"version"`
    Theme       string    `json:"theme"`
    Plugins     []string  `json:"plugins"`
    LastChecked time.Time `json:"last_checked"`
}

type FixResult struct {
    Success     bool      `json:"success"`
    Action      string    `json:"action"`
    Before      string    `json:"before"`
    After       string    `json:"after"`
    Message     string    `json:"message"`
    Error       string    `json:"error,omitempty"`
    BackupID    string    `json:"backup_id,omitempty"`
    Timestamp   time.Time `json:"timestamp"`
}

type SEOIssue struct {
    Type        string   `json:"type"`        // meta, title, content, schema, image, technical, performance
    Severity    string   `json:"severity"`    // critical, high, medium, low
    Location    string   `json:"location"`    // URL or post ID
    Description string   `json:"description"`
    FixAction   string   `json:"fix_action"`
    Current     string   `json:"current"`
    Suggested   string   `json:"suggested"`
}

type SEOReport struct {
    SiteURL          string       `json:"site_url"`
    Score            int          `json:"score"`
    Issues           []SEOIssue   `json:"issues"`
    FixesApplied     []FixResult  `json:"fixes_applied"`
    BackupCreated    bool         `json:"backup_created"`
    BackupID         string       `json:"backup_id"`
    RankingImpact    string       `json:"ranking_impact"`
    EstimatedTraffic string       `json:"estimated_traffic"`
    StartTime        time.Time    `json:"start_time"`
    EndTime          time.Time    `json:"end_time"`
}

type Backup struct {
    ID          string    `json:"id"`
    SiteURL     string    `json:"site_url"`
    CreatedAt   time.Time `json:"created_at"`
    DatabaseDump string   `json:"database_dump"`
    FilesBackup string    `json:"files_backup"`
    Size        int64     `json:"size"`
}

type FixOptions struct {
    DryRun          bool `json:"dry_run"`
    FixMeta         bool `json:"fix_meta"`
    FixContent      bool `json:"fix_content"`
    FixSchema       bool `json:"fix_schema"`
    FixImages       bool `json:"fix_images"`
    FixTechnical    bool `json:"fix_technical"`
    FixPerformance  bool `json:"fix_performance"`
    CreateBackup    bool `json:"create_backup"`
    Concurrency     int  `json:"concurrency"`
}

type WPPost struct {
    ID          int               `json:"id"`
    Title       WPPostTitle       `json:"title"`
    Content     WPPostContent     `json:"content"`
    Excerpt     WPPostExcerpt     `json:"excerpt"`
    Slug        string            `json:"slug"`
    Status      string            `json:"status"`
    Link        string            `json:"link"`
    Meta        map[string]interface{} `json:"meta"`
    YoastMeta   *YoastMeta        `json:"yoast_meta,omitempty"`
     Date     string `json:"date"`      // ← ADD
    Modified string `json:"modified"`  // ← ADD
     Author int `json:"author"` 
}

type WPPostTitle struct {
    Rendered string `json:"rendered"`
}

type WPPostContent struct {
    Rendered  string `json:"rendered"`
    Protected bool   `json:"protected"`
}

type WPPostExcerpt struct {
    Rendered  string `json:"rendered"`
    Protected bool   `json:"protected"`
}

type YoastMeta struct {
    Title       string `json:"title"`
    Description string `json:"description"`
    Canonical   string `json:"canonical"`
    NoIndex     bool   `json:"noindex"`
    NoFollow    bool   `json:"nofollow"`
    OgTitle     string `json:"opengraph-title"`
    OgDesc      string `json:"opengraph-description"`
    OgImage     string `json:"opengraph-image"`
    TwitterTitle string `json:"twitter-title"`
    TwitterDesc  string `json:"twitter-description"`
    TwitterImage string `json:"twitter-image"`
}

type FixRequest struct {
    SiteID      string     `json:"site_id"`
    Options     FixOptions `json:"options"`
}

type RollbackRequest struct {
    SiteID     string `json:"site_id"`
    BackupID   string `json:"backup_id"`
}

type StatusResponse struct {
    SiteID      string      `json:"site_id"`
    Status      string      `json:"status"` // pending, running, completed, failed
    Progress    int         `json:"progress"`
    Report      *SEOReport  `json:"report,omitempty"`
    Error       string      `json:"error,omitempty"`
}