package reporting

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PDFConfig holds configuration for PDF generation
type PDFConfig struct {
	OutputDir   string `json:"output_dir"`
	PageSize    string `json:"page_size"`    // A4, Letter, etc.
	Orientation string `json:"orientation"`   // portrait or landscape
	Margins     string `json:"margins"`       // e.g., "20mm"
}

// PDFGenerator handles PDF generation using wkhtmltopdf
type PDFGenerator struct {
	config   PDFConfig
	wkBinary string
}

// NewPDFGenerator creates a new PDF generator instance
func NewPDFGenerator(config PDFConfig) (*PDFGenerator, error) {
	// Check if wkhtmltopdf is installed
	path, err := exec.LookPath("wkhtmltopdf")
	if err != nil {
		return nil, fmt.Errorf("wkhtmltopdf not found in PATH: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Set default values if not provided
	if config.PageSize == "" {
		config.PageSize = "A4"
	}
	if config.Orientation == "" {
		config.Orientation = "portrait"
	}
	if config.Margins == "" {
		config.Margins = "20mm"
	}

	return &PDFGenerator{
		config:   config,
		wkBinary: path,
	}, nil
}

// GenerateFromHTML converts HTML content to PDF
func (pg *PDFGenerator) GenerateFromHTML(htmlContent string, filename string) (string, error) {
	// Create temporary HTML file
	tmpFile, err := os.CreateTemp("", "seo_report_*.html")
	if err != nil {
		return "", fmt.Errorf("failed to create temp HTML file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write HTML content
	if _, err := tmpFile.WriteString(htmlContent); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed to write HTML content: %w", err)
	}
	tmpFile.Close()

	// Generate PDF from temp file
	return pg.GenerateFromFile(tmpFile.Name(), filename)
}

// GenerateFromFile converts an HTML file to PDF
func (pg *PDFGenerator) GenerateFromFile(htmlPath string, pdfPath string) (string, error) {
	// Validate input file exists
	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		return "", fmt.Errorf("HTML file does not exist: %s", htmlPath)
	}

	// If pdfPath is empty, generate a default filename
	if pdfPath == "" {
		pdfPath = filepath.Join(pg.config.OutputDir,
			fmt.Sprintf("seo_report_%s.pdf",
				time.Now().Format("20060102_150405")))
	}

	// Ensure output directory exists
	pdfDir := filepath.Dir(pdfPath)
	if err := os.MkdirAll(pdfDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create PDF directory: %w", err)
	}

	// Build wkhtmltopdf command
	args := []string{
		"--page-size", pg.config.PageSize,
		"--orientation", pg.config.Orientation,
		"--margin-top", pg.config.Margins,
		"--margin-bottom", pg.config.Margins,
		"--margin-left", pg.config.Margins,
		"--margin-right", pg.config.Margins,
		"--enable-local-file-access",
		"--quiet",
		htmlPath,
		pdfPath,
	}

	// Execute command
	cmd := exec.Command(pg.wkBinary, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to generate PDF: %w\nStderr: %s", err, stderr.String())
	}

	// Verify PDF was created
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return "", fmt.Errorf("PDF file was not created: %s", pdfPath)
	}

	return pdfPath, nil
}

// MergePDFs combines multiple PDF files into one
func (pg *PDFGenerator) MergePDFs(pdfFiles []string, outputPath string) (string, error) {
	if len(pdfFiles) == 0 {
		return "", fmt.Errorf("no PDF files to merge")
	}

	// Validate all input files exist
	for _, file := range pdfFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return "", fmt.Errorf("PDF file does not exist: %s", file)
		}
	}

	// If outputPath is empty, generate a default filename
	if outputPath == "" {
		outputPath = filepath.Join(pg.config.OutputDir,
			fmt.Sprintf("merged_report_%s.pdf",
				time.Now().Format("20060102_150405")))
	}

	// Check if pdfunite is available
	if _, err := exec.LookPath("pdfunite"); err != nil {
		// Fallback to using wkhtmltopdf with a simple HTML wrapper
		return pg.mergeWithWKHTML(pdfFiles, outputPath)
	}

	// Use pdfunite for merging
	args := append([]string{}, pdfFiles...)
	args = append(args, outputPath)
	
	cmd := exec.Command("pdfunite", args...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to merge PDFs: %w", err)
	}

	return outputPath, nil
}

