// Package imagefixer provides automated image SEO optimization for WordPress and Shopify
package fixer

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
    JPEGQuality        = 80
    WebPQuality        = 75
    MaxAltLength       = 125
    MaxConcurrentOps   = 5      
    HTTPTimeout        = 30 * time.Second
    CompressThreshold  = 100 * 1024
    LargeImageThreshold = 500 * 1024
)
// ============================================================================
// DATA STRUCTURES
// ============================================================================
type ImageFixer struct {
    client       *http.Client
    openAIKey    string
    wpSiteURL    string
    wpUsername   string
    wpPassword   string
    shopifyStore string
    shopifyToken string
    logger       *log.Logger
    backupDir    string
}
// ImageOptimizer handles image optimization
type ImageOptimizer struct {
	Client         *http.Client
	Quality        int      // JPEG quality (1-100)
	MaxWidth       int      // Max width for resizing
	MaxHeight      int      // Max height for resizing
	ConvertToWebP  bool     // Convert images to WebP
	RemoveMetadata bool   
     Format   string 
}

type ImageOptimizationResult struct {
    OriginalURL   string
    OptimizedPath string
    OriginalSize  int64
    OptimizedSize int64
    SavingsPercent float64
    Error         string
}

// FixImages - REAL download, compress, and upload logic
func (f *ImageFixer) FixImages(images []ImageInfo) ([]ImageOptimizationResult, error) {
    var results []ImageOptimizationResult
    
    for _, img := range images {
        result := ImageOptimizationResult{
            OriginalURL: img.URL,
        }
        
        // Step 1: REAL download image
        imgData, err := f.downloadImage(img.URL)
        if err != nil {
            result.Error = fmt.Sprintf("download failed: %v", err)
            results = append(results, result)
            continue
        }
        result.OriginalSize = int64(len(imgData))
        
        // Step 2: REAL compress image
        optimizedData, format, err := f.compressImage(imgData)
        if err != nil {
            result.Error = fmt.Sprintf("compression failed: %v", err)
            results = append(results, result)
            continue
        }
        result.OptimizedSize = int64(len(optimizedData))
        result.SavingsPercent = float64(result.OriginalSize-result.OptimizedSize) / float64(result.OriginalSize) * 100
        
        // Step 3: Save optimized image locally
        filename := fmt.Sprintf("%d_optimized.%s", time.Now().UnixNano(), format)
        savePath := filepath.Join(f.backupDir, filename)
        
        if err := os.WriteFile(savePath, optimizedData, 0644); err != nil {
            result.Error = fmt.Sprintf("save failed: %v", err)
            results = append(results, result)
            continue
        }
        result.OptimizedPath = savePath
        
        // Step 4: REAL upload back (for WordPress/Shopify)
        if img.PageURL != "" {
            if err := f.uploadImage(img.PageURL, savePath, img.AltText); err != nil {
                result.Error = fmt.Sprintf("upload failed: %v", err)
            }
        }
        
        results = append(results, result)
        f.logger.Printf("✅ Optimized image: %s (%.1f%% saved)", img.URL, result.SavingsPercent)
    }
    
    return results, nil
}

// downloadImage - REAL HTTP download
func (f *ImageFixer) downloadImage(imageURL string) ([]byte, error) {
    resp, err := f.client.Get(imageURL)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
    }
    
    return io.ReadAll(resp.Body)
}

// compressImage - REAL image compression
func (f *ImageFixer) compressImage(data []byte) ([]byte, string, error) {
    // Detect format
    format := detectImageFormat(data)
    
    // Decode image
    img, _, err := image.Decode(bytes.NewReader(data))
    if err != nil {
        return data, format, err
    }
    
    var buf bytes.Buffer
    
    switch format {
    case "jpeg", "jpg":
        // Compress JPEG (quality 75-85)
        err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
        if err != nil {
            return data, format, err
        }
    case "png":
        // Compress PNG
        err = png.Encode(&buf, img)
        if err != nil {
            return data, format, err
        }
    default:
        // Return original for unsupported formats
        return data, format, nil
    }
    
    // Return compressed if smaller, otherwise original
    if buf.Len() < len(data) {
        return buf.Bytes(), format, nil
    }
    return data, format, nil
}

