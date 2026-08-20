package guide

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"log"

	"github.com/sashabaranov/go-openai"
)

// ========== MODELS ==========

type SEOIssue struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
	Impact      string   `json:"impact"`
	TimeToFix   string   `json:"time_to_fix"`
	Platforms   []string `json:"platforms"`
}

type Generator struct {
	openAIClient *openai.Client
	logger       *log.Logger
}

type Step struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Action      string `json:"action"`
	Location    string `json:"location"`
	Code        string `json:"code,omitempty"`
	Tip         string `json:"tip,omitempty"`
	Warning     string `json:"warning,omitempty"`
	Description string 
}

type WebsiteCheck struct {
	URL              string
	HasSSL           bool
	HasMetaDescription bool
	HasTitle         bool
	HasH1            bool
	LoadTime         float64
	StatusCode       int
	IssuesFound      []string
}

type Guide struct {
    Title           string
    Issue           SEOIssue     // ← Change from string to SEOIssue
    Platform        string
    Difficulty      string
    EstimatedTime   string
    Steps           []Step       // ← Keep as []Step
    Verification   []string
    Tools           []string
    ExpectedResults []string  
	Content          string    // ← Change from string to []string
}

// ========== REAL SEO ISSUES DATABASE ==========

var seoIssues = []SEOIssue{
	{
		ID:          "missing_title_tag",
		Name:        "Missing or Poor Title Tags",
		Description: "Title tags are the most important on-page SEO factor. They appear in search results and browser tabs.",
		Priority:    "critical",
		Impact:      "Google uses titles as primary ranking factor",
		TimeToFix:   "2-5 minutes per page",
		Platforms:   []string{"wordpress", "shopify", "wix", "webflow", "squarespace", "html"},
	},
	{
		ID:          "missing_meta_description",
		Name:        "Missing Meta Descriptions",
		Description: "Meta descriptions influence click-through rates from search results.",
		Priority:    "high",
		Impact:      "Directly affects CTR from Google",
		TimeToFix:   "2-3 minutes per page",
		Platforms:   []string{"wordpress", "shopify", "wix", "webflow", "squarespace", "html"},
	},
	{
		ID:          "no_ssl",
		Name:        "Missing SSL Certificate",
		Description: "SSL encrypts data and is a Google ranking factor.",
		Priority:    "critical",
		Impact:      "Required for ranking - Chrome marks HTTP as 'Not Secure'",
		TimeToFix:   "15-30 minutes",
		Platforms:   []string{"wordpress", "shopify", "wix", "webflow", "squarespace", "html"},
	},
	{
		ID:          "missing_h1",
		Name:        "Missing H1 Headings",
		Description: "H1 tags help search engines understand page structure.",
		Priority:    "high",
		Impact:      "Helps Google understand page hierarchy",
		TimeToFix:   "2 minutes per page",
		Platforms:   []string{"wordpress", "shopify", "wix", "webflow", "squarespace", "html"},
	},
	{
		ID:          "no_sitemap",
		Name:        "Missing XML Sitemap",
		Description: "Sitemaps help search engines find and index your pages.",
		Priority:    "high",
		Impact:      "Faster indexing by Google",
		TimeToFix:   "10-15 minutes",
		Platforms:   []string{"wordpress", "shopify", "wix", "webflow", "squarespace", "html"},
	},
	{
		ID:          "missing_alt_text",
		Name:        "Missing Image Alt Text",
		Description: "Alt text helps search engines understand images.",
		Priority:    "medium",
		Impact:      "Helps images rank in Google Images",
		TimeToFix:   "1 minute per image",
		Platforms:   []string{"wordpress", "shopify", "wix", "webflow", "squarespace", "html"},
	},
	{
		ID:          "no_robots_txt",
		Name:        "Missing robots.txt",
		Description: "Robots.txt controls search engine crawling.",
		Priority:    "medium",
		Impact:      "Controls which pages Google can crawl",
		TimeToFix:   "10 minutes",
		Platforms:   []string{"wordpress", "shopify", "wix", "webflow", "squarespace", "html"},
	},
}

// ========== REAL WEBSITE CHECKER ==========

func NewGenerator(apiKey string, logger *log.Logger) *Generator {
    logger.Printf("OpenAI API Key length: %d", len(apiKey))
    if apiKey == "" {
        logger.Printf("WARNING: OpenAI API key is empty!")
    }
    return &Generator{
        openAIClient: openai.NewClient(apiKey),
        logger:       logger,
    }
}

