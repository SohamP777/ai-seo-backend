package reporting

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EmailConfig holds configuration for email sending
type EmailConfig struct {
	SMTPHost   string `json:"smtp_host"`
	SMTPPort   int    `json:"smtp_port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromEmail  string `json:"from_email"`
	FromName   string `json:"from_name"`
	UseTLS     bool   `json:"use_tls"`
	RetryCount int    `json:"retry_count"`
}

// EmailAttachment represents a file attachment
type EmailAttachment struct {
	Filename    string `json:"filename"`
	Content     []byte `json:"-"`
	ContentType string `json:"content_type"`
}

// EmailMessage represents a complete email message
type EmailMessage struct {
	To          []string          `json:"to"`
	CC          []string          `json:"cc"`
	BCC         []string          `json:"bcc"`
	Subject     string            `json:"subject"`
	Body        string            `json:"body"`
	IsHTML      bool              `json:"is_html"`
	Attachments []EmailAttachment `json:"-"`
}

// EmailReporter handles email delivery of reports
type EmailReporter struct {
	config EmailConfig
}

// NewEmailReporter creates a new email reporter instance
func NewEmailReporter(config EmailConfig) (*EmailReporter, error) {
	// Validate required fields
	if config.SMTPHost == "" {
		return nil, fmt.Errorf("SMTP host is required")
	}
	if config.SMTPPort == 0 {
		return nil, fmt.Errorf("SMTP port is required")
	}
	if config.FromEmail == "" {
		return nil, fmt.Errorf("from email is required")
	}
	if config.RetryCount == 0 {
		config.RetryCount = 3 // Default retry count
	}

	// Validate email format
	if _, err := mail.ParseAddress(config.FromEmail); err != nil {
		return nil, fmt.Errorf("invalid from email format: %w", err)
	}

	return &EmailReporter{
		config: config,
	}, nil
}

// SendReport sends a report as an email attachment
func (er *EmailReporter) SendReport(to []string, subject string, reportPath string, scanData *ScanResult) error {
	if len(to) == 0 {
		return fmt.Errorf("recipient list cannot be empty")
	}

	// Validate report file exists
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		return fmt.Errorf("report file does not exist: %s", reportPath)
	}

	// Generate email body
	body := er.generateEmailBody(scanData)

	// Create message
	msg := &EmailMessage{
		To:      to,
		Subject: subject,
		Body:    body,
		IsHTML:  true,
	}

	// Attach report file
	if err := er.attachFile(msg, reportPath); err != nil {
		return fmt.Errorf("failed to attach file: %w", err)
	}

	// Send message with retry logic
	return er.sendWithRetry(msg)
}

// SendBatchReports sends multiple reports to multiple recipients
func (er *EmailReporter) SendBatchReports(recipients map[string][]string, reportPaths []string) error {
	if len(recipients) == 0 {
		return fmt.Errorf("recipients map cannot be empty")
	}
	if len(reportPaths) == 0 {
		return fmt.Errorf("report paths cannot be empty")
	}

	var errors []string
	for email, reports := range recipients {
		// For each recipient, create a message with their specific reports
		msg := &EmailMessage{
			To:      []string{email},
			Subject: fmt.Sprintf("SEO Reports - %s", time.Now().Format("2006-01-02")),
			Body:    "Please find attached the requested SEO reports.",
			IsHTML:  false,
		}

		// Attach each report
		for _, reportPath := range reports {
			if err := er.attachFile(msg, reportPath); err != nil {
				errors = append(errors, fmt.Sprintf("failed to attach %s for %s: %v", reportPath, email, err))
				continue
			}
		}

		// Send if we have attachments
		if len(msg.Attachments) > 0 {
			if err := er.sendWithRetry(msg); err != nil {
				errors = append(errors, fmt.Sprintf("failed to send to %s: %v", email, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch send completed with errors: %s", strings.Join(errors, "; "))
	}
	return nil
}

// generateEmailBody creates HTML email body with report summary
func (er *EmailReporter) generateEmailBody(scanData *ScanResult) string {
	if scanData == nil {
		return "Please find attached the SEO report."
	}

	// Calculate basic metrics
	totalIssues := len(scanData.Issues)
	var criticalCount, warningCount, infoCount int
	
	for _, issue := range scanData.Issues {
		switch strings.ToLower(issue.Severity) {
		case "critical", "high":
			criticalCount++
		case "warning", "medium":
			warningCount++
		default:
			infoCount++
		}
	}

	// Safely handle pages count
	pagesCount := 0
	if scanData.Pages != nil {
		pagesCount = len(scanData.Pages)
	}

	var body strings.Builder
	body.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6; 
            color: #333; 
            background: #f4f6f9;
            padding: 20px;
        }
        .container { 
            max-width: 600px; 
            margin: 0 auto; 
            background: white; 
            border-radius: 12px;
            overflow: hidden;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        .header { 
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white; 
            padding: 30px; 
            text-align: center; 
        }
        .header h1 { font-size: 28px; margin-bottom: 10px; }
        .header p { font-size: 16px; opacity: 0.9; }
        .content { padding: 30px; }
        .section { margin-bottom: 30px; }
        .section h2 { 
            font-size: 20px; 
            margin-bottom: 15px; 
            color: #333;
            border-bottom: 2px solid #f0f0f0;
            padding-bottom: 10px;
        }
        .metric-box {
            background: #f8fafc;
            border-radius: 8px;
            padding: 15px;
            margin: 15px 0;
            border-left: 4px solid #667eea;
        }
        .metric-row {
            display: flex;
            justify-content: space-between;
            margin: 10px 0;
        }
        .metric-label {
            color: #64748b;
            font-weight: 500;
        }
        .metric-value {
            font-weight: 600;
            color: #1e293b;
        }
        .severity-badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 14px;
            font-weight: 500;
            margin-right: 10px;
        }
        .severity-critical { background: #fee2e2; color: #dc2626; }
        .severity-warning { background: #fef3c7; color: #d97706; }
        .severity-info { background: #dbeafe; color: #2563eb; }
        .footer { 
            background: #f8fafc; 
            padding: 20px 30px; 
            text-align: center; 
            font-size: 14px; 
            color: #64748b;
            border-top: 1px solid #e2e8f0;
        }
        .button {
            display: inline-block;
            padding: 12px 24px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            text-decoration: none;
            border-radius: 8px;
            font-weight: 500;
            margin: 20px 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>SEO Report Summary</h1>
            <p>` + scanData.URL + `</p>
        </div>
        
        <div class="content">
            <div class="section">
                <h2>Quick Overview</h2>
                <div class="metric-box">
                    <div class="metric-row">
                        <span class="metric-label">Report Generated:</span>
                        <span class="metric-value">` + scanData.GeneratedAt.Format("January 2, 2006 15:04") + `</span>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">Pages Analyzed:</span>
                        <span class="metric-value">` + fmt.Sprintf("%d", pagesCount) + `</span>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">Total Issues:</span>
                        <span class="metric-value">` + fmt.Sprintf("%d", totalIssues) + `</span>
                    </div>
                </div>
            </div>

            <div class="section">
                <h2>Issues Summary</h2>
                <div class="metric-box">
                    <div class="metric-row">
                        <span class="metric-label">
                            <span class="severity-badge severity-critical">Critical</span>
                        </span>
                        <span class="metric-value">` + fmt.Sprintf("%d", criticalCount) + `</span>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">
                            <span class="severity-badge severity-warning">Warnings</span>
                        </span>
                        <span class="metric-value">` + fmt.Sprintf("%d", warningCount) + `</span>
                    </div>
                    <div class="metric-row">
                        <span class="metric-label">
                            <span class="severity-badge severity-info">Info</span>
                        </span>
                        <span class="metric-value">` + fmt.Sprintf("%d", infoCount) + `</span>
                    </div>
                </div>
            </div>`)

	if len(scanData.Issues) > 0 {
		body.WriteString(`
            <div class="section">
                <h2>Top Issues</h2>
                <div class="metric-box">`)
		
		// Show top 5 issues
		maxIssues := 5
		if len(scanData.Issues) < maxIssues {
			maxIssues = len(scanData.Issues)
		}
		
		for i := 0; i < maxIssues; i++ {
			issue := scanData.Issues[i]
			severityClass := "severity-info"
			if strings.ToLower(issue.Severity) == "critical" || strings.ToLower(issue.Severity) == "high" {
				severityClass = "severity-critical"
			} else if strings.ToLower(issue.Severity) == "warning" || strings.ToLower(issue.Severity) == "medium" {
				severityClass = "severity-warning"
			}
			
			body.WriteString(`
                    <div style="margin: 15px 0; padding: 10px 0; border-bottom: 1px solid #e2e8f0;">
                        <span class="severity-badge ` + severityClass + `">` + issue.Severity + `</span>
                        <p style="margin: 8px 0 0 0; color: #1e293b;"><strong>` + issue.Type + `:</strong> ` + issue.Description + `</p>
                    </div>`)
		}
		
		body.WriteString(`
                </div>
            </div>`)
	}

	body.WriteString(`
            <div style="text-align: center;">
                <p style="margin-bottom: 20px;">The detailed report is attached to this email.</p>
                <p style="color: #666; font-size: 14px;">Review the complete analysis and recommendations in the attached file.</p>
            </div>
        </div>
        
        <div class="footer">
            <p>This is an automated message from the SEO Reporting System.</p>
            <p style="margin-top: 10px;">© 2024 AI SEO Tool. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`)

	return body.String()
}

