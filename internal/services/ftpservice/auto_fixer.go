package ftpservice

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
)

// AutoFixer handles applying SEO fixes to HTML files
type AutoFixer struct {
	logger  *log.Logger
	ftpSvc  *FTPService
	backups []string
	fixes   []string
}

// FixResult represents the result of applying fixes
type FixResult struct {
	File         string   `json:"file"`
	FixesApplied []string `json:"fixes_applied"`
	BackupPath   string   `json:"backup_path"`
	Success      bool     `json:"success"`
	Error        string   `json:"error,omitempty"`
}

// NewAutoFixer creates a new auto-fixer
func NewAutoFixer(ftpSvc *FTPService, logger *log.Logger) *AutoFixer {
	return &AutoFixer{
		logger:  logger,
		ftpSvc:  ftpSvc,
		backups: []string{},
		fixes:   []string{},
	}
}

// FixFile applies all SEO fixes to a single file
func (f *AutoFixer) FixFile(path string) (*FixResult, error) {
	f.logger.Printf("🔧 Fixing file: %s", path)

	// Create backup first
	backupPath, err := f.ftpSvc.CreateBackup(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}
	f.backups = append(f.backups, backupPath)

	// Read file content
	content, err := f.ftpSvc.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Apply all fixes
	fixedContent, fixesApplied := f.applyAllFixes(content, path)

	// Write fixed content
	err = f.ftpSvc.WriteFile(path, fixedContent)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	f.fixes = append(f.fixes, fixesApplied...)

	result := &FixResult{
		File:         path,
		FixesApplied: fixesApplied,
		BackupPath:   backupPath,
		Success:      true,
	}

	f.logger.Printf("✅ Fixed %s: %d fixes applied", path, len(fixesApplied))
	return result, nil
}