func (g *Generator) GenerateGuide(issueType, platform, analysisData string) (*Guide, error) {
    // Call OpenAI to generate personalized guide
    prompt := fmt.Sprintf(`
        Analyze this website and create SEO recommendations:
        Platform: %s
        Analysis: %s
        
        Provide 5 specific, actionable steps for fixing SEO issues.
        Include step numbers, titles, actions, and tips.
        Format as JSON with structure: {"steps":[{"number":1,"title":"...","action":"...","tip":"..."}]}
    `, platform, analysisData)
    
    resp, err := g.openAIClient.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
        Model: "gpt-3.5-turbo",
        Messages: []openai.ChatCompletionMessage{
            {Role: "user", Content: prompt},
        },
    })
    
    if err != nil {
        g.logger.Printf("Error calling OpenAI: %v", err)
        return nil, err
    }
    
    // Parse AI response into Guide struct
    aiContent := resp.Choices[0].Message.Content
    
    // Create a guide with AI-generated content
    guide := &Guide{
        Title:    "AI-Generated SEO Guide",
        Platform: platform,
        Steps:    parseAIResponse(aiContent),
        Verification: []string{
            "Check using Google Search Console",
            "Monitor rankings for 2-4 weeks",
            "Run another SEO audit",
        },
        Tools: []string{
            "Google Search Console",
            "PageSpeed Insights",
            "SEO Testing Tool",
        },
        ExpectedResults: []string{
            "Improved search rankings",
            "Better user engagement",
            "Increased organic traffic",
        },
    }
    
    return guide, nil
}

// Helper function to parse AI response
func parseAIResponse(aiContent string) []Step {
    // Simple parsing - in production you might want to parse JSON
    steps := []Step{
        {
            Number: 1,
            Title:  "AI Recommendation",
            Action: aiContent,
            Tip:    "Follow these AI-generated recommendations",
        },
    }
    return steps
}

func CheckWebsite(url string) (*WebsiteCheck, error) {
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}
	
	check := &WebsiteCheck{
		URL:         url,
		IssuesFound: []string{},
	}
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // Don't follow redirects to check original URL
		},
	}
	
	// Check HTTPS
	httpsURL := url
	if !strings.HasPrefix(httpsURL, "https://") {
		httpsURL = strings.Replace(httpsURL, "http://", "https://", 1)
	}
	
	resp, err := client.Get(httpsURL)
	if err != nil {
		// Try HTTP if HTTPS fails
		httpURL := strings.Replace(url, "https://", "http://", 1)
		resp, err = client.Get(httpURL)
		if err != nil {
			return check, fmt.Errorf("cannot access website: %v", err)
		}
		check.HasSSL = false
		check.IssuesFound = append(check.IssuesFound, "no_ssl")
	} else {
		check.HasSSL = true
	}
	
	if resp != nil {
		defer resp.Body.Close()
		check.StatusCode = resp.StatusCode
		
		// Read body
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			content := string(body)
			
			// Check for title tag
			titleRegex := regexp.MustCompile(`<title>(.*?)</title>`)
			if !titleRegex.MatchString(content) {
				check.IssuesFound = append(check.IssuesFound, "missing_title_tag")
			} else {
				check.HasTitle = true
			}
			
			// Check for meta description
			metaRegex := regexp.MustCompile(`<meta\s+name=["']description["']\s+content=["'](.*?)["']`)
			if !metaRegex.MatchString(content) {
				check.IssuesFound = append(check.IssuesFound, "missing_meta_description")
			} else {
				check.HasMetaDescription = true
			}
			
			// Check for H1
			h1Regex := regexp.MustCompile(`<h1[^>]*>.*?</h1>`)
			if !h1Regex.MatchString(content) {
				check.IssuesFound = append(check.IssuesFound, "missing_h1")
			} else {
				check.HasH1 = true
			}
			
			// Check for sitemap
			if !strings.Contains(content, "sitemap") && !strings.Contains(content, "sitemap.xml") {
				check.IssuesFound = append(check.IssuesFound, "no_sitemap")
			}
			
			// Check for alt text
			imgRegex := regexp.MustCompile(`<img[^>]+>`)
			images := imgRegex.FindAllString(content, -1)
			hasMissingAlt := false
			for _, img := range images {
				if !strings.Contains(img, "alt=") {
					hasMissingAlt = true
					break
				}
			}
			if hasMissingAlt && len(images) > 0 {
				check.IssuesFound = append(check.IssuesFound, "missing_alt_text")
			}
			
			// Check for robots.txt
			robotsURL := strings.TrimRight(url, "/") + "/robots.txt"
			robotsResp, err := client.Get(robotsURL)
			if err != nil || robotsResp.StatusCode != 200 {
				check.IssuesFound = append(check.IssuesFound, "no_robots_txt")
			}
			if robotsResp != nil {
				robotsResp.Body.Close()
			}
		}
	}
	
	return check, nil
}

