// pkg/wordpress/content_fixer.go
package wordpress

import (
    "context"
    "regexp"
    "strings"
     "fmt"
    "time"
    "log"
)

type ContentFixer struct {
    client *Client
    logger  *log.Logger 
}

func NewContentFixer(client *Client, logger *log.Logger) *ContentFixer {
    return &ContentFixer{
        client: client,
        logger: logger,
    }
}

func (c *ContentFixer) Analyze(ctx context.Context) ([]SEOIssue, error) {
    var issues []SEOIssue
    
    posts, err := c.client.GetPosts(ctx, 1, 100)
    if err != nil {
        return nil, err
    }
    
    for _, post := range posts {
        // Check heading structure
        h1Count := strings.Count(post.Content.Rendered, "<h1")
        if h1Count == 0 {
            issues = append(issues, SEOIssue{
                Type:        "content",
                Severity:    "high",
                Location:    post.Link,
                Description: "Missing H1 heading",
                FixAction:   "Add H1 heading",
                Current:     "No H1 tag",
                Suggested:   "Add one H1 with primary keyword",
            })
        } else if h1Count > 1 {
            issues = append(issues, SEOIssue{
                Type:        "content",
                Severity:    "high",
                Location:    post.Link,
                Description: "Multiple H1 headings",
                FixAction:   "Reduce to single H1",
                Current:     fmt.Sprintf("%d H1 tags", h1Count),
                Suggested:   "One H1 tag",
            })
        }
        
        // Check for thin content
        wordCount := len(strings.Fields(post.Content.Rendered))
        if wordCount < 300 {
            issues = append(issues, SEOIssue{
                Type:        "content",
                Severity:    "high",
                Location:    post.Link,
                Description: "Thin content",
                FixAction:   "Expand content",
                Current:     fmt.Sprintf("%d words", wordCount),
                Suggested:   "300+ words",
            })
        }
        
        // Check for image alt text
        imgTags := regexp.MustCompile(`<img[^>]+>`)
        images := imgTags.FindAllString(post.Content.Rendered, -1)
        for _, img := range images {
            if !strings.Contains(img, "alt=") {
                issues = append(issues, SEOIssue{
                    Type:        "content",
                    Severity:    "medium",
                    Location:    post.Link,
                    Description: "Image missing alt text",
                    FixAction:   "Add alt text to images",
                    Current:     "No alt attribute",
                    Suggested:   "Descriptive alt text",
                })
                break
            }
        }
    }
    
    return issues, nil
}

func (c *ContentFixer) Fix(ctx context.Context, dryRun bool) ([]FixResult, error) {
    var results []FixResult
    
    posts, err := c.client.GetPosts(ctx, 1, 100)
    if err != nil {
        return nil, err
    }
    
    for _, post := range posts {
        updatedContent := post.Content.Rendered
        
        // Fix heading structure
        h1Count := strings.Count(updatedContent, "<h1")
        if h1Count == 0 {
            // Add H1 from title if no H1 exists
            h1Tag := "<h1>" + post.Title.Rendered + "</h1>"
            updatedContent = h1Tag + updatedContent
            
            result := FixResult{
                Action:    "add_h1",
                Before:    "No H1",
                After:     "Added H1 heading",
                Timestamp: time.Now(),
            }
            
            if !dryRun {
                result.Success = true
            } else {
                result.Success = true
                result.Message = "Dry run - would add H1"
            }
            results = append(results, result)
        } else if h1Count > 1 {
            // Keep only first H1, convert others to H2
            h1Regex := regexp.MustCompile(`<h1[^>]*>(.*?)</h1>`)
            matches := h1Regex.FindAllStringSubmatch(updatedContent, -1)
            if len(matches) > 1 {
                updatedContent = h1Regex.ReplaceAllStringFunc(updatedContent, func(match string) string {
                    if matches[0][0] == match {
                        return match // Keep first H1
                    }
                    return "<h2>" + matches[1][1] + "</h2>" // Convert others to H2
                })
            }
            
            result := FixResult{
                Action:    "fix_multiple_h1",
                Before:    fmt.Sprintf("%d H1 tags", h1Count),
                After:     "1 H1 tag",
                Timestamp: time.Now(),
            }
            
            if !dryRun {
                result.Success = true
            } else {
                result.Success = true
                result.Message = "Dry run - would fix H1 structure"
            }
            results = append(results, result)
        }
        
        // Fix image alt text
        imgRegex := regexp.MustCompile(`<img([^>]+)>`)
        updatedContent = imgRegex.ReplaceAllStringFunc(updatedContent, func(img string) string {
            if strings.Contains(img, "alt=") {
                return img
            }
            // Add alt text based on image filename or post context
            altText := "Image for " + post.Title.Rendered
            return strings.Replace(img, "<img", fmt.Sprintf(`<img alt="%s"`, altText), 1)
        })
        
        if updatedContent != post.Content.Rendered {
            result := FixResult{
                Action:    "add_image_alt_text",
                Before:    "Missing alt text",
                After:     "Added alt text",
                Timestamp: time.Now(),
            }
            
            if !dryRun {
                updates := map[string]interface{}{
                    "content": updatedContent,
                }
                if err := c.client.UpdatePost(ctx, post.ID, updates); err != nil {
                    result.Success = false
                    result.Error = err.Error()
                } else {
                    result.Success = true
                }
            } else {
                result.Success = true
                result.Message = "Dry run - would add image alt text"
            }
            results = append(results, result)
        }
    }
    
    return results, nil
}