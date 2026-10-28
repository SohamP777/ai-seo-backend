package services

import (
    "bytes"
    "crypto/tls"
    "fmt"
    "html/template"
    "net/smtp"
    "strings"
    "time"
)

type EmailService struct {
    host     string
    port     string
    username string
    password string
    from     string
    templates map[string]*template.Template
}

type Email struct {
    To      []string
    Subject string
    Body    string
    HTML    bool
}

// NewEmailService creates a new email service
func NewEmailService(host, port, username, password, from string) *EmailService {
    s := &EmailService{
        host:      host,
        port:      port,
        username:  username,
        password:  password,
        from:      from,
        templates: make(map[string]*template.Template),
    }

    // Load templates
    s.loadTemplates()
    
    return s
}

// Send sends an email
func (s *EmailService) Send(email Email) error {
    if len(email.To) == 0 {
        return fmt.Errorf("no recipients specified")
    }

    // Build email headers
    var body bytes.Buffer
    
    // Headers
    body.WriteString(fmt.Sprintf("From: %s\r\n", s.from))
    body.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ", ")))
    body.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
    
    if email.HTML {
        body.WriteString("MIME-Version: 1.0\r\n")
        body.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
    } else {
        body.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
    }
    
    body.WriteString("\r\n")
    body.WriteString(email.Body)

    // Send email
    auth := smtp.PlainAuth("", s.username, s.password, s.host)
    
    // Configure TLS
    tlsConfig := &tls.Config{
        InsecureSkipVerify: true,
        ServerName:         s.host,
    }

    conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%s", s.host, s.port), tlsConfig)
    if err != nil {
        return fmt.Errorf("failed to connect: %w", err)
    }

    client, err := smtp.NewClient(conn, s.host)
    if err != nil {
        return fmt.Errorf("failed to create client: %w", err)
    }
    defer client.Close()

    if err = client.Auth(auth); err != nil {
        return fmt.Errorf("failed to authenticate: %w", err)
    }

    if err = client.Mail(s.from); err != nil {
        return fmt.Errorf("failed to set from: %w", err)
    }

    for _, addr := range email.To {
        if err = client.Rcpt(addr); err != nil {
            return fmt.Errorf("failed to set recipient %s: %w", addr, err)
        }
    }

    w, err := client.Data()
    if err != nil {
        return fmt.Errorf("failed to start data: %w", err)
    }

    _, err = w.Write(body.Bytes())
    if err != nil {
        return fmt.Errorf("failed to write body: %w", err)
    }

    err = w.Close()
    if err != nil {
        return fmt.Errorf("failed to close writer: %w", err)
    }

    return nil
}

// SendWelcomeEmail sends a welcome email to new users
func (s *EmailService) SendWelcomeEmail(to, name string) error {
    data := map[string]interface{}{
        "Name":      name,
        "Year":      time.Now().Year(),
        "LoginURL":  "https://yourapp.com/login",
        "SupportURL": "https://yourapp.com/support",
    }

    body, err := s.renderTemplate("welcome", data)
    if err != nil {
        return err
    }

    email := Email{
        To:      []string{to},
        Subject: "Welcome to Keyword Optimizer!",
        Body:    body,
        HTML:    true,
    }

    return s.Send(email)
}

// SendPasswordResetEmail sends a password reset email
func (s *EmailService) SendPasswordResetEmail(to, resetToken string) error {
    data := map[string]interface{}{
        "ResetURL": fmt.Sprintf("https://yourapp.com/reset-password?token=%s", resetToken),
        "Expiry":   "1 hour",
        "Year":     time.Now().Year(),
    }

    body, err := s.renderTemplate("password_reset", data)
    if err != nil {
        return err
    }

    email := Email{
        To:      []string{to},
        Subject: "Password Reset Request",
        Body:    body,
        HTML:    true,
    }

    return s.Send(email)
}

// SendPaymentConfirmation sends a payment confirmation email
func (s *EmailService) SendPaymentConfirmation(to, name, plan string, amount float64) error {
    data := map[string]interface{}{
        "Name":      name,
        "Plan":      plan,
        "Amount":    amount,
        "Date":      time.Now().Format("January 2, 2006"),
        "InvoiceURL": "https://yourapp.com/invoice",
        "Year":      time.Now().Year(),
    }

    body, err := s.renderTemplate("payment_confirmation", data)
    if err != nil {
        return err
    }

    email := Email{
        To:      []string{to},
        Subject: fmt.Sprintf("Payment Confirmation - %s Plan", plan),
        Body:    body,
        HTML:    true,
    }

    return s.Send(email)
}

// SendSubscriptionExpiry sends a subscription expiry warning
func (s *EmailService) SendSubscriptionExpiry(to, name, plan string, daysLeft int) error {
    data := map[string]interface{}{
        "Name":      name,
        "Plan":      plan,
        "DaysLeft":  daysLeft,
        "RenewURL":  "https://yourapp.com/renew",
        "Year":      time.Now().Year(),
    }

    body, err := s.renderTemplate("subscription_expiry", data)
    if err != nil {
        return err
    }

    email := Email{
        To:      []string{to},
        Subject: fmt.Sprintf("Your %s Subscription Expires in %d Days", plan, daysLeft),
        Body:    body,
        HTML:    true,
    }

    return s.Send(email)
}

