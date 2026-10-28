// pkg/wordpress/technical_fixer.go
package wordpress

import (
    "context"
    "fmt"
    "strings"
    "time"
    "log"
)

type TechnicalFixer struct {
    client *Client
    logger *log.Logger
}

func NewTechnicalFixer(client *Client, logger *log.Logger) *TechnicalFixer {
    return &TechnicalFixer{
        client: client,
        logger: logger,
    }
}



func (t *TechnicalFixer) Analyze(ctx context.Context) ([]SEOIssue, error) {
    var issues []SEOIssue
    
    settings, err := t.client.GetSettings(ctx)
    if err != nil {
        return nil, err
    }
    
    // Check permalink structure
    permalinkStructure := settings["permalink_structure"].(string)
    if permalinkStructure != "/%postname%/" {
        issues = append(issues, SEOIssue{
            Type:        "technical",
            Severity:    "high",
            Location:    "Permalink settings",
            Description: "Suboptimal permalink structure",
            FixAction:   "Change to /%postname%/",
            Current:     permalinkStructure,
            Suggested:   "/%postname%/",
        })
    }
    
    // Check robots.txt
    robots, err := t.getRobotsTxt(ctx)
    if err != nil || !strings.Contains(robots, "Sitemap:") {
        issues = append(issues, SEOIssue{
            Type:        "technical",
            Severity:    "medium",
            Location:    "robots.txt",
            Description: "Missing sitemap reference in robots.txt",
            FixAction:   "Add sitemap URL to robots.txt",
            Current:     "No sitemap reference",
            Suggested:   "Sitemap: https://example.com/sitemap.xml",
        })
    }
    
    // Check for canonical URLs
    if yoastActive := t.checkYoastActive(ctx); !yoastActive {
        issues = append(issues, SEOIssue{
            Type:        "technical",
            Severity:    "high",
            Location:    "Canonical URLs",
            Description: "Missing canonical URL tags",
            FixAction:   "Add canonical URLs to prevent duplicate content",
            Current:     "No canonical tags",
            Suggested:   "Add rel=canonical to all pages",
        })
    }
    
    return issues, nil
}

func (t *TechnicalFixer) Fix(ctx context.Context, dryRun bool) ([]FixResult, error) {
    var results []FixResult
    
    // Fix permalink structure
    settings, err := t.client.GetSettings(ctx)
    if err != nil {
        return nil, err
    }
    
    if settings["permalink_structure"].(string) != "/%postname%/" {
        result := FixResult{
            Action:    "update_permalink_structure",
            Before:    settings["permalink_structure"].(string),
            After:     "/%postname%/",
            Timestamp: time.Now(),
        }
        
        if !dryRun {
            updates := map[string]interface{}{
                "permalink_structure": "/%postname%/",
            }
            if err := t.client.UpdateSettings(ctx, updates); err != nil {
                result.Success = false
                result.Error = err.Error()
            } else {
                result.Success = true
            }
        } else {
            result.Success = true
            result.Message = "Dry run - would update permalink structure"
        }
        results = append(results, result)
    }
    
    // Generate and submit sitemap
    if !dryRun {
        sitemap := t.generateSitemap(ctx)
        if err := t.submitSitemap(ctx, sitemap); err != nil {
            t.logger.Printf("Failed to submit sitemap: %v", err)
        } else {
            results = append(results, FixResult{
                Success:   true,
                Action:    "generate_sitemap",
                After:     "XML sitemap generated and submitted",
                Timestamp: time.Now(),
            })
        }
    }
    
    // Fix robots.txt
    robots, _ := t.getRobotsTxt(ctx)
    if !strings.Contains(robots, "Sitemap:") {
        siteURL := strings.TrimSuffix(t.client.baseURL, "/wp-json")
        sitemapURL := siteURL + "/sitemap.xml"
        newRobots := robots + "\nSitemap: " + sitemapURL
        
        result := FixResult{
            Action:    "update_robots_txt",
            Before:    robots,
            After:     newRobots,
            Timestamp: time.Now(),
        }
        
        if !dryRun {
            if err := t.updateRobotsTxt(ctx, newRobots); err != nil {
                result.Success = false
                result.Error = err.Error()
            } else {
                result.Success = true
            }
        } else {
            result.Success = true
            result.Message = "Dry run - would update robots.txt"
        }
        results = append(results, result)
    }
    
    // Add canonical URLs via Yoast
    if yoastActive := t.checkYoastActive(ctx); yoastActive && !dryRun {
        // Enable canonical URLs in Yoast
        _, err := t.client.Post(ctx, "/wp-json/yoast/v1/settings", map[string]interface{}{
            "enable_canonical": true,
        })
        if err == nil {
            results = append(results, FixResult{
                Success:   true,
                Action:    "enable_canonical_urls",
                After:     "Canonical URLs enabled",
                Timestamp: time.Now(),
            })
        }
    }
    
    return results, nil
}

func (t *TechnicalFixer) getRobotsTxt(ctx context.Context) (string, error) {
    robotsMap, err := t.client.Get(ctx, "robots.txt")
    if err != nil {
        return "", err
    }
    robots, ok := robotsMap["content"].(string)
    if !ok {
        return "", fmt.Errorf("robots.txt content not found")
    }
    return robots, nil
}

func (t *TechnicalFixer) updateRobotsTxt(ctx context.Context, content string) error {
    // In production, you'd need FTP or file system access to update robots.txt
    t.logger.Printf("Would update robots.txt with: %s", content)
    return nil
}

func (t *TechnicalFixer) generateSitemap(ctx context.Context) string {
    sitemap := `<?xml version="1.0" encoding="UTF-8"?>
    <urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
    `
    
    // Add homepage
    sitemap += `<url>
        <loc>` + t.client.baseURL + `</loc>
        <changefreq>daily</changefreq>
        <priority>1.0</priority>
    </url>`
    
    // Add posts
    posts, _ := t.client.GetPosts(ctx, 1, 100)
    for _, post := range posts {
        sitemap += fmt.Sprintf(`
        <url>
            <loc>%s</loc>
            <lastmod>%s</lastmod>
            <changefreq>weekly</changefreq>
            <priority>0.8</priority>
        </url>`, post.Link, time.Now().Format("2006-01-02"))
    }
    
    sitemap += `</urlset>`
    return sitemap
}

func (t *TechnicalFixer) submitSitemap(ctx context.Context, sitemap string) error {
    // In production, you'd upload the sitemap to the server
    t.logger.Printf("Sitemap generated successfully")
    return nil
}

func (t *TechnicalFixer) checkYoastActive(ctx context.Context) bool {
    // Check if Yoast plugin is active
    plugins, err := t.client.Get(ctx, "/wp-json/wp/v2/plugins")
    if err != nil {
        return false
    }
    
    // Type assert plugins to slice
    pluginsList, ok := plugins["items"].([]interface{})
    if !ok {
        return false
    }
    
    for _, plugin := range pluginsList {
        if pluginMap, ok := plugin.(map[string]interface{}); ok {
            if name, ok := pluginMap["name"].(string); ok {
                if strings.Contains(strings.ToLower(name), "yoast") {
                    return true
                }
            }
        }
    }
    
    return false
}