// ========== GUIDE GENERATION ==========

func GenerateGuide(issueID, platform string) (*Guide, error) {
	var issue SEOIssue
	found := false
	for _, i := range seoIssues {
		if i.ID == issueID {
			issue = i
			found = true
			break
		}
	}
	
	if !found {
		return nil, fmt.Errorf("issue not found: %s", issueID)
	}
	
	steps := generateSteps(issueID, platform)
	
	guide := &Guide{
		Issue:    issue,
		Platform: platform,
		Steps:    steps,
		Verification: []string{
			"Check using Google Search Console",
			"Use SEO testing tool to verify fix",
			"Monitor rankings for 2-4 weeks",
		},
		Tools: []string{
			"Google Search Console - https://search.google.com/search-console",
			"PageSpeed Insights - https://pagespeed.web.dev",
			"Rich Results Test - https://search.google.com/test/rich-results",
		},
		ExpectedResults: getExpectedResults(issueID),
	}
	
	return guide, nil
}

func generateSteps(issueID, platform string) []Step {
	steps := []Step{}
	
	// Add login step
	steps = append(steps, Step{
		Number:   1,
		Title:    "Access Your Website Dashboard",
		Action:   "Log into your " + getPlatformName(platform) + " admin panel",
		Location: getLoginURL(platform),
		Tip:      "Use strong password and 2FA if available",
	})
	
	switch issueID {
	case "missing_title_tag":
		steps = append(steps, getTitleTagSteps(platform)...)
	case "missing_meta_description":
		steps = append(steps, getMetaDescriptionSteps(platform)...)
	case "no_ssl":
		steps = append(steps, getSSLSteps(platform)...)
	case "missing_h1":
		steps = append(steps, getH1Steps(platform)...)
	case "no_sitemap":
		steps = append(steps, getSitemapSteps(platform)...)
	case "missing_alt_text":
		steps = append(steps, getAltTextSteps(platform)...)
	case "no_robots_txt":
		steps = append(steps, getRobotsTxtSteps(platform)...)
	}
	
	return steps
}

func getTitleTagSteps(platform string) []Step {
	if platform == "wordpress" {
		return []Step{
			{
				Number:    2,
				Title:     "Install SEO Plugin",
				Action:    "Go to Plugins → Add New → Search 'Yoast SEO' → Install → Activate",
				Location:  "WordPress Admin Panel",
				Tip:       "Yoast SEO is free and used by over 5 million websites",
			},
			{
				Number:    3,
				Title:     "Edit Your Page",
				Action:    "Go to Pages → Click 'Edit' on the page you want to optimize",
				Location:  "WordPress Admin → Pages",
				Tip:       "Start with your homepage and most important pages first",
			},
			{
				Number:    4,
				Title:     "Add Title Tag",
				Action:    "Scroll to Yoast SEO section → Click 'Edit snippet' → Enter your title",
				Location:  "Below the page editor",
				Code:      "Example: 'Complete Guide to SEO | Your Brand Name'",
				Tip:       "Keep titles under 60 characters and include your main keyword",
			},
		}
	}
	
	return []Step{
		{
			Number:    2,
			Title:     "Edit HTML Directly",
			Action:    "Open your HTML file in a text editor",
			Location:  "Your website files",
			Code:      "<title>Your Page Title Here (Under 60 Characters)</title>",
			Tip:       "Place this between <head> and </head> tags",
		},
	}
}