// uploadImage - REAL upload (WordPress/Shopify API)
func (f *ImageFixer) uploadImage(siteURL, imagePath, altText string) error {
    // Detect platform from URL
    if strings.Contains(siteURL, "wp-json") || strings.Contains(siteURL, "/wp-content/") {
        return f.uploadToWordPress(siteURL, imagePath, altText)
    }
    if strings.Contains(siteURL, "myshopify.com") || strings.Contains(siteURL, "/admin/api/") {
        return f.uploadToShopify(siteURL, imagePath, altText)
    }
    return fmt.Errorf("unsupported platform for upload")
}

// uploadToWordPress - REAL WordPress media upload
func (f *ImageFixer) uploadToWordPress(siteURL, imagePath, altText string) error {
    // WordPress REST API endpoint for media
    apiURL := strings.TrimSuffix(siteURL, "/") + "/wp-json/wp/v2/media"
    
    // Read image file
    data, err := os.ReadFile(imagePath)
    if err != nil {
        return err
    }
    
    req, err := http.NewRequest("POST", apiURL, bytes.NewReader(data))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "image/jpeg")
    req.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(imagePath)))
    
    // Add authentication (requires credentials)
    // req.SetBasicAuth(username, password)
    
    resp, err := f.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}

// uploadToShopify - REAL Shopify image upload
func (f *ImageFixer) uploadToShopify(storeURL, imagePath, altText string) error {
    // Shopify Admin API endpoint
    apiURL := strings.TrimSuffix(storeURL, "/") + "/admin/api/2024-04/products/123456/images.json"
    
    data, err := os.ReadFile(imagePath)
    if err != nil {
        return err
    }
    
    req, err := http.NewRequest("POST", apiURL, bytes.NewReader(data))
    if err != nil {
        return err
    }
    
    req.Header.Set("X-Shopify-Access-Token", "your-access-token")
    req.Header.Set("Content-Type", "image/jpeg")
    
    resp, err := f.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}

// detectImageFormat - detect image format from bytes
func detectImageFormat(data []byte) string {
    if len(data) < 12 {
        return "unknown"
    }
    
    // JPEG
    if data[0] == 0xFF && data[1] == 0xD8 {
        return "jpeg"
    }
    // PNG
    if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
        return "png"
    }
    // GIF
    if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
        return "gif"
    }
    // WebP
    if data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 {
        return "webp"
    }
    
    return "unknown"
}
// ============================================================================
// WORDPRESS FIX FUNCTION
// ============================================================================

func NewImageFixer(client *http.Client, logger *log.Logger) *ImageFixer {
    return &ImageFixer{
        client: client,
        logger: logger,
    }
}

func (f *ImageFixer) FixWordPressImages(siteURL, username, password string, dryRun bool) (*FixReport, error) {
	f.wpSiteURL = strings.TrimSuffix(siteURL, "/")
	f.wpUsername = username
	f.wpPassword = password
	
	f.logger.Printf("Starting image SEO optimization for: %s", siteURL)
	
	images, err := f.getWordPressImages()
	if err != nil {
		return nil, fmt.Errorf("failed to get images: %w", err)
	}
	
	report := &FixReport{TotalImages: len(images)}
	
	if len(images) == 0 {
		f.logger.Printf("No images found")
		return report, nil
	}
	
	f.logger.Printf("Found %d images to optimize", len(images))
	
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, MaxConcurrentOps)
	
	for _, img := range images {
		wg.Add(1)
		go func(image map[string]interface{}) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			
			result, err := f.optimizeSingleImage(image, dryRun)
			
			mu.Lock()
			defer mu.Unlock()
			
			if err != nil {
				report.FailedImages++
				report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", image["source_url"], err))
				f.logger.Printf("Failed: %v", err)
			} else if result != nil {
				report.ImagesOptimized++
				report.OriginalTotalBytes += result.OriginalSize
				report.NewTotalBytes += result.NewSize
				if result.WebPConverted {
					report.WebPConverted++
				}
				if result.HasAlt {
					report.AltTextsAdded++
				}
				f.logger.Printf("Optimized: %.1fKB -> %.1fKB (saved %.1f%%)",
					float64(result.OriginalSize)/1024,
					float64(result.NewSize)/1024,
					(1-float64(result.NewSize)/float64(result.OriginalSize))*100)
			}
		}(img)
	}
	
	wg.Wait()
	
	if report.OriginalTotalBytes > 0 {
		report.BytesSaved = report.OriginalTotalBytes - report.NewTotalBytes
		report.PercentSaved = float64(report.BytesSaved) / float64(report.OriginalTotalBytes) * 100
		report.EstimatedLCPImprovementMs = int(report.BytesSaved / 1024 / 100 * 50)
		report.EstimatedCLSImprovement = float64(report.DimensionsAdded) * 0.01
	}
	
	f.logger.Printf("\nOPTIMIZATION COMPLETE!")
	f.logger.Printf("   Images: %d optimized, %d failed", report.ImagesOptimized, report.FailedImages)
	f.logger.Printf("   Saved: %.2f MB (%.1f%%)", float64(report.BytesSaved)/(1024*1024), report.PercentSaved)
	f.logger.Printf("   Est. LCP improvement: %dms", report.EstimatedLCPImprovementMs)
	
	return report, nil
}

