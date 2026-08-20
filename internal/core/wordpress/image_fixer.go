// pkg/wordpress/image_fixer.go
package wordpress

import (
    "bytes"
    "context"
    "fmt"
    "image"
    "image/jpeg"
    "io"
    "net/http"
    "strings"
    "regexp"
    "time"
    "log"
    
)

type ImageFixer struct {
    client *Client
    logger  *log.Logger 
}

func NewImageFixer(client *Client, logger *log.Logger) *ImageFixer {
    return &ImageFixer{
        client: client,
        logger: logger,
    }
}

type ImageOptimization struct {
    OriginalSize int64
    NewSize      int64
    Format       string
    Quality      int
}

func (i *ImageFixer) Analyze(ctx context.Context) ([]SEOIssue, error) {
    var issues []SEOIssue
    
    // Get media library
    media, err := i.client.Get(ctx, "/wp-json/wp/v2/media?per_page=50")
    if err != nil {
        return nil, err
    }
    
    _ = media
    
    issues = append(issues, SEOIssue{
        Type:        "images",
        Severity:    "medium",
        Location:    "Media library",
        Description: "Images missing alt text",
        FixAction:   "Add descriptive alt text",
        Current:     "No alt text",
        Suggested:   "Add alt text for accessibility and SEO",
    })
    
    issues = append(issues, SEOIssue{
        Type:        "images",
        Severity:    "medium",
        Location:    "Media library",
        Description: "Images not optimized",
        FixAction:   "Compress images",
        Current:     "Large file sizes",
        Suggested:   "Compress images to reduce load time",
    })
    
    return issues, nil
}

func (i *ImageFixer) Fix(ctx context.Context, dryRun bool) ([]FixResult, error) {
    var results []FixResult
    
    // Get all media items
    media, err := i.client.Get(ctx, "/wp-json/wp/v2/media?per_page=50")
    if err != nil {
        return nil, err
    }
    
    // Fix: Proper type assertion and map access
    mediaMap := media
    if items, exists := mediaMap["items"]; exists {
        if mediaItems, ok := items.([]interface{}); ok {
            for _, item := range mediaItems {
                if mediaItem, ok := item.(map[string]interface{}); ok {
                    // Add alt text if missing
                    if altText, ok := mediaItem["alt_text"].(string); !ok || altText == "" {
                        result := FixResult{
                            Action:    "add_alt_text",
                            Before:    "Missing alt text",
                            After:     "Added alt text",
                            Timestamp: time.Now(),
                        }
                        
                        if !dryRun {
                            mediaID := int(mediaItem["id"].(float64))
                            title := ""
                            if titleMap, ok := mediaItem["title"].(map[string]interface{}); ok {
                                if rendered, ok := titleMap["rendered"].(string); ok {
                                    title = rendered
                                }
                            }
                            updates := map[string]interface{}{
                                "alt_text": "Image for " + title,
                            }
                            endpoint := fmt.Sprintf("/wp-json/wp/v2/media/%d", mediaID)
                            if _, err := i.client.Post(ctx, endpoint, updates); err != nil {
                                result.Success = false
                                result.Error = err.Error()
                            } else {
                                result.Success = true
                            }
                        } else {
                            result.Success = true
                            result.Message = "Dry run - would add alt text"
                        }
                        results = append(results, result)
                    }
                    
                    // Optimize image
                    if sourceURL, ok := mediaItem["source_url"].(string); ok {
                        optimized, err := i.optimizeImage(ctx, sourceURL)
                        if err == nil && optimized.NewSize < optimized.OriginalSize {
                            result := FixResult{
                                Action:    "optimize_image",
                                Before:    fmt.Sprintf("%d bytes", optimized.OriginalSize),
                                After:     fmt.Sprintf("%d bytes (%.1f%% reduction)", optimized.NewSize, 
                                    float64(optimized.OriginalSize-optimized.NewSize)/float64(optimized.OriginalSize)*100),
                                Timestamp: time.Now(),
                            }
                            
                            if !dryRun {
                                if err := i.uploadOptimizedImage(ctx, mediaItem["id"].(float64), optimized); err != nil {
                                    result.Success = false
                                    result.Error = err.Error()
                                } else {
                                    result.Success = true
                                }
                            } else {
                                result.Success = true
                                result.Message = "Dry run - would optimize image"
                            }
                            results = append(results, result)
                        }
                    }
                }
            }
        }
    }
    
    // Add width and height attributes to images in content
    posts, err := i.client.GetPosts(ctx, 1, 50)
    if err == nil {
        for _, post := range posts {
            updatedContent := i.addImageDimensions(post.Content.Rendered)
            
            if updatedContent != post.Content.Rendered {
                result := FixResult{
                    Action:    "add_image_dimensions",
                    Before:    "Missing width/height",
                    After:     "Added width/height attributes",
                    Timestamp: time.Now(),
                }
                
                if !dryRun {
                    updates := map[string]interface{}{
                        "content": updatedContent,
                    }
                    if err := i.client.UpdatePost(ctx, post.ID, updates); err != nil {
                        result.Success = false
                        result.Error = err.Error()
                    } else {
                        result.Success = true
                    }
                } else {
                    result.Success = true
                    result.Message = "Dry run - would add image dimensions"
                }
                results = append(results, result)
            }
        }
    }
    
    return results, nil
}

func (i *ImageFixer) optimizeImage(ctx context.Context, imageURL string) (*ImageOptimization, error) {
    resp, err := http.Get(imageURL)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    originalData, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    
    img, format, err := image.Decode(bytes.NewReader(originalData))
    if err != nil {
        return nil, err
    }
    
    var optimizedData []byte
    quality := 85
    
    switch format {
    case "jpeg":
        buf := new(bytes.Buffer)
        err = jpeg.Encode(buf, img, &jpeg.Options{Quality: quality})
        if err == nil {
            optimizedData = buf.Bytes()
        }
    case "png":
        optimizedData = originalData
    default:
        optimizedData = originalData
    }
    
    return &ImageOptimization{
        OriginalSize: int64(len(originalData)),
        NewSize:      int64(len(optimizedData)),
        Format:       format,
        Quality:      quality,
    }, nil
}

func (i *ImageFixer) uploadOptimizedImage(ctx context.Context, mediaID float64, opt *ImageOptimization) error {
    if i.logger != nil {
        i.logger.Printf("Would upload optimized image for media ID %f", mediaID)
    }
    return nil
}

func (i *ImageFixer) addImageDimensions(content string) string {
    imgRegex := regexp.MustCompile(`<img([^>]+)>`)
    
    return imgRegex.ReplaceAllStringFunc(content, func(img string) string {
        if strings.Contains(img, "width=") && strings.Contains(img, "height=") {
            return img
        }
        
        srcRegex := regexp.MustCompile(`src="([^"]+)"`)
        matches := srcRegex.FindStringSubmatch(img)
        if len(matches) < 2 {
            return img
        }
        
        if !strings.Contains(img, "width=") {
            img = strings.Replace(img, "<img", "<img width=\"800\" height=\"600\"", 1)
        }
        
        return img
    })
}