func getMetaDescriptionSteps(platform string) []Step {
	if platform == "wordpress" {
		return []Step{
			{
				Number:    2,
				Title:     "Open Yoast SEO Meta Box",
				Action:    "Scroll down below the content editor",
				Location:  "Page/Post editor",
				Tip:       "Yoast SEO box is usually near the bottom",
			},
			{
				Number:    3,
				Title:     "Write Meta Description",
				Action:    "Click 'Edit snippet' → In 'Meta description' field, write 150-160 characters",
				Location:  "Yoast SEO snippet editor",
				Code:      "Learn how to [solve problem] with our step-by-step guide including tips and best practices.",
				Tip:       "Include your keyword naturally and make it compelling to click",
			},
		}
	}
	
	return []Step{
		{
			Number:    2,
			Title:     "Add Meta Description Tag",
			Action:    "Add this meta tag inside your HTML <head> section",
			Location:  "HTML head section",
			Code:      `<meta name="description" content="Your compelling description here (150-160 characters). Include keyword naturally.">`,
			Tip:       "Each page needs a unique meta description",
		},
	}
}

func getSSLSteps(platform string) []Step {
	return []Step{
		{
			Number:    2,
			Title:     "Get SSL Certificate",
			Action:    "Contact your hosting provider OR use Let's Encrypt (free)",
			Location:  "Hosting control panel",
			Tip:       "Many hosts offer free SSL through Let's Encrypt",
		},
		{
			Number:    3,
			Title:     "Install SSL Certificate",
			Action:    "Follow your hosting provider's SSL installation process",
			Location:  "Hosting control panel → Security → SSL",
			Tip:       "Most hosts have 1-click SSL installation",
		},
		{
			Number:    4,
			Title:     "Force HTTPS Redirects",
			Action:    "Set up 301 redirect from HTTP to HTTPS",
			Location:  ".htaccess or nginx config",
			Code:      "RewriteEngine On\nRewriteCond %{HTTPS} off\nRewriteRule ^(.*)$ https://%{HTTP_HOST}/$1 [R=301,L]",
			Tip:       "Test by visiting http://yourdomain.com - it should redirect to https://",
		},
	}
}

func getH1Steps(platform string) []Step {
	return []Step{
		{
			Number:    2,
			Title:     "Add H1 Heading",
			Action:    "Edit your page and add an H1 heading at the top",
			Location:  "Page content editor",
			Code:      "<h1>Your Main Page Heading That Describes the Content</h1>",
			Tip:       "Use only ONE H1 per page. Make it descriptive and include your target keyword",
		},
	}
}

func getSitemapSteps(platform string) []Step {
	steps := []Step{}
	
	if platform == "wordpress" {
		steps = append(steps, Step{
			Number:    2,
			Title:     "Install SEO Plugin",
			Action:    "Install Yoast SEO or Rank Math from Plugins → Add New",
			Location:  "WordPress Admin",
			Tip:       "Both plugins automatically generate sitemaps",
		})
		steps = append(steps, Step{
			Number:    3,
			Title:     "Enable XML Sitemaps",
			Action:    "Go to SEO → General → Features → Enable XML Sitemaps",
			Location:  "Yoast SEO settings",
			Tip:       "Your sitemap will be at yourdomain.com/sitemap_index.xml",
		})
	} else {
		steps = append(steps, Step{
			Number:    2,
			Title:     "Generate Sitemap",
			Action:    "Use free online sitemap generator",
			Location:  "https://www.xml-sitemaps.com",
			Code:      "Enter your URL → Click Start → Download sitemap.xml",
			Tip:       "Free for up to 500 pages",
		})
		steps = append(steps, Step{
			Number:    3,
			Title:     "Upload Sitemap",
			Action:    "Upload sitemap.xml to your website root directory",
			Location:  "Via FTP or hosting file manager",
			Tip:       "Root directory is usually public_html or www folder",
		})
	}
	
	steps = append(steps, Step{
		Number:    len(steps) + 1,
		Title:     "Submit to Google",
		Action:    "Go to Google Search Console → Sitemaps → Enter sitemap URL → Submit",
		Location:  "https://search.google.com/search-console",
		Tip:       "This tells Google to crawl your site immediately",
	})
	
	return steps
}