// applyAllFixes applies all SEO fixes to HTML content
func (f *AutoFixer) applyAllFixes(content, path string) (string, []string) {
	modified := content
	fixesApplied := []string{}

	// ========== FIX 1: Add Title Tag ==========
	if !strings.Contains(modified, "<title>") {
		title := f.generateTitle(modified, path)
		modified = strings.Replace(modified, "<head>", "<head>\n    <title>"+title+"</title>", 1)
		fixesApplied = append(fixesApplied, "✅ Added title tag: "+title)
	}

	// ========== FIX 2: Add Meta Description ==========
	if !strings.Contains(modified, `name="description"`) {
		desc := f.generateMetaDescription(modified)
		modified = strings.Replace(modified, "</head>", `    <meta name="description" content="`+desc+`">\n</head>`, 1)
		fixesApplied = append(fixesApplied, "✅ Added meta description")
	}

	// ========== FIX 3: Add Viewport ==========
	if !strings.Contains(modified, "viewport") {
		modified = strings.Replace(modified, "</head>", `    <meta name="viewport" content="width=device-width, initial-scale=1.0">\n</head>`, 1)
		fixesApplied = append(fixesApplied, "✅ Added viewport meta tag")
	}

	// ========== FIX 4: Ensure H1 Tag ==========
	if !strings.Contains(modified, "<h1>") {
		title := f.extractTitle(modified)
		if title == "" {
			title = "Welcome to Our Website"
		}
		modified = strings.Replace(modified, "<body>", "<body>\n    <h1>"+title+"</h1>", 1)
		fixesApplied = append(fixesApplied, "✅ Added H1 heading")
	}

	// ========== FIX 5: Add Alt Text to Images ==========
	imgRegex := regexp.MustCompile(`<img(?!.*alt=)(.*?)>`)
	imgMatches := imgRegex.FindAllStringSubmatch(modified, -1)
	altCount := 0
	for _, match := range imgMatches {
		if len(match) > 0 {
			altText := f.generateAltText(path, altCount)
			modified = strings.Replace(modified, match[0], `<img`+match[1]+` alt="`+altText+`">`, 1)
			altCount++
		}
	}
	if altCount > 0 {
		fixesApplied = append(fixesApplied, fmt.Sprintf("✅ Added alt text to %d images", altCount))
	}

	// ========== FIX 6: Add Canonical URL ==========
	if !strings.Contains(modified, `rel="canonical"`) {
		canonical := `<link rel="canonical" href="https://` + f.extractDomain(path) + `">`
		modified = strings.Replace(modified, "</head>", `    `+canonical+`\n</head>`, 1)
		fixesApplied = append(fixesApplied, "✅ Added canonical URL")
	}

	// ========== FIX 7: Add Open Graph Tags ==========
	if !strings.Contains(modified, `property="og:title"`) {
		ogTags := `
    <meta property="og:title" content="` + f.extractTitle(modified) + `">
    <meta property="og:type" content="website">
    <meta property="og:url" content="https://` + f.extractDomain(path) + `">
    <meta property="og:description" content="` + f.generateMetaDescription(modified) + `">
    <meta property="og:image" content="https://` + f.extractDomain(path) + `/og-image.jpg">`
		modified = strings.Replace(modified, "</head>", ogTags+`\n</head>`, 1)
		fixesApplied = append(fixesApplied, "✅ Added Open Graph tags")
	}

	// ========== FIX 8: Add Twitter Card Tags ==========
	if !strings.Contains(modified, `name="twitter:card"`) {
		twitterTags := `
    <meta name="twitter:card" content="summary_large_image">
    <meta name="twitter:title" content="` + f.extractTitle(modified) + `">
    <meta name="twitter:description" content="` + f.generateMetaDescription(modified) + `">
    <meta name="twitter:image" content="https://` + f.extractDomain(path) + `/twitter-image.jpg">`
		modified = strings.Replace(modified, "</head>", twitterTags+`\n</head>`, 1)
		fixesApplied = append(fixesApplied, "✅ Added Twitter Card tags")
	}

	// ========== FIX 9: Add FAQ Schema (AEO) ==========
	if !strings.Contains(modified, "FAQPage") {
		faqSchema := f.generateFAQSchema()
		modified = strings.Replace(modified, "</head>", faqSchema+`\n</head>`, 1)
		fixesApplied = append(fixesApplied, "✅ Added FAQ schema (AEO)")
	}

	// ========== FIX 10: Add Organization Schema (GEO) ==========
	if !strings.Contains(modified, "Organization") {
		orgSchema := f.generateOrganizationSchema()
		modified = strings.Replace(modified, "</head>", orgSchema+`\n</head>`, 1)
		fixesApplied = append(fixesApplied, "✅ Added Organization schema (GEO)")
	}

	// ========== FIX 11: Add Semantic HTML ==========
	if !strings.Contains(modified, "<article>") && !strings.Contains(modified, "<section>") {
		modified = f.addSemanticHTML(modified)
		fixesApplied = append(fixesApplied, "✅ Added semantic HTML elements")
	}

	return modified, fixesApplied
}

// generateTitle generates a title from content
func (f *AutoFixer) generateTitle(content, path string) string {
	// Try to extract from H1
	h1Regex := regexp.MustCompile(`<h1[^>]*>(.*?)</h1>`)
	matches := h1Regex.FindStringSubmatch(content)
	if len(matches) > 1 && matches[1] != "" {
		return f.cleanText(matches[1])
	}

	// Try to extract from content
	contentRegex := regexp.MustCompile(`<p[^>]*>(.*?)</p>`)
	matches = contentRegex.FindStringSubmatch(content)
	if len(matches) > 1 && matches[1] != "" {
		words := strings.Fields(matches[1])
		if len(words) > 10 {
			return f.cleanText(strings.Join(words[:10], " "))
		}
		return f.cleanText(matches[1])
	}

	// Fallback to filename
	filename := filepath.Base(path)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	filename = strings.ReplaceAll(filename, "-", " ")
	filename = strings.ReplaceAll(filename, "_", " ")
	return f.cleanText(filename)
}

