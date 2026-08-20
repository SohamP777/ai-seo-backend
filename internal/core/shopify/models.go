// pkg/shopify/models.go - UPDATED with GID support
package shopify

import (
    "fmt"
    "time"
)

type ShopifyStore struct {
    ID          string    `json:"id"`
    URL         string    `json:"url"`
    AccessToken string    `json:"access_token"`
    APIVersion  string    `json:"api_version"`
    Scopes      []string  `json:"scopes"`
    Plan        string    `json:"plan"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// GetGID returns the GraphQL Global ID
func (s *ShopifyStore) GetGID() string {
    return fmt.Sprintf("gid://shopify/Shop/%s", s.ID)
}

// GetGID returns the GraphQL Global ID
func (p *ShopifyProduct) GetGID() string {
    return fmt.Sprintf("gid://shopify/Product/%d", p.ID)
}

type Collection struct {
    ID          int64       `json:"id"`
    Title       string      `json:"title"`
    Handle      string      `json:"handle"`
    BodyHTML    string      `json:"body_html"`
    Image       Image       `json:"image"`
    SEO         SEOData     `json:"seo"`
}

// GetGID returns the GraphQL Global ID
func (c *Collection) GetGID() string {
    return fmt.Sprintf("gid://shopify/Collection/%d", c.ID)
}

type Page struct {
    ID          int64       `json:"id"`
    Title       string      `json:"title"`
    BodyHTML    string      `json:"body_html"`
    Handle      string      `json:"handle"`
    SEO         SEOData     `json:"seo"`
}

// GetGID returns the GraphQL Global ID
func (p *Page) GetGID() string {
    return fmt.Sprintf("gid://shopify/Page/%d", p.ID)
}

type Blog struct {
    ID          int64       `json:"id"`
    Title       string      `json:"title"`
    Handle      string      `json:"handle"`
}

type Article struct {
    ID          int64       `json:"id"`
    BlogID      int64       `json:"blog_id"`
    Title       string      `json:"title"`
    BodyHTML    string      `json:"body_html"`
    Handle      string      `json:"handle"`
    Author      string      `json:"author"`
    SEO         SEOData     `json:"seo"`
    Image       Image       `json:"image"`
}

// GetGID returns the GraphQL Global ID
func (a *Article) GetGID() string {
    return fmt.Sprintf("gid://shopify/Article/%d", a.ID)
}

type SEOData struct {
    Title       string `json:"title"`
    Description string `json:"description"`
}

type Image struct {
    ID          int64     `json:"id"`
    Src         string    `json:"src"`
    Alt         string    `json:"alt"`
    Width       int       `json:"width"`
    Height      int       `json:"height"`
    Position    int       `json:"position"`
}

type Variant struct {
    ID           int64   `json:"id"`
    Title        string  `json:"title"`
    Price        string  `json:"price"`
    SKU          string  `json:"sku"`
    InventoryQty int     `json:"inventory_quantity"`
}


type Theme struct {
    ID          int64  `json:"id"`
    Name        string `json:"name"`
    Role        string `json:"role"`
    CreatedAt   string `json:"created_at"`
    UpdatedAt   string `json:"updated_at"`
}

type ThemeAsset struct {
    Key         string `json:"key"`
    Value       string `json:"value"`
    ContentType string `json:"content_type"`
    Size        int    `json:"size"`
    UpdatedAt   string `json:"updated_at"`
}

type FixResult struct {
    Success     bool      `json:"success"`
    Action      string    `json:"action"`
    Target      string    `json:"target"`
    Before      string    `json:"before"`
    After       string    `json:"after"`
    Message     string    `json:"message"`
    Error       string    `json:"error"`
    BackupID    string    `json:"backup_id"`
    Timestamp   time.Time `json:"timestamp"`
}

type SEOIssue struct {
    Type        string   `json:"type"`
    Severity    string   `json:"severity"`
    Target      string   `json:"target"`
    Description string   `json:"description"`
    Suggestion  string   `json:"suggestion"`
    Current     string   `json:"current"`
    Suggested   string   `json:"suggested"`
}

type SEOReport struct {
    StoreURL         string       `json:"store_url"`
    Score            int          `json:"score"`
    TotalProducts    int          `json:"total_products"`
    TotalPages       int          `json:"total_pages"`
    TotalArticles    int          `json:"total_articles"`
    Issues           []SEOIssue   `json:"issues"`
    FixesApplied     []FixResult  `json:"fixes_applied"`
    ThemeBackupID    string       `json:"theme_backup_id"`
    RankingImpact    string       `json:"ranking_impact"`
}


type Backup struct {
    ID          string                 `json:"id"`
    StoreID     string                 `json:"store_id"`
    ThemeID     int64                  `json:"theme_id"`
    Assets      map[string]ThemeAsset  `json:"assets"`
    CreatedAt   time.Time              `json:"created_at"`
    Description string                 `json:"description"`
}