// validateEmail performs basic email validation
func (er *EmailReporter) validateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	
	// Basic format validation
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email format: %w", err)
	}
	
	// Check for common invalid patterns
	if strings.Contains(addr.Address, "..") || 
	   strings.HasPrefix(addr.Address, ".") || 
	   strings.HasSuffix(addr.Address, ".") {
		return fmt.Errorf("email contains invalid dot patterns")
	}
	
	return nil
}

// attachFile adds a file as an attachment to the message
func (er *EmailReporter) attachFile(msg *EmailMessage, filePath string) error {
	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Determine content type
	contentType := "application/octet-stream"
	switch {
	case strings.HasSuffix(filePath, ".pdf"):
		contentType = "application/pdf"
	case strings.HasSuffix(filePath, ".html"):
		contentType = "text/html"
	case strings.HasSuffix(filePath, ".json"):
		contentType = "application/json"
	case strings.HasSuffix(filePath, ".csv"):
		contentType = "text/csv"
	case strings.HasSuffix(filePath, ".md"):
		contentType = "text/markdown"
	}

	// Get filename from path
	filename := filepath.Base(filePath)

	// Create attachment
	attachment := EmailAttachment{
		Filename:    filename,
		Content:     content,
		ContentType: contentType,
	}

	msg.Attachments = append(msg.Attachments, attachment)
	return nil
}