// mergeWithWKHTML provides a fallback merge method using wkhtmltopdf
func (pg *PDFGenerator) mergeWithWKHTML(pdfFiles []string, outputPath string) (string, error) {
	// Create a temporary HTML file that embeds all PDFs
	var html strings.Builder
	html.WriteString(`<!DOCTYPE html>
<html>
<head>
    <style>
        body { margin: 0; padding: 0; }
        object { width: 100%; height: 100vh; }
    </style>
</head>
<body>
`)
	for i, pdf := range pdfFiles {
		// Copy PDF to a location wkhtmltopdf can access
		tempPDF := filepath.Join(pg.config.OutputDir, fmt.Sprintf("temp_%d_%s", i, filepath.Base(pdf)))
		if err := copyFile(pdf, tempPDF); err != nil {
			return "", fmt.Errorf("failed to copy PDF: %w", err)
		}
		defer os.Remove(tempPDF)
		
		html.WriteString(fmt.Sprintf(`    <object data="%s" type="application/pdf"></object>
`, filepath.Base(tempPDF)))
		html.WriteString(`    <div style="page-break-after: always;"></div>
`)
	}
	html.WriteString(`</body>
</html>`)

	// Create temp HTML file
	tmpHTML := filepath.Join(pg.config.OutputDir, "merge_temp_"+time.Now().Format("20060102_150405")+".html")
	if err := os.WriteFile(tmpHTML, []byte(html.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to create temp HTML: %w", err)
	}
	defer os.Remove(tmpHTML)

	// Convert to PDF
	return pg.GenerateFromFile(tmpHTML, outputPath)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// AddCoverPage adds a cover page to an existing PDF
func (pg *PDFGenerator) AddCoverPage(pdfPath string, title string, date string, domain string) (string, error) {
	// Generate cover page HTML
	coverHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0; 
            padding: 0;
            height: 100vh;
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
            text-align: center;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        .cover {
            max-width: 800px;
            padding: 40px;
        }
        h1 { 
            font-size: 48px; 
            margin-bottom: 20px; 
            font-weight: 700;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.2);
        }
        h2 { 
            font-size: 24px; 
            margin-bottom: 40px; 
            opacity: 0.95;
            font-weight: 400;
        }
        .date { 
            font-size: 18px; 
            opacity: 0.9;
            margin-bottom: 30px;
        }
        .domain { 
            font-size: 20px; 
            padding: 15px 30px; 
            background: rgba(255,255,255,0.15); 
            border-radius: 50px;
            display: inline-block;
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255,255,255,0.3);
        }
    </style>
</head>
<body>
    <div class="cover">
        <h1>SEO Audit Report</h1>
        <h2>%s</h2>
        <div class="date">Generated: %s</div>
        <div class="domain">%s</div>
    </div>
</body>
</html>`, title, date, domain)

	// Generate cover page PDF
	coverPath := filepath.Join(pg.config.OutputDir, "cover_"+time.Now().Format("20060102_150405")+".pdf")
	_, err := pg.GenerateFromHTML(coverHTML, coverPath)
	if err != nil {
		return "", fmt.Errorf("failed to generate cover page: %w", err)
	}
	defer os.Remove(coverPath)

	// Merge cover with original PDF
	outputPath := strings.TrimSuffix(pdfPath, ".pdf") + "_with_cover.pdf"
	return pg.MergePDFs([]string{coverPath, pdfPath}, outputPath)
}

// AddTableOfContents generates a TOC for multi-page reports
func (pg *PDFGenerator) AddTableOfContents(pdfPath string) (string, error) {
	// Generate TOC HTML
	tocHTML := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 40px;
            background: white;
        }
        h1 { 
            color: #333; 
            border-bottom: 3px solid #667eea; 
            padding-bottom: 15px;
            margin-bottom: 30px;
            font-size: 32px;
        }
        .toc-list {
            list-style: none;
            padding: 0;
        }
        .toc-item {
            margin: 15px 0;
            padding: 15px 20px;
            background: #f8f9fa;
            border-radius: 8px;
            border-left: 4px solid #667eea;
            transition: transform 0.2s;
        }
        .toc-item:hover {
            transform: translateX(5px);
            background: #f1f3f5;
        }
        .toc-link {
            text-decoration: none;
            color: #333;
            display: flex;
            justify-content: space-between;
            align-items: center;
            font-size: 16px;
        }
        .page-num {
            color: #667eea;
            font-weight: 600;
            background: white;
            padding: 5px 12px;
            border-radius: 20px;
            font-size: 14px;
        }
        .section {
            margin-top: 30px;
        }
        .section-title {
            font-size: 20px;
            color: #667eea;
            margin: 20px 0 10px 0;
            font-weight: 600;
        }
    </style>
</head>
<body>
    <h1>Table of Contents</h1>
    
    <div class="section">
        <div class="section-title">Executive Summary</div>
        <div class="toc-list">
            <div class="toc-item">
                <a class="toc-link" href="#executive-summary">
                    <span>Executive Summary</span>
                    <span class="page-num">1</span>
                </a>
            </div>
        </div>
    </div>

    <div class="section">
        <div class="section-title">Key Metrics</div>
        <div class="toc-list">
            <div class="toc-item">
                <a class="toc-link" href="#key-metrics">
                    <span>Performance Metrics</span>
                    <span class="page-num">2</span>
                </a>
            </div>
            <div class="toc-item">
                <a class="toc-link" href="#issues-summary">
                    <span>Issues Summary</span>
                    <span class="page-num">2</span>
                </a>
            </div>
        </div>
    </div>

    <div class="section">
        <div class="section-title">Issues Found</div>
        <div class="toc-list">
            <div class="toc-item">
                <a class="toc-link" href="#critical-issues">
                    <span>Critical Issues</span>
                    <span class="page-num">3</span>
                </a>
            </div>
            <div class="toc-item">
                <a class="toc-link" href="#warnings">
                    <span>Warnings</span>
                    <span class="page-num">4</span>
                </a>
            </div>
            <div class="toc-item">
                <a class="toc-link" href="#info">
                    <span>Information</span>
                    <span class="page-num">5</span>
                </a>
            </div>
        </div>
    </div>

    <div class="section">
        <div class="section-title">Recommendations</div>
        <div class="toc-list">
            <div class="toc-item">
                <a class="toc-link" href="#recommendations">
                    <span>Action Items</span>
                    <span class="page-num">6</span>
                </a>
            </div>
        </div>
    </div>
</body>
</html>`

	// Generate TOC PDF
	tocPath := filepath.Join(pg.config.OutputDir, "toc_"+time.Now().Format("20060102_150405")+".pdf")
	_, err := pg.GenerateFromHTML(tocHTML, tocPath)
	if err != nil {
		return "", fmt.Errorf("failed to generate TOC: %w", err)
	}
	defer os.Remove(tocPath)

	// Merge TOC with original PDF
	outputPath := strings.TrimSuffix(pdfPath, ".pdf") + "_with_toc.pdf"
	return pg.MergePDFs([]string{tocPath, pdfPath}, outputPath)
}