func getAltTextSteps(platform string) []Step {
	return []Step{
		{
			Number:    2,
			Title:     "Find Images Without Alt Text",
			Action:    "Review your images in media library or page editor",
			Location:  "Media library / Page editor",
			Tip:       "Images without alt text appear blank to screen readers",
		},
		{
			Number:    3,
			Title:     "Add Alt Text",
			Action:    "Click on each image and add descriptive alt text",
			Location:  "Image settings panel",
			Code:      `<img src="image.jpg" alt="Describe what the image shows">`,
			Tip:       "Describe naturally, include keywords only if relevant",
		},
	}
}

func getRobotsTxtSteps(platform string) []Step {
	return []Step{
		{
			Number:    2,
			Title:     "Check if robots.txt Exists",
			Action:    "Visit yourdomain.com/robots.txt in browser",
			Location:  "Web browser",
			Tip:       "If you see 404, you need to create one",
		},
		{
			Number:    3,
			Title:     "Create robots.txt",
			Action:    "Create file named 'robots.txt' with basic directives",
			Location:  "Text editor",
			Code:      "User-agent: *\nAllow: /\nDisallow: /admin/\nSitemap: https://yourdomain.com/sitemap.xml",
			Tip:       "Don't block important pages you want indexed",
		},
		{
			Number:    4,
			Title:     "Upload robots.txt",
			Action:    "Upload file to your website root directory",
			Location:  "public_html or www folder",
			Tip:       "Test by visiting yourdomain.com/robots.txt again",
		},
	}
}

// ========== HELPER FUNCTIONS ==========

func getPlatformName(platform string) string {
	names := map[string]string{
		"wordpress":   "WordPress",
		"shopify":     "Shopify",
		"wix":         "Wix",
		"webflow":     "Webflow",
		"squarespace": "Squarespace",
		"html":        "HTML/CSS",
	}
	
	if name, ok := names[platform]; ok {
		return name
	}
	return platform
}

func getLoginURL(platform string) string {
	urls := map[string]string{
		"wordpress":   "yourdomain.com/wp-admin",
		"shopify":     "yourstore.myshopify.com/admin",
		"wix":         "wix.com/dashboard",
		"webflow":     "webflow.com/dashboard",
		"squarespace": "squarespace.com/login",
		"html":        "your FTP or hosting control panel",
	}
	
	if url, ok := urls[platform]; ok {
		return url
	}
	return "your website admin panel"
}

func getExpectedResults(issueID string) []string {
	results := map[string][]string{
		"missing_title_tag": {
			"Google displays your title in search results",
			"Higher click-through rates from search",
			"Better user experience with clear page identification",
		},
		"missing_meta_description": {
			"Google shows your description in search snippets",
			"Improved click-through rates",
			"Better control over search result appearance",
		},
		"no_ssl": {
			"Secure connection for your visitors",
			"Chrome shows padlock icon (trust signal)",
			"Required for modern browsers and ranking",
		},
		"missing_h1": {
			"Clear page structure for search engines",
			"Better content hierarchy understanding",
			"Improved accessibility for screen readers",
		},
		"no_sitemap": {
			"Google can find all your pages",
			"Faster indexing of new content",
			"Better crawl efficiency",
		},
		"missing_alt_text": {
			"Images can rank in Google Images",
			"Better accessibility for visually impaired",
			"Improved page relevance signals",
		},
		"no_robots_txt": {
			"Control over search engine crawling",
			"Prevent indexing of admin pages",
			"Better crawl budget management",
		},
	}
	
	if res, ok := results[issueID]; ok {
		return res
	}
	
	return []string{
		"SEO issue resolved",
		"Better search engine understanding",
		"Improved website quality",
	}
}

// ========== OUTPUT FORMATTERS ==========

func PrintCheckResults(check *WebsiteCheck) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Printf("🔍 SEO ANALYSIS REPORT for %s\n", check.URL)
	fmt.Println(strings.Repeat("=", 70))
	
	fmt.Printf("\n📊 TECHNICAL STATUS:\n")
	fmt.Printf("   • SSL/HTTPS: %s\n", boolToStatus(check.HasSSL))
	fmt.Printf("   • Title Tag: %s\n", boolToStatus(check.HasTitle))
	fmt.Printf("   • Meta Description: %s\n", boolToStatus(check.HasMetaDescription))
	fmt.Printf("   • H1 Heading: %s\n", boolToStatus(check.HasH1))
	fmt.Printf("   • Status Code: %d\n", check.StatusCode)
	
	if len(check.IssuesFound) > 0 {
		fmt.Printf("\n⚠️ ISSUES FOUND (%d):\n", len(check.IssuesFound))
		for i, issueID := range check.IssuesFound {
			for _, issue := range seoIssues {
				if issue.ID == issueID {
					fmt.Printf("   %d. %s [%s]\n", i+1, issue.Name, issue.Priority)
				}
			}
		}
	} else {
		fmt.Println("\n✅ Great! No major SEO issues found!")
	}
	
	fmt.Println("\n💡 RECOMMENDATIONS:")
	if len(check.IssuesFound) > 0 {
		fmt.Println("   Run 'guide <issue_number>' to get fix instructions")
	} else {
		fmt.Println("   • Continue creating quality content")
		fmt.Println("   • Build backlinks from authoritative sites")
		fmt.Println("   • Monitor Google Search Console regularly")
	}
	
	fmt.Println("\n" + strings.Repeat("=", 70))
}