// buildMimeMessage constructs the complete MIME email message
func (er *EmailReporter) buildMimeMessage(msg *EmailMessage) ([]byte, error) {
	var buffer bytes.Buffer
	
	// Create multipart writer
	writer := multipart.NewWriter(&buffer)
	boundary := writer.Boundary()

	// Write headers
	buffer.WriteString(fmt.Sprintf("From: %s <%s>\r\n", er.config.FromName, er.config.FromEmail))
	buffer.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	if len(msg.CC) > 0 {
		buffer.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.CC, ", ")))
	}
	buffer.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	buffer.WriteString("MIME-Version: 1.0\r\n")
	buffer.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", boundary))
	buffer.WriteString("\r\n")

	// Write body part
	buffer.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	if msg.IsHTML {
		buffer.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	} else {
		buffer.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	}
	buffer.WriteString("Content-Transfer-Encoding: 7bit\r\n")
	buffer.WriteString("\r\n")
	buffer.WriteString(msg.Body)
	buffer.WriteString("\r\n")

	// Write attachments
	for _, attachment := range msg.Attachments {
		buffer.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buffer.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", 
			attachment.ContentType, attachment.Filename))
		buffer.WriteString("Content-Transfer-Encoding: base64\r\n")
		buffer.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", 
			attachment.Filename))
		buffer.WriteString("\r\n")
		
		// Encode content as base64
		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(attachment.Content)))
		base64.StdEncoding.Encode(encoded, attachment.Content)
		
		// Write in chunks of 76 characters
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			buffer.Write(encoded[i:end])
			buffer.WriteString("\r\n")
		}
		buffer.WriteString("\r\n")
	}
	
	// Close multipart
	buffer.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return buffer.Bytes(), nil
}

// Send sends the email message via SMTP
func (er *EmailReporter) Send(msg *EmailMessage) error {
	if len(msg.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	// Validate all email addresses
	allRecipients := append(msg.To, msg.CC...)
	allRecipients = append(allRecipients, msg.BCC...)
	
	for _, email := range allRecipients {
		if err := er.validateEmail(email); err != nil {
			return fmt.Errorf("invalid recipient email %s: %w", email, err)
		}
	}

	// Build MIME message
	mimeMessage, err := er.buildMimeMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to build MIME message: %w", err)
	}

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", er.config.SMTPHost, er.config.SMTPPort)
	
	var conn *smtp.Client
	
	if er.config.UseTLS {
		// TLS connection
		tlsConfig := &tls.Config{
			ServerName: er.config.SMTPHost,
		}
		conn, err = smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to dial SMTP: %w", err)
		}
		defer conn.Close()
		
		if err = conn.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	} else {
		// Plain connection
		conn, err = smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to dial SMTP: %w", err)
		}
		defer conn.Close()
	}

	// Auth if credentials provided
	if er.config.Username != "" && er.config.Password != "" {
		auth := smtp.PlainAuth("", er.config.Username, er.config.Password, er.config.SMTPHost)
		if err = conn.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Set sender and recipients
	if err = conn.Mail(er.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	
	for _, recipient := range msg.To {
		if err = conn.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send data
	wc, err := conn.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer wc.Close()

	if _, err = wc.Write(mimeMessage); err != nil {
		return fmt.Errorf("failed to write email data: %w", err)
	}

	// Log successful send
	fmt.Printf("Email sent successfully to %s at %s\n", 
		strings.Join(msg.To, ", "), 
		time.Now().Format("2006-01-02 15:04:05"))

	return nil
}

// sendWithRetry sends email with retry logic and exponential backoff
func (er *EmailReporter) sendWithRetry(msg *EmailMessage) error {
	var lastErr error
	
	for attempt := 1; attempt <= er.config.RetryCount; attempt++ {
		err := er.Send(msg)
		if err == nil {
			return nil
		}
		
		lastErr = err
		
		if attempt < er.config.RetryCount {
			// Exponential backoff: 1s, 2s, 4s, etc.
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			fmt.Printf("Send attempt %d failed, retrying in %v: %v\n", 
				attempt, backoff, err)
			time.Sleep(backoff)
		}
	}
	
	return fmt.Errorf("failed after %d attempts: %w", er.config.RetryCount, lastErr)
}