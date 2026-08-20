// pkg/wordpress/meta_fixer.go
package wordpress

import (
    "context"
    "fmt"
    "time"
    "strings"
    "log"
)

type MetaFixer struct {
    client *Client
    logger  *log.Logger 
}

func NewMetaFixer(client *Client, logger *log.Logger) *MetaFixer {
    return &MetaFixer{
        client: client,
        logger: logger,
    }
}

func (m *MetaFixer) Analyze(ctx context.Context) ([]SEOIssue, error) {
    var issues []SEOIssue
    
    // Get all posts and pages
    posts, err := m.client.GetPosts(ctx, 1, 100)
    if err != nil {
        return nil, err
    }
    
    for _, post := range posts {
        // Check meta description
        if post.Meta == nil || post.Meta["_yoast_wpseo_metadesc"] == nil {
            issues = append(issues, SEOIssue{
                Type:        "meta",
                Severity:    "high",
                Location:    post.Link,
                Description: "Missing meta description",
                FixAction:   "Generate meta description from content",
                Current:     "",
                Suggested:   "Add a compelling meta description (150-160 characters)",
            })
        } else {
            desc := post.Meta["_yoast_wpseo_metadesc"].(string)
            if len(desc) < 120 || len(desc) > 160 {
                issues = append(issues, SEOIssue{
                    Type:        "meta",
                    Severity:    "medium",
                    Location:    post.Link,
                    Description: "Meta description length issue",
                    FixAction:   "Optimize meta description length",
                    Current:     fmt.Sprintf("%d characters", len(desc)),
                    Suggested:   "150-160 characters",
                })
            }
        }
        
        // Check title length
        if len(post.Title.Rendered) < 30 || len(post.Title.Rendered) > 60 {
            issues = append(issues, SEOIssue{
                Type:        "title",
                Severity:    "medium",
                Location:    post.Link,
                Description: "Title length issue",
                FixAction:   "Optimize title length",
                Current:     fmt.Sprintf("%d characters", len(post.Title.Rendered)),
                Suggested:   "50-60 characters",
            })
        }
    }
    
    return issues, nil
}

func (m *MetaFixer) Fix(ctx context.Context, dryRun bool) ([]FixResult, error) {
    var results []FixResult
    
    posts, err := m.client.GetPosts(ctx, 1, 100)
    if err != nil {
        return nil, err
    }
    
    for _, post := range posts {
        // Fix meta description
        if post.Meta == nil || post.Meta["_yoast_wpseo_metadesc"] == nil {
            generatedDesc := m.generateMetaDescription(post.Content.Rendered)
            if len(generatedDesc) > 160 {
                generatedDesc = generatedDesc[:157] + "..."
            }
            
            result := FixResult{
                Action:    "add_meta_description",
                Before:    "Missing",
                After:     generatedDesc,
                Timestamp: time.Now(),
            }
            
            if !dryRun {
                meta := &YoastMeta{
                    Description: generatedDesc,
                }
                if err := m.client.UpdateYoastMeta(ctx, post.ID, meta); err != nil {
                    result.Success = false
                    result.Error = err.Error()
                } else {
                    result.Success = true
                }
            } else {
                result.Success = true
                result.Message = "Dry run - would add meta description"
            }
            
            results = append(results, result)
        }
        
        // Fix title
        if len(post.Title.Rendered) < 30 {
            optimizedTitle := m.optimizeTitle(post.Title.Rendered, post.Content.Rendered)
            
            result := FixResult{
                Action:    "optimize_title",
                Before:    post.Title.Rendered,
                After:     optimizedTitle,
                Timestamp: time.Now(),
            }
            
            if !dryRun {
                updates := map[string]interface{}{
                    "title": optimizedTitle,
                }
                if err := m.client.UpdatePost(ctx, post.ID, updates); err != nil {
                    result.Success = false
                    result.Error = err.Error()
                } else {
                    result.Success = true
                }
            } else {
                result.Success = true
                result.Message = "Dry run - would optimize title"
            }
            
            results = append(results, result)
        }
    }
    
    return results, nil
}

func (m *MetaFixer) generateMetaDescription(content string) string {
    // Remove HTML tags
    content = strings.ReplaceAll(content, "<p>", "")
    content = strings.ReplaceAll(content, "</p>", "")
    content = strings.ReplaceAll(content, "<br>", " ")
    
    // Get first 160 characters
    if len(content) > 160 {
        return content[:157] + "..."
    }
    return content
}

func (m *MetaFixer) optimizeTitle(title, content string) string {
    // Add primary keyword if missing
    // For simplicity, just ensure title is at least 50 chars
    if len(title) < 50 {
        words := strings.Split(content, " ")
        if len(words) > 10 {
            title = title + " - " + strings.Join(words[:5], " ")
        }
    }
    
    if len(title) > 60 {
        title = title[:57] + "..."
    }
    
    return title
}