// generateMetaDescription generates a meta description
func (f *AutoFixer) generateMetaDescription(content string) string {
	// Remove HTML tags
	text := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(content, " ")
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// Get first 150-160 characters
	words := strings.Fields(text)
	if len(words) > 20 {
		return f.cleanText(strings.Join(words[:20], " "))
	}
	if len(text) > 160 {
		return f.cleanText(text[:160])
	}
	return f.cleanText(text)
}

// extractTitle extracts title from HTML
func (f *AutoFixer) extractTitle(content string) string {
	titleRegex := regexp.MustCompile(`<title[^>]*>(.*?)</title>`)
	matches := titleRegex.FindStringSubmatch(content)
	if len(matches) > 1 && matches[1] != "" {
		return f.cleanText(matches[1])
	}
	return ""
}

// extractDomain extracts domain from path
func (f *AutoFixer) extractDomain(path string) string {
	// This is a simplified version - in production, use the actual domain
	return "example.com"
}

// generateAltText generates alt text for images
func (f *AutoFixer) generateAltText(path string, index int) string {
	return fmt.Sprintf("Image %d - %s", index+1, f.extractDomain(path))
}

// generateFAQSchema generates FAQ schema for AEO
func (f *AutoFixer) generateFAQSchema() string {
	return `
    <script type="application/ld+json">
    {
        "@context": "https://schema.org",
        "@type": "FAQPage",
        "mainEntity": [
            {
                "@type": "Question",
                "name": "What is SEOSPS?",
                "acceptedAnswer": {
                    "@type": "Answer",
                    "text": "SEOSPS is an AI-powered SEO automation tool that optimizes websites for SEO, AEO, GEO, and AIO in one platform."
                }
            },
            {
                "@type": "Question",
                "name": "How does SEOSPS help with SEO?",
                "acceptedAnswer": {
                    "@type": "Answer",
                    "text": "SEOSPS analyzes your website, identifies SEO issues, and automatically fixes them. It also optimizes for AI search engines."
                }
            }
        ]
    }
    </script>`
}

// generateOrganizationSchema generates Organization schema for GEO
func (f *AutoFixer) generateOrganizationSchema() string {
	return `
    <script type="application/ld+json">
    {
        "@context": "https://schema.org",
        "@type": "Organization",
        "name": "SEOSPS",
        "description": "AI-powered SEO automation tool",
        "url": "https://seosps.com",
        "logo": "https://seosps.com/logo.png"
    }
    </script>`
}

// addSemanticHTML adds semantic HTML elements
func (f *AutoFixer) addSemanticHTML(content string) string {
	modified := content

	// Replace main content div with semantic elements
	modified = strings.Replace(modified, `<div class="content">`, `<article>`, -1)
	modified = strings.Replace(modified, `<div class="section">`, `<section>`, -1)
	modified = strings.Replace(modified, `<div class="header">`, `<header>`, -1)
	modified = strings.Replace(modified, `<div class="footer">`, `<footer>`, -1)

	// Add semantic wrappers if missing
	if !strings.Contains(modified, "<main>") {
		modified = strings.Replace(modified, "<body>", "<body>\n    <main>", 1)
		modified = strings.Replace(modified, "</body>", "</main>\n</body>", 1)
	}

	return modified
}

// cleanText cleans text by removing extra whitespace
func (f *AutoFixer) cleanText(text string) string {
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// Rollback rolls back all changes
func (f *AutoFixer) Rollback() error {
	f.logger.Println("🔄 Rolling back changes...")

	for _, backup := range f.backups {
		err := f.ftpSvc.RestoreBackup(backup)
		if err != nil {
			f.logger.Printf("Warning: Failed to restore backup %s: %v", backup, err)
		}
	}

	f.logger.Println("✅ Rollback completed")
	return nil
}

// GetFixesApplied returns all fixes applied
func (f *AutoFixer) GetFixesApplied() []string {
	return f.fixes
}