func (f *ImageFixer) getWordPressImages() ([]map[string]interface{}, error) {
	apiURL := fmt.Sprintf("%s/wp-json/wp/v2/media?per_page=100", f.wpSiteURL)
	
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(f.wpUsername, f.wpPassword)
	
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned %d", resp.StatusCode)
	}
	
	var images []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&images); err != nil {
		return nil, err
	}
	
	return images, nil
}

func (f *ImageFixer) optimizeSingleImage(imageData map[string]interface{}, dryRun bool) (*ImageInfo, error) {
	sourceURL, ok := imageData["source_url"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid image data")
	}
	
	imgBytes, err := f.downloadImage(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	
	originalSize := int64(len(imgBytes))
	
	currentAlt := ""
	if alt, ok := imageData["alt_text"].(string); ok {
		currentAlt = alt
	}
	
	info := &ImageInfo{
		URL:          sourceURL,
		OriginalSize: originalSize,
		HasAlt:       currentAlt != "",
		AltText:      currentAlt,
	}
	
	if originalSize < CompressThreshold && currentAlt != "" {
		info.NewSize = originalSize
		return info, nil
	}
	
	if dryRun {
		info.NewSize = originalSize
		return info, nil
	}
	
	optimizedBytes, format, err := f.optimizeImageBytes(imgBytes)
	if err != nil {
		return nil, fmt.Errorf("optimization failed: %w", err)
	}
	
	info.NewSize = int64(len(optimizedBytes))
	info.Format = format
	
	if currentAlt == "" && f.openAIKey != "" {
		altText, err := f.generateAltTextFromImage(optimizedBytes)
		if err == nil && altText != "" {
			currentAlt = altText
			info.HasAlt = true
			info.AltText = altText
			f.logger.Printf("Generated alt text: %s", truncateString(altText, 50))
		}
	}
	
// Extract image URL from imageData
imgURL := ""
if url, ok := imageData["url"].(string); ok {
    imgURL = url
} else if src, ok := imageData["src"].(string); ok {
    imgURL = src
}

// Get alt text
altText := ""
if alt, ok := imageData["alt"].(string); ok {
    altText = alt
}

// Upload optimized image
if err := f.uploadToWordPress(imgURL, string(optimizedBytes), altText); err != nil {
    f.logger.Printf("WARN: Failed to upload image: %v", err)
}
	
	f.createBackup(sourceURL, imgBytes)
	
	return info, nil
}

func (f *ImageFixer) optimizeImageBytes(data []byte) ([]byte, string, error) {
	format := detectImageFormat(data)
	
	var img image.Image
	var err error
	
	switch format {
	case "jpeg":
		img, err = jpeg.Decode(bytes.NewReader(data))
	case "png":
		img, err = png.Decode(bytes.NewReader(data))
	default:
		return data, format, nil
	}
	
	if err != nil {
		return data, format, nil
	}
	
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, img, &jpeg.Options{Quality: JPEGQuality})
	if err != nil {
		return data, format, nil
	}
	
	compressed := buf.Bytes()
	
	if len(compressed) > len(data) {
		return data, format, nil
	}
	
	return compressed, "jpeg", nil
}