// SendReportEmail sends an SEO report via email
func (s *EmailService) SendReportEmail(to string, reportData map[string]interface{}) error {
    data := map[string]interface{}{
        "Date":      time.Now().Format("January 2006"),
        "Year":      time.Now().Year(),
        "ReportURL": "https://yourapp.com/reports/latest",
    }

    // Merge report data
    for k, v := range reportData {
        data[k] = v
    }

    body, err := s.renderTemplate("seo_report", data)
    if err != nil {
        return err
    }

    email := Email{
        To:      []string{to},
        Subject: fmt.Sprintf("Your SEO Report - %s", time.Now().Format("January 2006")),
        Body:    body,
        HTML:    true,
    }

    return s.Send(email)
}

// SendBulk sends bulk emails
func (s *EmailService) SendBulk(emails []Email) (successful int, failed []error) {
    for _, email := range emails {
        if err := s.Send(email); err != nil {
            failed = append(failed, err)
        } else {
            successful++
        }
    }
    return
}

// Helper functions
func (s *EmailService) loadTemplates() {
    // Define email templates
    templates := map[string]string{
        "welcome": `
<!DOCTYPE html>
<html>
<head>
    <title>Welcome to Keyword Optimizer</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #4CAF50;">Welcome, {{.Name}}!</h1>
        <p>Thank you for joining Keyword Optimizer. We're excited to help you improve your SEO!</p>
        
        <h2>Getting Started</h2>
        <ul>
            <li><a href="{{.LoginURL}}">Login to your account</a></li>
            <li>Analyze your first keyword</li>
            <li>Check competitor gaps</li>
            <li>Optimize your content</li>
        </ul>
        
        <p>Need help? Contact our support team at <a href="mailto:support@keywordoptimizer.com">support@keywordoptimizer.com</a></p>
        
        <hr>
        <p style="color: #777; font-size: 12px;">&copy; {{.Year}} Keyword Optimizer. All rights reserved.</p>
    </div>
</body>
</html>
`,
        "password_reset": `
<!DOCTYPE html>
<html>
<head>
    <title>Password Reset Request</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2>Password Reset Request</h2>
        <p>We received a request to reset your password. Click the button below to reset it:</p>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.ResetURL}}" style="background-color: #4CAF50; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px;">Reset Password</a>
        </div>
        
        <p>This link will expire in {{.Expiry}}.</p>
        <p>If you didn't request this, please ignore this email.</p>
        
        <hr>
        <p style="color: #777; font-size: 12px;">&copy; {{.Year}} Keyword Optimizer. All rights reserved.</p>
    </div>
</body>
</html>
`,
        "payment_confirmation": `
<!DOCTYPE html>
<html>
<head>
    <title>Payment Confirmation</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #4CAF50;">Payment Confirmed!</h2>
        <p>Thank you for your payment, {{.Name}}!</p>
        
        <div style="background-color: #f5f5f5; padding: 20px; border-radius: 5px; margin: 20px 0;">
            <h3>Payment Details</h3>
            <p><strong>Plan:</strong> {{.Plan}}</p>
            <p><strong>Amount:</strong> ${{.Amount}}</p>
            <p><strong>Date:</strong> {{.Date}}</p>
        </div>
        
        <p><a href="{{.InvoiceURL}}">View Invoice</a></p>
        
        <hr>
        <p style="color: #777; font-size: 12px;">&copy; {{.Year}} Keyword Optimizer. All rights reserved.</p>
    </div>
</body>
</html>
`,
        "subscription_expiry": `
<!DOCTYPE html>
<html>
<head>
    <title>Subscription Expiry Notice</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2>Your Subscription is Expiring Soon</h2>
        <p>Hi {{.Name}},</p>
        <p>Your {{.Plan}} subscription will expire in <strong>{{.DaysLeft}} days</strong>.</p>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.RenewURL}}" style="background-color: #4CAF50; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px;">Renew Now</a>
        </div>
        
        <p>Renew now to continue enjoying all features of Keyword Optimizer.</p>
        
        <hr>
        <p style="color: #777; font-size: 12px;">&copy; {{.Year}} Keyword Optimizer. All rights reserved.</p>
    </div>
</body>
</html>
`,
        "seo_report": `
<!DOCTYPE html>
<html>
<head>
    <title>Your SEO Report</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #4CAF50;">Your Monthly SEO Report - {{.Date}}</h2>
        
        <div style="background-color: #f5f5f5; padding: 20px; border-radius: 5px; margin: 20px 0;">
            <h3>Summary</h3>
            {{if .TotalKeywords}}
            <p><strong>Keywords Tracked:</strong> {{.TotalKeywords}}</p>
            {{end}}
            {{if .AvgPosition}}
            <p><strong>Average Position:</strong> {{.AvgPosition}}</p>
            {{end}}
            {{if .TotalTraffic}}
            <p><strong>Estimated Traffic:</strong> {{.TotalTraffic}}</p>
            {{end}}
        </div>
        
        {{if .TopOpportunities}}
        <h3>Top Opportunities</h3>
        <ul>
            {{range .TopOpportunities}}
            <li>{{.}}</li>
            {{end}}
        </ul>
        {{end}}
        
        <p><a href="{{.ReportURL}}" style="background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">View Full Report</a></p>
        
        <hr>
        <p style="color: #777; font-size: 12px;">&copy; {{.Year}} Keyword Optimizer. All rights reserved.</p>
    </div>
</body>
</html>
`,
    }

    // Parse templates
    for name, content := range templates {
        s.templates[name] = template.Must(template.New(name).Parse(content))
    }
}

func (s *EmailService) renderTemplate(name string, data interface{}) (string, error) {
    tmpl, ok := s.templates[name]
    if !ok {
        return "", fmt.Errorf("template %s not found", name)
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("failed to render template: %w", err)
    }

    return buf.String(), nil
}