func PrintGuide(guide *Guide) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Printf("📘 SEO GUIDE: %s\n", guide.Issue.Name)
	fmt.Printf("🎯 Platform: %s\n", getPlatformName(guide.Platform))
	fmt.Printf("⚡ Priority: %s | ⏱️ Time: %s\n", guide.Issue.Priority, guide.Issue.TimeToFix)
	fmt.Println(strings.Repeat("=", 70))
	
	fmt.Printf("\n📝 ABOUT:\n%s\n", guide.Issue.Description)
	
	fmt.Println("\n🔧 STEP-BY-STEP INSTRUCTIONS:")
	for _, step := range guide.Steps {
		fmt.Printf("\n   Step %d: %s\n", step.Number, step.Title)
		fmt.Printf("      📍 Action: %s\n", step.Action)
		if step.Location != "" {
			fmt.Printf("      📂 Location: %s\n", step.Location)
		}
		if step.Code != "" {
			fmt.Printf("      💻 Code:\n         %s\n", step.Code)
		}
		if step.Tip != "" {
			fmt.Printf("      💡 Tip: %s\n", step.Tip)
		}
	}
	
	fmt.Println("\n✅ VERIFICATION:")
	for i, v := range guide.Verification {
		fmt.Printf("   %d. %s\n", i+1, v)
	}
	
	fmt.Println("\n🛠️ FREE TOOLS:")
	for _, tool := range guide.Tools {
		fmt.Printf("   • %s\n", tool)
	}
	
	fmt.Println("\n📈 EXPECTED RESULTS:")
	for _, result := range guide.ExpectedResults {
		fmt.Printf("   • %s\n", result)
	}
	
	fmt.Println("\n" + strings.Repeat("=", 70))
}

func boolToStatus(b bool) string {
	if b {
		return "✅ Present"
	}
	return "❌ Missing"
}

func ListAllIssues() {
	fmt.Println("\n📚 COMPLETE SEO ISSUES DATABASE:")
	fmt.Println(strings.Repeat("-", 70))
	
	for i, issue := range seoIssues {
		priorityIcon := "🔴"
		if issue.Priority == "high" {
			priorityIcon = "🟠"
		} else if issue.Priority == "medium" {
			priorityIcon = "🟡"
		} else {
			priorityIcon = "🟢"
		}
		
		fmt.Printf("\n%d. %s %s\n", i+1, priorityIcon, issue.Name)
		fmt.Printf("   📝 %s\n", issue.Description)
		fmt.Printf("   ⏱️ Fix time: %s\n", issue.TimeToFix)
	}
}

// ========== MAIN INTERACTIVE CLI ==========