// ValidateWKHTMLToPDF checks if wkhtmltopdf is properly installed
func (pg *PDFGenerator) ValidateWKHTMLToPDF() error {
	cmd := exec.Command(pg.wkBinary, "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute wkhtmltopdf: %w", err)
	}
	
	// Check if output contains version info
	if !strings.Contains(string(output), "wkhtmltopdf") {
		return fmt.Errorf("invalid wkhtmltopdf installation")
	}
	
	return nil
}

// CleanupTempFiles removes temporary files older than specified duration
func (pg *PDFGenerator) CleanupTempFiles(maxAge time.Duration) error {
	files, err := os.ReadDir(pg.config.OutputDir)
	if err != nil {
		return fmt.Errorf("failed to read output directory: %w", err)
	}

	now := time.Now()
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "temp_") || 
		   strings.Contains(file.Name(), "_temp_") ||
		   strings.HasPrefix(file.Name(), "cover_") ||
		   strings.HasPrefix(file.Name(), "toc_") {
			info, err := file.Info()
			if err != nil {
				continue
			}
			
			if now.Sub(info.ModTime()) > maxAge {
				filePath := filepath.Join(pg.config.OutputDir, file.Name())
				if err := os.Remove(filePath); err != nil {
					return fmt.Errorf("failed to remove temp file %s: %w", filePath, err)
				}
			}
		}
	}

	return nil
}