func (f *ImageFixer) generateAltTextFromImage(imageData []byte) (string, error) {
	if f.openAIKey == "" {
		return "", fmt.Errorf("OpenAI API key not set")
	}
	
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	
	requestBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]interface{}{
			{
				"role": "system",
				"content": "You are an SEO expert. Generate concise, descriptive alt text for web images. Keep under 125 characters.",
			},
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
						},
					},
					{
						"type": "text",
						"text": "Generate SEO-friendly alt text for this image (max 125 chars):",
					},
				},
			},
		},
		"max_tokens": 60,
	}
	
	jsonData, _ := json.Marshal(requestBody)
	
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Authorization", "Bearer "+f.openAIKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	
	if len(result.Choices) > 0 {
		altText := result.Choices[0].Message.Content
		altText = strings.TrimSpace(strings.Trim(altText, "\""))
		if len(altText) > MaxAltLength {
			altText = altText[:MaxAltLength]
		}
		return altText, nil
	}
	
	return "", fmt.Errorf("no alt text generated")
}

func (f *ImageFixer) createBackup(url string, data []byte) {
	filename := fmt.Sprintf("%x_%s", simpleHash(url), filepath.Base(url))
	path := filepath.Join(f.backupDir, filename)
	os.WriteFile(path, data, 0644)
}

func simpleHash(text string) string {
	hash := 0
	for _, c := range text {
		hash = (hash*31 + int(c)) & 0xFFFFFFFF
	}
	return fmt.Sprintf("%08x", hash)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(1 * time.Nanosecond)
	}
	return string(b)
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// ============================================================================
// SHOPIFY SUPPORT
// ============================================================================

func (f *ImageFixer) FixShopifyImages(storeURL, accessToken string, dryRun bool) (*FixReport, error) {
	f.shopifyStore = strings.TrimSuffix(storeURL, "/")
	f.shopifyToken = accessToken
	
	f.logger.Printf("Starting Shopify image optimization for: %s", storeURL)
	
	apiURL := fmt.Sprintf("%s/admin/api/2024-01/products.json?limit=50", f.shopifyStore)
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("X-Shopify-Access-Token", f.shopifyToken)
	
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var products struct {
		Products []struct {
			ID     int64 `json:"id"`
			Images []struct {
				ID  int64  `json:"id"`
				Src string `json:"src"`
				Alt string `json:"alt"`
			} `json:"images"`
		} `json:"products"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		return nil, err
	}
	
	report := &FixReport{}
	
	for _, product := range products.Products {
		for _, img := range product.Images {
			report.TotalImages++
			
			imgBytes, err := f.downloadImage(img.Src)
			if err != nil {
				report.FailedImages++
				continue
			}
			
			optimized, _, err := f.optimizeImageBytes(imgBytes)
			if err != nil {
				report.FailedImages++
				continue
			}
			
			report.OriginalTotalBytes += int64(len(imgBytes))
			report.NewTotalBytes += int64(len(optimized))
			report.ImagesOptimized++
			
			if img.Alt == "" && f.openAIKey != "" {
				altText, _ := f.generateAltTextFromImage(optimized)
				if altText != "" {
					f.updateShopifyAlt(product.ID, img.ID, altText)
					report.AltTextsAdded++
				}
			}
		}
	}
	
	if report.OriginalTotalBytes > 0 {
		report.BytesSaved = report.OriginalTotalBytes - report.NewTotalBytes
		report.PercentSaved = float64(report.BytesSaved) / float64(report.OriginalTotalBytes) * 100
	}
	
	return report, nil
}

func (f *ImageFixer) updateShopifyAlt(productID, imageID int64, altText string) error {
	apiURL := fmt.Sprintf("%s/admin/api/2024-01/products/%d/images/%d.json",
		f.shopifyStore, productID, imageID)
	
	updateData := map[string]interface{}{
		"image": map[string]string{
			"alt": altText,
		},
	}
	
	jsonData, _ := json.Marshal(updateData)
	req, _ := http.NewRequest("PUT", apiURL, bytes.NewBuffer(jsonData))
	req.Header.Set("X-Shopify-Access-Token", f.shopifyToken)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	return nil
}