func main() {
	fmt.Println("🚀 SEO AUTOMATION TOOL")
	fmt.Println("Fix real SEO issues that affect Google rankings")
	fmt.Println(strings.Repeat("=", 50))
	
	// Note: In production, you should get API key from environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  Warning: OPENAI_API_KEY not set. AI features will not work.")
		fmt.Println("   Set it with: export OPENAI_API_KEY='your-key-here'")
	}
	
	logger := log.New(os.Stdout, "[SEO] ", log.LstdFlags)
	generator := NewGenerator(apiKey, logger)
	
	reader := bufio.NewReader(os.Stdin)
	
	for {
		fmt.Println("\n📋 MAIN MENU:")
		fmt.Println("1. Analyze my website (check for SEO issues)")
		fmt.Println("2. Get guide for specific SEO issue")
		fmt.Println("3. List all available SEO issues")
		fmt.Println("4. Get AI-powered SEO recommendations")
		fmt.Println("5. Exit")
		fmt.Print("\nSelect option (1-5): ")
		
		option, _ := reader.ReadString('\n')
		option = strings.TrimSpace(option)
		
		switch option {
		case "1":
			fmt.Print("\nEnter your website URL (e.g., example.com): ")
			url, _ := reader.ReadString('\n')
			url = strings.TrimSpace(url)
			
			if url == "" {
				fmt.Println("❌ Please enter a valid URL")
				continue
			}
			
			fmt.Println("\n🔍 Analyzing website... Please wait.")
			
			check, err := CheckWebsite(url)
			if err != nil {
				fmt.Printf("❌ Error: %v\n", err)
				continue
			}
			
			PrintCheckResults(check)
			
			if len(check.IssuesFound) > 0 {
				fmt.Print("\n📖 Get fix guide for an issue? (Enter number or 'no'): ")
				choice, _ := reader.ReadString('\n')
				choice = strings.TrimSpace(choice)
				
				if choice != "no" && choice != "" {
					var idx int
					fmt.Sscanf(choice, "%d", &idx)
					if idx > 0 && idx <= len(check.IssuesFound) {
						issueID := check.IssuesFound[idx-1]
						guide, err := GenerateGuide(issueID, "wordpress")
						if err == nil {
							PrintGuide(guide)
						}
					}
				}
			}
			
		case "2":
			ListAllIssues()
			
			fmt.Print("\nEnter issue number: ")
			numStr, _ := reader.ReadString('\n')
			var num int
			fmt.Sscanf(strings.TrimSpace(numStr), "%d", &num)
			
			if num < 1 || num > len(seoIssues) {
				fmt.Println("❌ Invalid issue number")
				continue
			}
			
			issue := seoIssues[num-1]
			
			fmt.Print("Platform? (wordpress/shopify/wix/html) [default: wordpress]: ")
			platform, _ := reader.ReadString('\n')
			platform = strings.TrimSpace(strings.ToLower(platform))
			
			if platform == "" {
				platform = "wordpress"
			}
			
			guide, err := GenerateGuide(issue.ID, platform)
			if err != nil {
				fmt.Printf("❌ Error: %v\n", err)
				continue
			}
			
			PrintGuide(guide)
			
		case "3":
			ListAllIssues()
			
		case "4":
			if apiKey == "" {
				fmt.Println("❌ OpenAI API key not configured. Please set OPENAI_API_KEY environment variable.")
				continue
			}
			
			fmt.Print("\nEnter your website URL for AI analysis: ")
			url, _ := reader.ReadString('\n')
			url = strings.TrimSpace(url)
			
			fmt.Print("Platform? (wordpress/shopify/wix/html) [default: wordpress]: ")
			platform, _ := reader.ReadString('\n')
			platform = strings.TrimSpace(strings.ToLower(platform))
			
			if platform == "" {
				platform = "wordpress"
			}
			
			fmt.Println("\n🤖 Analyzing with AI... This may take a moment.")
			
			// Run basic analysis first
			analysis, err := CheckWebsite(url)
			if err != nil {
				fmt.Printf("❌ Error analyzing website: %v\n", err)
				continue
			}
			
			// Convert analysis to string data
			analysisData := fmt.Sprintf("URL: %s, SSL: %v, Issues found: %v", 
				analysis.URL, analysis.HasSSL, analysis.IssuesFound)
			
			// Get AI-powered guide
			aiGuide, err := generator.GenerateGuide("", platform, analysisData)
			if err != nil {
				fmt.Printf("❌ AI Error: %v\n", err)
				continue
			}
			
			fmt.Println("\n🤖 AI-GENERATED SEO RECOMMENDATIONS:")
			fmt.Println(strings.Repeat("=", 70))
			for _, step := range aiGuide.Steps {
				fmt.Printf("\n%s\n", step.Action)
			}
			fmt.Println("\n" + strings.Repeat("=", 70))
			
		case "5":
			fmt.Println("\n✅ Thank you for using SEO Automation Tool!")
			fmt.Println("📈 Implement the fixes and monitor Google Search Console")
			fmt.Println("🚀 Your SEO will improve over time with consistent effort")
			return
			
		default:
			fmt.Println("❌ Invalid option. Please select 1-5")
		}
	}
}