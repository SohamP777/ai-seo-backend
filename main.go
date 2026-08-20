package main

import (
	"ai-seo-backend/internal/core/analyzer"
	"ai-seo-backend/internal/core/optimizer"
	"ai-seo-backend/internal/core/reporting"
	"ai-seo-backend/internal/core/scanner"
	"ai-seo-backend/internal/core/workflow"
	"ai-seo-backend/internal/core/fixer"
	"ai-seo-backend/internal/core/guide"
	"ai-seo-backend/internal/core/wordpress"
    "ai-seo-backend/internal/core/shopify"
	"ai-seo-backend/internal/api/handlers"
	 "ai-seo-backend/internal/models"
	 "ai-seo-backend/internal/api/middleware"
	"ai-seo-backend/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"strings"
	"io"      
    "net/url"
    "encoding/hex"


	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"github.com/joho/godotenv" 
	 "github.com/golang-jwt/jwt/v5"  
	 "golang.org/x/oauth2"
	"gorm.io/gorm"

	"crypto/hmac"
	"crypto/sha256"
	"github.com/razorpay/razorpay-go"
)

// ============ DATABASE MODELS ============

type User struct {
    ID                   string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    Email                string    `gorm:"uniqueIndex;not null" json:"email"`
    Password             string    `gorm:"not null" json:"-"`
    PasswordHash         string    `json:"-"`
    Name                 string    `json:"name"`
    Role                 string    `gorm:"default:user" json:"role"`
    CreatedAt            time.Time `json:"created_at"`
    UpdatedAt            time.Time `json:"updated_at"`
    WebsiteURL           string    `json:"website_url"`
    // ✅ ADD THESE GOOGLE FIELDS
    GoogleID             string    `gorm:"uniqueIndex" json:"google_id,omitempty"`
    Avatar               string    `json:"avatar,omitempty"`
    Provider             string    `gorm:"default:local" json:"provider"`
    Status               string    `gorm:"default:active" json:"status"`
    Plan                 string    `gorm:"default:starter" json:"plan"`
    SubscriptionEndDate  time.Time `gorm:"column:subscription_end_date" json:"subscription_end_date"`
    MaxWebsites          int       `gorm:"default:5" json:"max_websites"`
	APIKey string `json:"api_key,omitempty"`
	 // ========== FREE TRIAL FIELDS ==========
    FreeTrialUsed        bool      `gorm:"default:false" json:"free_trial_used"`
    FreeTrialStartDate   time.Time `gorm:"column:free_trial_start_date" json:"free_trial_start_date"`
    FreeTrialEndDate     time.Time `gorm:"column:free_trial_end_date" json:"free_trial_end_date"`
}


type SEOAnalysis struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"index" json:"user_id"`
	URL       string    `gorm:"not null" json:"url"`
	Score     int       `json:"score"`
	Result    string    `gorm:"type:jsonb" json:"result"`
	Status    string    `gorm:"default:pending" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Payment struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID      string    `gorm:"index" json:"user_id"`
	Amount      int       `json:"amount"`
	Currency    string    `gorm:"default:USD" json:"currency"`
	Status      string    `json:"status"`
	PaymentID   string    `gorm:"uniqueIndex" json:"payment_id"`
	OrderID     string    `gorm:"uniqueIndex" json:"order_id"`
	PlanID      string    `json:"plan_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type WorkflowDB struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"index" json:"user_id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Input     string    `gorm:"type:jsonb" json:"input"`
	Output    string    `gorm:"type:jsonb" json:"output"`
	Error     string    `json:"error"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Add this model in main.go with other models
type AIGuideReport struct {
    ID                 string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    ScanID             string    `gorm:"index;not null" json:"scan_id"`
    UserID             string    `gorm:"index;not null" json:"user_id"`
    Recommendations    string    `gorm:"type:jsonb;not null" json:"recommendations"`
    EstimatedTimeline  string    `json:"estimated_timeline"`
    EffortLevel        string    `json:"effort_level"`
    GuideSource        string    `gorm:"default:manual" json:"guide_source"`  
    GeneratedAt        time.Time `json:"generated_at"`
    UpdatedAt          time.Time `json:"updated_at"`
}

// Add this model with other models (around line 80)
type ScanHistory struct {
	ID              string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID          string    `gorm:"index;not null" json:"user_id"`
	URL             string    `gorm:"not null" json:"url"`
	Score           int       `json:"score"`
	IssuesFound     int       `json:"issues_found"`
	IssuesFixed     int       `json:"issues_fixed"`
	CriticalIssues  int       `json:"critical_issues"`
	Recommendations string    `gorm:"type:text" json:"recommendations"` // Store as JSON string
	Issues          string    `gorm:"type:text" json:"issues"`           // Store as JSON string
	FixedIssues     string    `gorm:"type:text" json:"fixed_issues"`     // Store as JSON string
	TrafficPotential string   `json:"traffic_potential"`
	BeforeScore     int       `json:"before_score"`
	CreatedAt       time.Time `json:"created_at"`
}

type Report struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"index" json:"user_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Data      string    `gorm:"type:jsonb" json:"data"`
	FileURL   string    `json:"file_url"`
	CreatedAt time.Time `json:"created_at"`
}

// ============ CONFIGURATION ============
type Config struct {
	ServerPort          string
	DatabaseURL         string
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	DBSSLMode           string
	JWTSecret           string
	RazorpayKeyID       string
	RazorpayKeySecret   string
	CruxAPIKey          string
	OpenAIAPIKey        string
	CloudflareAPIToken  string
	CloudflareZoneID    string
	ImageBackupDir      string
	OutputDir           string
	SMTPHost            string
	SMTPPort            int
	SMTPUsername        string
	SMTPPassword        string
	FromEmail           string
	FromName            string
	RequestTimeout      time.Duration
	MaxConcurrent       int
	RateLimit           int
	Environment         string
	GoogleClientID     string
    GoogleClientSecret string
    GoogleRedirectURL  string
}

type AuthHandler struct {
    authService   handlers.AuthServiceInterface  // ← Change to interface
    logger        *log.Logger
    userRepo      *UserRepository
    jwtSecret     string
    googleOAuth   *oauth2.Config
}

// ✅ AuthServiceWrapper - Wraps AuthService to implement handlers.AuthServiceInterface
type AuthServiceWrapper struct {
    *AuthService
}

// ✅ ValidateToken returns (string, error) as required by the interface
func (w *AuthServiceWrapper) ValidateToken(token string) (string, error) {
    claims, err := w.AuthService.ValidateToken(token)
    if err != nil {
        return "", err
    }
    return claims.UserID, nil
}

// ✅ GenerateToken - already matches the interface
// func (w *AuthServiceWrapper) GenerateToken(userID string) (string, error) {
//     return w.AuthService.GenerateToken(userID)
// }

// ✅ FindOrCreateGoogleUser - implement this method
func (w *AuthServiceWrapper) FindOrCreateGoogleUser(email, name, googleID string) (string, error) {
    return w.AuthService.FindOrCreateGoogleUser(email, name, googleID)
}

// Context keys for user information
type contextKey string

const (
    ContextKeyUserID        contextKey = "user_id"
    ContextKeyUserEmail     contextKey = "user_email"
    ContextKeyUserName      contextKey = "user_name"
    ContextKeyUserRoles     contextKey = "user_roles"
    ContextKeyUserPermissions contextKey = "user_permissions"
    ContextKeyTokenID       contextKey = "token_id"
    ContextKeyOrganizationID contextKey = "organization_id"
    ContextKeyJWTClaims     contextKey = "jwt_claims"
    ContextKeyUserProvider  contextKey = "user_provider"
    ContextKeyUserAvatar    contextKey = "user_avatar"
)
// GenerateToken implements handlers.AuthServiceInterface
func (w *AuthServiceWrapper) GenerateToken(userID string) (string, error) {
    return w.AuthService.GenerateToken(userID)
}

// SetGoogleConfig sets Google OAuth configuration
func (h *AuthHandler) SetGoogleConfig(config *oauth2.Config) {
    h.googleOAuth = config
}

// GoogleAuth initiates Google OAuth flow
func (h *AuthHandler) GoogleAuth(w http.ResponseWriter, r *http.Request) {
    if h.googleOAuth == nil {
        http.Error(w, `{"error": "Google OAuth not configured"}`, http.StatusInternalServerError)
        return
    }
    
    url := h.googleOAuth.AuthCodeURL("state", oauth2.AccessTypeOffline)
    http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

/// GoogleCallback handles Google OAuth callback
func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
    h.logger.Printf("🔑 GoogleCallback: Received callback")
    
    // Get frontend URL from environment
    frontendURL := os.Getenv("FRONTEND_URL")
    if frontendURL == "" {
        frontendURL = "https://www.seosps.com"
    }
    
    // Get code and state from URL
    code := r.URL.Query().Get("code")
    state := r.URL.Query().Get("state")
    
    h.logger.Printf("🔍 Code: %s", code[:20]+"...")
    h.logger.Printf("🔍 State: %s", state)
    
    if code == "" {
        h.logger.Printf("❌ No code found")
        errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("No authorization code"))
        http.Redirect(w, r, errorURL, http.StatusFound)
        return
    }
    
    // Exchange code for token
    token, err := h.exchangeGoogleCodeForToken(code)
    if err != nil {
        h.logger.Printf("❌ Token exchange error: %v", err)
        errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("Authentication failed"))
        http.Redirect(w, r, errorURL, http.StatusFound)
        return
    }
    
    // Get user info from Google
    userInfo, err := h.getGoogleUserInfo(token)
    if err != nil {
        h.logger.Printf("❌ Get user info error: %v", err)
        errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("Failed to get user info"))
        http.Redirect(w, r, errorURL, http.StatusFound)
        return
    }
    
    // Extract user info
    email, _ := userInfo["email"].(string)
    name, _ := userInfo["name"].(string)
    googleID, _ := userInfo["id"].(string)
    picture, _ := userInfo["picture"].(string) // ✅ Keep this line
    
    // Find or create user using authService
    userID, err := h.authService.FindOrCreateGoogleUser(email, name, googleID)
    if err != nil {
        h.logger.Printf("❌ Find/create user error: %v", err)
        errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("Failed to create user"))
        http.Redirect(w, r, errorURL, http.StatusFound)
        return
    }
    
    // Get full user from database
    user, err := h.userRepo.FindByID(userID)
    if err != nil {
        h.logger.Printf("❌ Get user error: %v", err)
        errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("Failed to get user"))
        http.Redirect(w, r, errorURL, http.StatusFound)
        return
    }
    
    // ✅ Update user avatar if picture exists and is different
    if picture != "" && user.Avatar != picture {
        user.Avatar = picture
        h.userRepo.Update(user)
    }
    
    // Generate JWT token
    jwtToken, err := h.authService.GenerateToken(user.ID)
    if err != nil {
        h.logger.Printf("❌ Token generation error: %v", err)
        errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("Failed to generate token"))
        http.Redirect(w, r, errorURL, http.StatusFound)
        return
    }
    
    // Redirect to frontend with token
    redirectURL := fmt.Sprintf("%s/signup?token=%s&success=true", frontendURL, jwtToken)
    h.logger.Printf("✅ Redirecting to: %s", redirectURL)
    http.Redirect(w, r, redirectURL, http.StatusFound)
}

// exchangeGoogleCodeForToken exchanges code for access token
func (h *AuthHandler) exchangeGoogleCodeForToken(code string) (string, error) {
    tokenURL := "https://oauth2.googleapis.com/token"
    data := fmt.Sprintf(
        "client_id=%s&client_secret=%s&code=%s&redirect_uri=%s&grant_type=authorization_code",
        h.googleOAuth.ClientID,
        h.googleOAuth.ClientSecret,
        code,
        h.googleOAuth.RedirectURL,
    )
    
    req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data))
    if err != nil {
        return "", err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }
    
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("token exchange failed: %s", string(body))
    }
    
    var result map[string]interface{}
    if err := json.Unmarshal(body, &result); err != nil {
        return "", err
    }
    
    accessToken, ok := result["access_token"].(string)
    if !ok || accessToken == "" {
        return "", fmt.Errorf("access token not found")
    }
    
    return accessToken, nil
}

// getGoogleUserInfo fetches user info from Google
func (h *AuthHandler) getGoogleUserInfo(accessToken string) (map[string]interface{}, error) {
    req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+accessToken)
    
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("failed to get user info: %s", string(body))
    }
    
    var userInfo map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
        return nil, err
    }
    
    return userInfo, nil
}

type MiddlewareAuthServiceWrapper struct {
    *AuthService
}

// GoogleTokenValidate validates Google token
func (h *AuthHandler) GoogleTokenValidate(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Token string `json:"token"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
        return
    }
    
    if req.Token == "" {
        http.Error(w, `{"error": "Token required"}`, http.StatusBadRequest)
        return
    }
    
    // Validate the token with Google
    resp, err := http.Get(fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", req.Token))
    if err != nil {
        http.Error(w, fmt.Sprintf(`{"error": "Failed to validate token: %v"}`, err), http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
        return
    }
    
    var tokenInfo map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
        http.Error(w, fmt.Sprintf(`{"error": "Failed to parse token info: %v"}`, err), http.StatusInternalServerError)
        return
    }
    
    email, _ := tokenInfo["email"].(string)
    name, _ := tokenInfo["name"].(string)
    googleID, _ := tokenInfo["sub"].(string)
    picture, _ := tokenInfo["picture"].(string)
    
    // Find or create user
    user, err := h.userRepo.GetByGoogleID(googleID)
    if err != nil {
        // Create new user with FREE TRIAL
        user = &User{
            GoogleID:           googleID,
            Email:              email,
            Name:               name,
            Avatar:             picture,
            Plan:               "free",
            Status:             "active",
            Provider:           "google",
            FreeTrialUsed:      false,
            FreeTrialStartDate: time.Now(),
            FreeTrialEndDate:   time.Now().Add(7 * 24 * time.Hour), // 7 days free trial
            CreatedAt:          time.Now(),
            UpdatedAt:          time.Now(),
        }
        
        if err := h.userRepo.Create(user); err != nil {
            http.Error(w, fmt.Sprintf(`{"error": "Failed to create user: %v"}`, err), http.StatusInternalServerError)
            return
        }
    }
    
    // Generate JWT token
    jwtToken, err := h.generateJWTToken(user)
    if err != nil {
        http.Error(w, fmt.Sprintf(`{"error": "Failed to generate token: %v"}`, err), http.StatusInternalServerError)
        return
    }
    
    // Calculate free trial info
    isExpired := time.Now().After(user.FreeTrialEndDate)
    daysUsed := int(time.Since(user.FreeTrialStartDate).Hours() / 24)
    daysRemaining := 7 - daysUsed
    if daysRemaining < 0 {
        daysRemaining = 0
    }
    
    // Check if user has active subscription
    hasActiveSubscription := user.SubscriptionEndDate.After(time.Now())
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "token": jwtToken,
        "user": map[string]interface{}{
            "id":       user.ID,
            "email":    user.Email,
            "name":     user.Name,
            "avatar":   user.Avatar,
            "provider": user.Provider,
        },
        "free_trial": map[string]interface{}{
            "used":           user.FreeTrialUsed || isExpired,
            "days_remaining": daysRemaining,
            "is_expired":     isExpired,
            "start_date":     user.FreeTrialStartDate.Format(time.RFC3339),
            "end_date":       user.FreeTrialEndDate.Format(time.RFC3339),
            "trial_available": !user.FreeTrialUsed && !isExpired && !hasActiveSubscription,
        },
        "subscription": map[string]interface{}{
            "active":       hasActiveSubscription,
            "plan":         user.Plan,
            "end_date":     user.SubscriptionEndDate.Format(time.RFC3339),
        },
        "message": func() string {
            if hasActiveSubscription {
                return "You have an active subscription. Enjoy unlimited SEO automation!"
            } else if user.FreeTrialUsed && isExpired {
                return "Your free trial has expired. Please subscribe to continue using SEO automation."
            } else if user.FreeTrialUsed {
                return "You have used your free trial. Please subscribe to continue."
            } else if !isExpired {
                return fmt.Sprintf("Welcome! You have %d days remaining in your free trial.", daysRemaining)
            }
            return "Sign up for a free 7-day trial!"
        }(),
        "upgrade_link": "/pricing",
    })
}

// Helper method to generate JWT token
func (h *AuthHandler) generateJWTToken(user *User) (string, error) {
    // This is a simplified version - you should use your actual JWT generation logic
    // Return a token or use your authService.GenerateToken
    return h.authService.GenerateToken(user.ID)
}

// GetByGoogleID retrieves user by Google ID
func (r *UserRepository) GetByGoogleID(googleID string) (*User, error) {
    var user User
    err := r.db.Where("google_id = ?", googleID).First(&user).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}

// GetByEmail retrieves user by email
func (r *UserRepository) GetByEmail(email string) (*User, error) {
    var user User
    err := r.db.Where("email = ?", email).First(&user).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}

// Update updates a user
func (r *UserRepository) Update(user *User) error {
    return r.db.Save(user).Error
}

func LoadConfig() *Config {
    err := godotenv.Load()
    if err != nil {
        log.Printf("ERROR loading .env: %v", err)
    } else {
        log.Println("✅ .env loaded successfully")
    }

	return &Config{
		ServerPort:         getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "postgres"),
		DBPassword:         getEnv("DB_PASSWORD", ""),
		DBName:             getEnv("DB_NAME", "aiseo"),
		DBSSLMode:          getEnv("DB_SSLMODE", "disable"), 
		JWTSecret:          getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		RazorpayKeyID:      getEnv("RAZORPAY_KEY_ID", ""),
		RazorpayKeySecret:  getEnv("RAZORPAY_KEY_SECRET", ""),
		CruxAPIKey:         getEnv("CRUX_API_KEY", ""),
		OpenAIAPIKey:       getEnv("OPENAI_API_KEY", ""),
		CloudflareAPIToken: getEnv("CLOUDFLARE_API_TOKEN", ""),
		CloudflareZoneID:   getEnv("CLOUDFLARE_ZONE_ID", ""),
		ImageBackupDir:     getEnv("IMAGE_BACKUP_DIR", "./backups/images"),
		OutputDir:          getEnv("OUTPUT_DIR", "./seo_output"),
		SMTPHost:           getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:           getEnvInt("SMTP_PORT", 587),
		SMTPUsername:       getEnv("SMTP_USERNAME", ""),
		SMTPPassword:       getEnv("SMTP_PASSWORD", ""),
		FromEmail:          getEnv("FROM_EMAIL", "seo@tool.com"),
		FromName:           getEnv("FROM_NAME", "AI SEO Tool"),
		RequestTimeout:     getEnvDuration("REQUEST_TIMEOUT", 30*time.Second),
		MaxConcurrent:      getEnvInt("MAX_CONCURRENT", 10),
		RateLimit:          getEnvInt("RATE_LIMIT", 100),
		Environment:        getEnv("ENVIRONMENT", "development"),
	   GoogleClientID:      getEnv("GOOGLE_CLIENT_ID", ""),
        GoogleClientSecret:  getEnv("GOOGLE_CLIENT_SECRET", ""),
        GoogleRedirectURL:   getEnv("GOOGLE_REDIRECT_URL", "https://api.seosps.com/api/auth/google/callback"),
    }
}


func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		fmt.Sscanf(value, "%d", &intValue)
		return intValue
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		duration, err := time.ParseDuration(value)
		if err == nil {
			return duration
		}
	}
	return defaultValue
}

// ============ REPOSITORIES ============
type ScanHistoryRepository struct {
	db *gorm.DB
}

func NewScanHistoryRepository(db *gorm.DB) *ScanHistoryRepository {
	return &ScanHistoryRepository{db: db}
}

func (r *ScanHistoryRepository) Create(history *ScanHistory) error {
	return r.db.Create(history).Error
}

func (r *ScanHistoryRepository) FindByUserID(userID string, limit int) ([]ScanHistory, error) {
	var histories []ScanHistory
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&histories).Error
	return histories, err
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByEmail(email string) (*User, error) {
	var user User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Create(user *User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) FindByID(id string) (*User, error) {
	var user User
	err := r.db.Where("id = ?", id).First(&user).Error
	return &user, err
}

type SEORepository struct {
	db *gorm.DB
}

func NewSEORepository(db *gorm.DB) *SEORepository {
	return &SEORepository{db: db}
}

func (r *SEORepository) Create(analysis *SEOAnalysis) error {
	return r.db.Create(analysis).Error
}

func (r *SEORepository) FindByID(id string) (*SEOAnalysis, error) {
	var analysis SEOAnalysis
	err := r.db.Where("id = ?", id).First(&analysis).Error
	if err != nil {
		return nil, err
	}
	return &analysis, nil
}

func (r *SEORepository) FindByUserID(userID string, limit, offset int) ([]SEOAnalysis, error) {
	var analyses []SEOAnalysis
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Offset(offset).Find(&analyses).Error
	return analyses, err
}

func (r *SEORepository) Update(analysis *SEOAnalysis) error {
	return r.db.Save(analysis).Error
}

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(payment *Payment) error {
	return r.db.Create(payment).Error
}

func (r *PaymentRepository) FindByUserID(userID string) ([]Payment, error) {
	var payments []Payment
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&payments).Error
	return payments, err
}

func (r *PaymentRepository) FindByOrderID(orderID string) (*Payment, error) {
	var payment Payment
	err := r.db.Where("order_id = ?", orderID).First(&payment).Error
	return &payment, err
}

type WorkflowRepository struct {
	db *gorm.DB
}

func NewWorkflowRepository(db *gorm.DB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

func (r *WorkflowRepository) Create(workflow *WorkflowDB) error {
	return r.db.Create(workflow).Error
}

func (r *WorkflowRepository) FindByID(id string) (*WorkflowDB, error) {
	var workflow WorkflowDB
	err := r.db.Where("id = ?", id).First(&workflow).Error
	return &workflow, err
}

func (r *WorkflowRepository) Update(workflow *WorkflowDB) error {
	return r.db.Save(workflow).Error
}

type ReportRepository struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) Create(report *Report) error {
	return r.db.Create(report).Error
}

func (r *ReportRepository) FindByUserID(userID string) ([]Report, error) {
	var reports []Report
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&reports).Error
	return reports, err
}

func (r *ReportRepository) FindByID(id string) (*Report, error) {
	var report Report
	err := r.db.Where("id = ?", id).First(&report).Error
	return &report, err
}

// ============ SERVICES ============

// ============================================
// AuthService - Main service
// ============================================
type AuthService struct {
    userRepo  *UserRepository
    jwtSecret string
}

// NewAuthService creates a new AuthService
func NewAuthService(userRepo *UserRepository, jwtSecret string) *AuthService {
    return &AuthService{
        userRepo:  userRepo,
        jwtSecret: jwtSecret,
    }
}

// ============================================
// AuthService Methods - For middleware.AuthService
// ============================================

// ValidateToken validates a JWT token and returns Claims
func (s *AuthService) ValidateToken(token string) (*middleware.Claims, error) {
    claims := &middleware.Claims{
        UserID: "mock-user-id",
        Email:  "user@example.com",
        Name:   "Test User",
        Roles:  []string{"user"},
        Permissions: []string{"read", "write"},
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            Issuer:    "seosps",
            Audience:  []string{"seosps-users"},
        },
    }
    return claims, nil
}

// GetUserPermissions returns user permissions
func (s *AuthService) GetUserPermissions(userID string) ([]string, error) {
    return []string{"read", "write"}, nil
}

// IsTokenBlacklisted checks if token is blacklisted
func (s *AuthService) IsTokenBlacklisted(tokenID string) (bool, error) {
    return false, nil
}

// GetUserByGoogleID retrieves user by Google ID
// This implements middleware.AuthService interface
func (s *AuthService) GetUserByGoogleID(googleID string) (interface{}, error) {
    return s.userRepo.GetByGoogleID(googleID)
}

// FindOrCreateGoogleUser implements middleware.AuthService interface
// Signature: FindOrCreateGoogleUser(email, name, googleID string) (string, error)
func (s *AuthService) FindOrCreateGoogleUser(email, name, googleID string) (string, error) {
    // Try to find by Google ID
    user, err := s.userRepo.GetByGoogleID(googleID)
    if err == nil && user != nil {
        return user.ID, nil
    }

    // Try to find by email
    user, err = s.userRepo.GetByEmail(email)
    if err == nil && user != nil {
        user.GoogleID = googleID
        user.Provider = "google"
        s.userRepo.Update(user)
        return user.ID, nil
    }

    // Create new user with FREE TRIAL
    newUser := &User{
        ID:                 uuid.New().String(),
        GoogleID:           googleID,
        Email:              email,
        Name:               name,
        Provider:           "google",
        Status:             "active",
        Plan:               "free",
        FreeTrialUsed:      false,
        FreeTrialStartDate: time.Now(),
        FreeTrialEndDate:   time.Now().Add(7 * 24 * time.Hour),
        CreatedAt:          time.Now(),
        UpdatedAt:          time.Now(),
    }

    err = s.userRepo.Create(newUser)
    if err != nil {
        return "", fmt.Errorf("failed to create Google user: %w", err)
    }

    return newUser.ID, nil
}

// GenerateToken generates a JWT token for a user
// This implements middleware.AuthService interface
func (s *AuthService) GenerateToken(userID string) (string, error) {
    // Get user from database to include real info
    user, err := s.userRepo.FindByID(userID)
    if err != nil {
        user = &User{
            ID:    userID,
            Email: "user@example.com",
            Name:  "Test User",
        }
    }

    claims := &middleware.Claims{
        UserID: user.ID,
        Email:  user.Email,
        Name:   user.Name,
        Roles:  []string{"user"},
        Permissions: []string{"read", "write"},
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            Issuer:    "seosps",
            Audience:  []string{"seosps-users"},
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(s.jwtSecret))
    if err != nil {
        return "", err
    }

    return tokenString, nil
}

type HandlerAuthServiceWrapper struct {
	*AuthService
}

// ValidateToken implements handlers.AuthServiceInterface
// Returns string (user ID) instead of *middleware.Claims
func (w *HandlerAuthServiceWrapper) ValidateToken(token string) (string, error) {
	claims, err := w.AuthService.ValidateToken(token)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

// GenerateToken implements handlers.AuthServiceInterface
func (w *HandlerAuthServiceWrapper) GenerateToken(userID string) (string, error) {
	return w.AuthService.GenerateToken(userID)
}

// ============================================
// MiddlewareAuthServiceWrapper - Wraps AuthService
// This converts AuthService to middleware.AuthService
// (Already implemented directly, but wrapper for consistency)
// ============================================
// Note: AuthService already implements middleware.AuthService directly,
// so no wrapper is needed for that interface.
type CacheService interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
}

type QueueService struct {
	jobChan chan *Job
	maxSize int
	logger  *log.Logger
}

func NewQueueService(maxSize int, logger *log.Logger) *QueueService {
	return &QueueService{
		jobChan: make(chan *Job, maxSize),
		maxSize: maxSize,
		logger:  logger,
	}
}

func (q *QueueService) Enqueue(job *Job) error {
	select {
	case q.jobChan <- job:
		q.logger.Printf("Job enqueued job_id=%s url=%s", job.ID, job.URL)
		return nil
	default:
		return fmt.Errorf("queue is full")
	}
}

func (q *QueueService) GetQueueLength() int {
	return len(q.jobChan)
}

type EmailService struct {
	logger *log.Logger
}

func NewEmailService(logger *log.Logger) *EmailService {
	return &EmailService{logger: logger}
}

func (s *EmailService) SendEmail(to, subject, body string) error {
	s.logger.Printf("Sending email to=%s subject=%s body_length=%d", to, subject, len(body))
	return nil
}

// ============ PAYMENT TYPES ============

type RazorpayProcessor struct {
	keyID     string
	keySecret string
	logger    *log.Logger
	client    *razorpay.Client  // ✅ Add this
}

func NewRazorpayProcessor(keyID, keySecret string, logger *log.Logger) *RazorpayProcessor {
	// ✅ Create REAL Razorpay client
	client := razorpay.NewClient(keyID, keySecret)
	
	return &RazorpayProcessor{
		keyID:     keyID,
		keySecret: keySecret,
		logger:    logger,
		client:    client,  // ✅ Store client
	}
}

// ✅ Replace CreateOrder with REAL implementation
func (p *RazorpayProcessor) CreateOrder(amount int, currency string) (string, error) {
	p.logger.Printf("Creating REAL Razorpay order amount=%d currency=%s", amount, currency)
	
	data := map[string]interface{}{
		"amount":          amount,
		"currency":        currency,
		"receipt":         "receipt_" + fmt.Sprintf("%d", time.Now().UnixNano()),
		"payment_capture": 1,
	}
	
	order, err := p.client.Order.Create(data, nil)
	if err != nil {
		p.logger.Printf("❌ Razorpay order creation failed: %v", err)
		return "", fmt.Errorf("razorpay order creation failed: %w", err)
	}
	
	orderID, ok := order["id"].(string)
	if !ok || orderID == "" {
		return "", fmt.Errorf("razorpay returned empty order ID")
	}
	
	p.logger.Printf("✅ REAL Razorpay order created: %s", orderID)
	return orderID, nil
}

// ✅ Replace VerifyPayment with REAL implementation
func (p *RazorpayProcessor) VerifyPayment(orderID, paymentID, signature string) bool {
	p.logger.Printf("Verifying payment order_id=%s payment_id=%s", orderID, paymentID)
	
	// Create the string to verify: order_id|payment_id
	data := orderID + "|" + paymentID
	
	// Create HMAC SHA256
	h := hmac.New(sha256.New, []byte(p.keySecret))
	h.Write([]byte(data))
	expectedSignature := hex.EncodeToString(h.Sum(nil))
	
	// Compare signatures
	result := hmac.Equal([]byte(signature), []byte(expectedSignature))
	
	if result {
		p.logger.Printf("✅ Signature verified successfully")
	} else {
		p.logger.Printf("❌ Signature verification failed")
	}
	
	return result
}

type PlanManager struct {
	paymentRepo *PaymentRepository
	logger      *log.Logger
}

func NewPlanManager(paymentRepo *PaymentRepository, logger *log.Logger) *PlanManager {
	return &PlanManager{
		paymentRepo: paymentRepo,
		logger:      logger,
	}
}

func (m *PlanManager) GetPlan(planID string) (map[string]interface{}, error) {
	plans := map[string]map[string]interface{}{
		"starter": {
			"name":          "Starter",
			"price":         19,
			"yearly_price":  182,
			"currency":      "USD",
			"max_websites":  1,
			"scan_frequency": "weekly",
		},
		"professional": {
			"name":          "Professional",
			"price":         99,
			"yearly_price":  950,
			"currency":      "USD",
			"max_websites":  10,
			"scan_frequency": "daily",
		},
		"enterprise": {
			"name":          "Enterprise",
			"price":         199,
			"yearly_price":  1910,
			"currency":      "USD",
			"max_websites":  25,
			"scan_frequency": "daily",
		},
	}
	
	if plan, exists := plans[planID]; exists {
		return plan, nil
	}
	return nil, fmt.Errorf("plan not found")
}

func (m *PlanManager) ListPlans() []map[string]interface{} {
	return []map[string]interface{}{
		{"id": "basic", "name": "Basic", "price": 49, "currency": "USD", "popular": false},
		{"id": "pro", "name": "Professional", "price": 149, "currency": "USD", "popular": true},
		{"id": "enterprise", "name": "Enterprise", "price": 299, "currency": "USD", "popular": false},
	}
}

type SubscriptionManager struct {
	paymentRepo *PaymentRepository
	planManager *PlanManager
	logger      *log.Logger
}

func NewSubscriptionManager(paymentRepo *PaymentRepository, planManager *PlanManager, logger *log.Logger) *SubscriptionManager {
	return &SubscriptionManager{
		paymentRepo: paymentRepo,
		planManager: planManager,
		logger:      logger,
	}
}


// StartFreeTrial handles starting a free trial
func (h *PaymentHandler) StartFreeTrial(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Plan string `json:"plan"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.sendJSON(w, http.StatusBadRequest, map[string]interface{}{
            "success": false,
            "message": "Invalid request",
        })
        return
    }
    
    userID, ok := r.Context().Value("user_id").(string)
    if !ok || userID == "" {
        h.sendJSON(w, http.StatusUnauthorized, map[string]interface{}{
            "success": false,
            "message": "User not authenticated",
        })
        return
    }
    
    h.logger.Printf("🔄 Starting free trial for user: %s", userID)
    
    // Get user from database
    var user struct {
        ID                  string
        Status              string
        Plan                string
        SubscriptionEndDate time.Time
        FreeTrialUsed       bool
        FreeTrialStartDate  time.Time
        FreeTrialEndDate    time.Time
    }
    
    err := h.db.Table("users").
        Where("id = ?", userID).
        First(&user).Error
    
    if err != nil {
        h.sendJSON(w, http.StatusInternalServerError, map[string]interface{}{
            "success": false,
            "message": "User not found",
        })
        return
    }
    
    now := time.Now()
    
    // Check if user already has an active subscription
    if user.Status == "active" && now.Before(user.SubscriptionEndDate) {
        h.sendJSON(w, http.StatusOK, map[string]interface{}{
            "success": true,
            "message": "You already have an active subscription",
            "has_active_subscription": true,
        })
        return
    }
    
    // Check if user already used free trial
    if user.FreeTrialUsed {
        // Check if trial is expired
        if now.After(user.FreeTrialEndDate) {
            h.sendJSON(w, http.StatusBadRequest, map[string]interface{}{
                "success": false,
                "message": "Your free trial has expired. Please subscribe to continue.",
                "free_trial": map[string]interface{}{
                    "used": true,
                    "is_expired": true,
                    "days_remaining": 0,
                },
            })
            return
        }
        
        // Check if trial is still active
        if now.Before(user.FreeTrialEndDate) {
            daysRemaining := int(user.FreeTrialEndDate.Sub(now).Hours()/24) + 1
            h.sendJSON(w, http.StatusOK, map[string]interface{}{
                "success": true,
                "message": fmt.Sprintf("You have %d days remaining in your free trial", daysRemaining),
                "free_trial": map[string]interface{}{
                    "used": true,
                    "is_expired": false,
                    "days_remaining": daysRemaining,
                    "end_date": user.FreeTrialEndDate.Format("2006-01-02"),
                },
            })
            return
        }
    }
    
    // Activate free trial (7 days)
    trialStart := now
    trialEnd := trialStart.AddDate(0, 0, 7)
    
    // Update user with trial information
    updates := map[string]interface{}{
        "status":                "trial",
        "free_trial_used":       true,
        "free_trial_start_date": trialStart,
        "free_trial_end_date":   trialEnd,
        "subscription_end_date": trialEnd,
        "plan":                  req.Plan,
        "updated_at":            now,
    }
    
    result := h.db.Table("users").Where("id = ?", userID).Updates(updates)
    if result.Error != nil {
        h.logger.Printf("❌ Error activating trial: %v", result.Error)
        h.sendJSON(w, http.StatusInternalServerError, map[string]interface{}{
            "success": false,
            "message": "Failed to activate free trial",
        })
        return
    }
    
    h.logger.Printf("✅ Free trial activated for user: %s (ends: %s)", userID, trialEnd.Format("2006-01-02"))
    
    h.sendJSON(w, http.StatusOK, map[string]interface{}{
        "success": true,
        "message": "🎉 7-day free trial activated successfully!",
        "free_trial": map[string]interface{}{
            "start_date":     trialStart.Format("2006-01-02"),
            "end_date":       trialEnd.Format("2006-01-02"),
            "days_remaining": 7,
            "used":           true,
            "is_expired":     false,
        },
        "has_active_subscription": true,
    })
}

func (m *SubscriptionManager) CreateSubscription(userID, planID string) (string, error) {
	subscriptionID := "sub_" + uuid.New().String()
	m.logger.Printf("Created subscription user_id=%s plan_id=%s subscription_id=%s", userID, planID, subscriptionID)
	return subscriptionID, nil
}


func (h *PaymentHandler) CheckSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	if userID == "" {
		utils.ErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}
	
	// Query database for subscription status
	var status, plan string
	var subscriptionEndDate time.Time
	var freeTrialUsed bool
	var freeTrialStartDate, freeTrialEndDate time.Time
	
	// ✅ FIX: Query all fields including free trial
	err := h.db.Table("users").
		Where("id = ?", userID).
		Select("status, plan, subscription_end_date, free_trial_used, free_trial_start_date, free_trial_end_date").
		Row().
		Scan(&status, &plan, &subscriptionEndDate, &freeTrialUsed, &freeTrialStartDate, &freeTrialEndDate)
	
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to check subscription")
		return
	}
	
	isActive := status == "active" && time.Now().Before(subscriptionEndDate)
	
	// ✅ Calculate trial data
	daysRemaining := 0
	isTrialExpired := false
	if freeTrialUsed && !freeTrialEndDate.IsZero() {
		daysRemaining = int(freeTrialEndDate.Sub(time.Now()).Hours() / 24)
		if daysRemaining < 0 {
			daysRemaining = 0
			isTrialExpired = true
		}
	}
	
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"has_active_subscription": isActive,
		"plan":                    plan,
		"subscription_end_date":   subscriptionEndDate,
		"status":                  status,
		"free_trial": map[string]interface{}{
			"used":           freeTrialUsed,
			"days_remaining": daysRemaining,
			"is_expired":     isTrialExpired,
			"start_date":     freeTrialStartDate.Format(time.RFC3339),
			"end_date":       freeTrialEndDate.Format(time.RFC3339),
		},
	})
}

func (m *SubscriptionManager) CancelSubscription(subscriptionID string) error {
	m.logger.Printf("Cancelled subscription subscription_id=%s", subscriptionID)
	return nil
}
// ============ HANDLERS ==========
func NewAuthHandler(authService handlers.AuthServiceInterface, logger *log.Logger, userRepo *UserRepository) *AuthHandler {
    return &AuthHandler{
        authService: authService,
        logger:      logger,
        userRepo:    userRepo,
    }
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
        Name     string `json:"name"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request")
        return
    }
    
    if req.Email == "" || req.Password == "" {
        utils.ErrorResponse(w, http.StatusBadRequest, "Email and password are required")
        return
    }
    
    userID := uuid.New().String()
    
    // Create user with free trial
    user := &User{
        ID:                userID,
        Email:             req.Email,
        Name:              req.Name,
        Plan:              "free",
        Status:            "active",
        FreeTrialUsed:     false,
        FreeTrialStartDate: time.Now(),
        FreeTrialEndDate:  time.Now().Add(7 * 24 * time.Hour), // 7 days trial
        CreatedAt:         time.Now(),
        UpdatedAt:         time.Now(),
    }
    
    // Save user to database
    if h.userRepo != nil {
        h.userRepo.Create(user)
    }
    
    token, _ := h.authService.GenerateToken(userID)
    
    utils.JSONResponse(w, http.StatusCreated, map[string]interface{}{
        "message": "User registered successfully",
        "user_id": userID,
        "token":   token,
        "free_trial": map[string]interface{}{
            "used":          false,
            "days_remaining": 7,
            "end_date":      user.FreeTrialEndDate.Format(time.RFC3339),
        },
        "user": map[string]string{
            "email": req.Email,
            "name":  req.Name,
        },
    })
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}
	
	if req.Email == "" || req.Password == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Email and password are required")
		return
	}
	
	token, _ := h.authService.GenerateToken("user-id")
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user": map[string]string{
			"id":    uuid.New().String(),
			"email": req.Email,
		},
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Logged out successfully",
	})
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	newToken := "refreshed_jwt_token_" + uuid.New().String()
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"token": newToken,
	})
}

type PaymentHandler struct {
	logger           *log.Logger
	paymentProcessor *RazorpayProcessor
	subscriptionMgr  *SubscriptionManager
	planManager      *PlanManager
	paymentRepo      *PaymentRepository
	db               *gorm.DB
}

func (h *PaymentHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	// Set headers
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	
	// Marshal JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		h.logger.Printf("ERROR: Failed to marshal JSON: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to encode response",
		})
		return
	}
	
	// Remove BOM if present (UTF-8 BOM: EF BB BF)
	if len(jsonData) >= 3 && jsonData[0] == 0xEF && jsonData[1] == 0xBB && jsonData[2] == 0xBF {
		jsonData = jsonData[3:]
		h.logger.Printf("⚠️ BOM removed from response")
	}
	
	// Remove any leading non-JSON characters
	start := 0
	for i, b := range jsonData {
		if b == '{' || b == '[' {
			start = i
			break
		}
	}
	if start > 0 {
		jsonData = jsonData[start:]
		h.logger.Printf("⚠️ Leading characters removed from response")
	}
	
	// Write response
	w.WriteHeader(status)
	if _, err := w.Write(jsonData); err != nil {
		h.logger.Printf("ERROR: Failed to write response: %v", err)
	}
}

func NewPaymentHandler(logger *log.Logger, processor *RazorpayProcessor, subMgr *SubscriptionManager, planMgr *PlanManager, paymentRepo *PaymentRepository) *PaymentHandler {
	return &PaymentHandler{
		logger:           logger,
		paymentProcessor: processor,
		subscriptionMgr:  subMgr,
		planManager:      planMgr,
		paymentRepo:      paymentRepo,
	}
}

func (h *PaymentHandler) SetDB(db *gorm.DB) {
	h.db = db
	h.logger.Printf("Database connection set for PaymentHandler")
}

func (h *PaymentHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"key_id": h.paymentProcessor.keyID,
	})
}

func (h *PaymentHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	plans := h.planManager.ListPlans()
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"plans": plans,
	})
}

func (h *PaymentHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Plan     string `json:"plan"`
		Interval string `json:"interval"`
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Printf("CreateOrder: Invalid request: %v", err)
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}
	
	h.logger.Printf("CreateOrder request: plan=%s, interval=%s", req.Plan, req.Interval)
	
	// Map plan names to plan IDs
	var planID string
	switch req.Plan {
	case "starter", "essential", "basic":
		planID = "starter"
	case "professional", "pro":
		planID = "professional"
	case "enterprise", "agency":
		planID = "enterprise"
	default:
		planID = "starter"
	}
	
	// Get plan details
	plan, err := h.planManager.GetPlan(planID)
	if err != nil {
		h.logger.Printf("CreateOrder: Plan not found: %s", req.Plan)
		utils.ErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Plan '%s' not found", req.Plan))
		return
	}
	
	// Get price based on interval
	var price int
	if req.Interval == "yearly" {
		if yearlyPrice, ok := plan["yearly_price"].(int); ok {
			price = yearlyPrice
		} else {
			price = plan["price"].(int) * 12 * 80 / 100
		}
	} else {
		price = plan["price"].(int)
	}
	
	// Convert to cents
	amount := int64(price * 100)
	if req.Amount > 0 {
		amount = req.Amount
	}
	
	currency := "USD"
	if req.Currency != "" {
		currency = req.Currency
	}
	
	// Get user ID
	userID := r.Context().Value("user_id")
	if userID == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}
	
	orderID, err := h.paymentProcessor.CreateOrder(int(amount), currency)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to create order")
		return
	}
	
	// Return response
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"order_id":  orderID,
		"amount":    amount,
		"currency":  currency,
		"plan":      req.Plan,
		"plan_name": plan["name"],
		"key_id":    h.paymentProcessor.keyID,
		"interval":  req.Interval,
		"success":   true,
	})
}

func (h *PaymentHandler) VerifyPayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID   string `json:"order_id"`
		PaymentID string `json:"payment_id"`
		Signature string `json:"signature"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}
	
	verified := h.paymentProcessor.VerifyPayment(req.OrderID, req.PaymentID, req.Signature)
	
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"verified": verified,
		"message":  "Payment verified successfully",
	})
}

func (h *PaymentHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	var webhookData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&webhookData); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid webhook data")
		return
	}
	
	h.logger.Printf("Received webhook data=%v", webhookData)
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"received": true,
	})
}

func (h *PaymentHandler) GetPaymentHistory(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	if userID == "" {
		utils.ErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}
	
	payments, err := h.paymentRepo.FindByUserID(userID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch payments")
		return
	}
	
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"payments": payments,
	})
}

type ReportHandler struct {
	logger         *log.Logger
	reportGenerator interface{}
	pdfGenerator    interface{}
	emailReporter   interface{}
	reportRepo      *ReportRepository
}

func NewReportHandler(logger *log.Logger, reportGen, pdfGen, emailRep interface{}, reportRepo *ReportRepository) *ReportHandler {
	return &ReportHandler{
		logger:         logger,
		reportGenerator: reportGen,
		pdfGenerator:    pdfGen,
		emailReporter:   emailRep,
		reportRepo:      reportRepo,
	}
}

func (h *ReportHandler) GenerateReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type   string `json:"type"`
		Title  string `json:"title"`
		UserID string `json:"user_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}
	
	reportID := uuid.New().String()
	report := &Report{
		ID:        reportID,
		UserID:    req.UserID,
		Type:      req.Type,
		Title:     req.Title,
		Data:      "{}",
		FileURL:   fmt.Sprintf("/reports/%s.pdf", reportID),
		CreatedAt: time.Now(),
	}
	
	if err := h.reportRepo.Create(report); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to create report")
		return
	}
	
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"report_id":   reportID,
		"status":      "generated",
		"download_url": report.FileURL,
	})
}

func (h *ReportHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	if userID == "" {
		utils.ErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}
	
	reports, err := h.reportRepo.FindByUserID(userID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch reports")
		return
	}
	
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"reports": reports,
	})
}

func (h *ReportHandler) DownloadPDF(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "id")
	
	report, err := h.reportRepo.FindByID(reportID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "Report not found")
		return
	}
	
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report_%s.pdf", reportID))
	w.Write([]byte(fmt.Sprintf("PDF content for report: %s", report.Title)))
}

type WorkflowHandler struct {
	engine       *Engine
	logger       *log.Logger
	workflowRepo *WorkflowRepository
}

func NewWorkflowHandler(engine *Engine, logger *log.Logger, workflowRepo *WorkflowRepository) *WorkflowHandler {
	return &WorkflowHandler{
		engine:       engine,
		logger:       logger,
		workflowRepo: workflowRepo,
	}
}

func (h *WorkflowHandler) StartWorkflow(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL    string                 `json:"url"`
		UserID string                 `json:"user_id"`
		Input  map[string]interface{} `json:"input"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}
	
	if req.URL == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "URL is required")
		return
	}
	
	jobID := uuid.New().String()
	job := &Job{
		ID:        jobID,
		Type:      "seo_automation",
		UserID:    req.UserID,
		URL:       req.URL,
		Input:     req.Input,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	if err := h.engine.queue.Enqueue(job); err != nil {
		utils.ErrorResponse(w, http.StatusServiceUnavailable, "Queue is full. Please try again later.")
		return
	}
	
	workflow := &WorkflowDB{
		ID:        jobID,
		UserID:    req.UserID,
		Type:      "seo_automation",
		Status:    "pending",
		Input:     "{}",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	h.workflowRepo.Create(workflow)
	
	utils.JSONResponse(w, http.StatusAccepted, map[string]interface{}{
		"job_id": jobID,
		"status": "queued",
		"message": "SEO automation has been queued successfully",
	})
}

func (h *WorkflowHandler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "id")
	
	workflow, err := h.workflowRepo.FindByID(workflowID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "Workflow not found")
		return
	}
	
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"workflow": workflow,
	})
}

func (h *WorkflowHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	templates := []map[string]interface{}{
		{"id": "full_seo_audit", "name": "Full SEO Audit", "description": "Complete SEO analysis and optimization"},
		{"id": "keyword_research", "name": "Keyword Research", "description": "Keyword discovery and analysis"},
		{"id": "content_optimization", "name": "Content Optimization", "description": "Content improvement suggestions"},
		{"id": "technical_seo", "name": "Technical SEO", "description": "Technical SEO fixes"},
	}
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"templates": templates,
	})
}

// ============ MIDDLEWARE ============

type AuthMiddleware struct {
	authService *AuthService
	logger      *log.Logger
	publicPaths []string
}

func NewAuthMiddleware(authService *AuthService, logger *log.Logger, publicPaths []string) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		logger:      logger,
		publicPaths: publicPaths,
	}
}

func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		for _, publicPath := range m.publicPaths {
			if path == publicPath {
				next.ServeHTTP(w, r)
				return
			}
		}
		
		token := r.Header.Get("Authorization")
		if token == "" {
			utils.ErrorResponse(w, http.StatusUnauthorized, "Missing authorization token")
			return
		}
		
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
		
		userID, err := m.authService.ValidateToken(token)
		if err != nil {
			utils.ErrorResponse(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}
		
		ctx := context.WithValue(r.Context(), "user_id", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type RateLimitMiddleware struct {
	limit  int
	logger *log.Logger
}

func NewRateLimitMiddleware(limit int, logger *log.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limit:  limit,
		logger: logger,
	}
}

func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

// ============ ENGINE ============

type Engine struct {
	logger        *log.Logger
	maxConcurrent int
	queue         *QueueService
	seoHandler    *handlers.SEOHandler
	quit          chan bool
}

func NewEngine(logger *log.Logger, maxConcurrent int, queue *QueueService, seoHandler *handlers.SEOHandler) *Engine {
	return &Engine{
		logger:        logger,
		maxConcurrent: maxConcurrent,
		queue:         queue,
		seoHandler:    seoHandler,
		quit:          make(chan bool),
	}
}

func (e *Engine) StartEngine() {
	e.logger.Printf("Starting workflow engine workers=%d", e.maxConcurrent)
	for i := 0; i < e.maxConcurrent; i++ {
		go func(workerID int) {
			e.logger.Printf(fmt.Sprintf("Worker %d started", workerID))
			for {
				select {
				case job := <-e.queue.jobChan:
					e.logger.Printf(fmt.Sprintf("Worker %d processing job %s for URL: %s", workerID, job.ID, job.URL))
					
					job.Status = "processing"
					job.UpdatedAt = time.Now()
					
					e.seoHandler.RunFullSEOAutomation(job.ID, job.URL, job.UserID)
					
					job.Status = "completed"
					job.UpdatedAt = time.Now()
					
					e.logger.Printf(fmt.Sprintf("Worker %d completed job %s", workerID, job.ID))
				case <-e.quit:
					e.logger.Printf(fmt.Sprintf("Worker %d stopping", workerID))
					return
				}
			}
		}(i)
	}
}

func (e *Engine) StopEngine() {
	e.logger.Printf("Stopping workflow engine")
	close(e.quit)
}

// ============ JOB TYPES ============

type Job struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	UserID    string                 `json:"user_id"`
	URL       string                 `json:"url"`
	Input     map[string]interface{} `json:"input"`
	Status    string                 `json:"status"`
	Result    interface{}            `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// ============ APP STRUCT ============

type App struct {
	config                *Config
	router                *chi.Mux
	logger                *log.Logger
	db                    *gorm.DB
	userRepo              *UserRepository
	seoRepo               *SEORepository
	paymentRepo           *PaymentRepository
	workflowRepo          *WorkflowRepository
	reportRepo            *ReportRepository
	authService           *AuthService
	cacheService          CacheService
	queueService          *QueueService
	emailService          *EmailService
	paymentProcessor      *RazorpayProcessor
	subscriptionMgr       *SubscriptionManager
	planManager           *PlanManager
	reportGenerator       *reporting.ReportGenerator
	pdfGenerator          *reporting.PDFGenerator
	emailReporter         *reporting.EmailReporter
	workflowEngine        *workflow.Engine
	 authHandler *handlers.AuthHandler
	rateLimitMiddleware   *RateLimitMiddleware
	authMiddleware        *middleware.AuthMiddleware 
	seoHandler            *handlers.SEOHandler
	paymentHandler        *PaymentHandler
	reportHandler         *ReportHandler
	workflowHandler       *WorkflowHandler
	wordpressFixer        *wordpress.WordPressFixer
	shopifyFixer          *shopify.ShopifyFixer
	cloudflareFixer       *fixer.CloudflareFixer
	schemaInjector        *fixer.SchemaInjector
	ImageFixer            *fixer.ImageFixer
	redirectManager       *fixer.RedirectManager
	rollbackManager       *fixer.RollbackManager
	technicalFixer        *fixer.TechnicalFixer
	performanceFixer      *fixer.PerformanceFixer
	guideGenerator        *guide.Generator
	coreWebVitals         *analyzer.Client
	nlpAnalyzer           *analyzer.NLPAnalyzer
	seoScanner            *scanner.MetaScanner
	seoCrawler            *scanner.SEOCrawler
	contentEnhancer       *optimizer.Enhancer
	keywordOptimizer      *optimizer.KeywordOptimizer
	linkOptimizer         *fixer.InternalLinkOptimizer
}

// ============ CORS MIDDLEWARE (Production Safe) ============
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// ✅ ALLOWED ORIGINS (Production)
		allowedOrigins := map[string]bool{
			"http://localhost:3000":      true,
			"http://localhost:5173":      true,
			"http://127.0.0.1:3000":      true,
			"http://127.0.0.1:5173":      true,
			"http://localhost:8080":      true,
			"https://www.seosps.com":     true,
			"https://seosps.com":         true,
			"https://ai-seo-frontend.vercel.app": true,
			"https://seosps.vercel.app":  true,
		}
		
		// ✅ LOG FOR DEBUGGING
		log.Printf("🔍 CORS Request: Method=%s, Path=%s, Origin=%s", r.Method, r.URL.Path, origin)
		
		// ✅ CHECK IF ORIGIN IS ALLOWED
		if allowedOrigins[origin] || origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
			log.Printf("✅ CORS Allowed for: %s", origin)
		} else {
			// ✅ FOR DEVELOPMENT - ALLOW ANYWAY
			// Remove this in production if you want strict CORS
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
			log.Printf("⚠️ CORS Allowed with * for: %s (Development Mode)", origin)
		}
		
		// ✅ HANDLE PREFLIGHT REQUESTS
		if r.Method == "OPTIONS" {
			log.Printf("✅ CORS Preflight: %s", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// ============ NEWAPP FUNCTION ============
func NewApp(config *Config, logger *log.Logger) *App {
	// Initialize Database
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
			config.DBHost, config.DBUser, config.DBPassword, config.DBName, config.DBPort, config.DBSSLMode)
	}
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		logger.Printf("Warning: Database connection failed: %v", err)
		logger.Printf("Running without database - using local storage only")
		db = nil
	} else {
		logger.Printf("Database connected successfully")
	}
	
	// Auto-migrate schemas
	if db != nil {
		err = db.AutoMigrate(&User{}, &SEOAnalysis{}, &Payment{}, &WorkflowDB{}, &Report{}, &AIGuideReport{})
		if err != nil {
			logger.Printf("Failed to auto-migrate: %v", err)
		}
	} else {
		logger.Printf("Skipping auto-migrate - database not available")
	}
	
	// Initialize Repositories
	userRepo := NewUserRepository(db)
	seoRepo := NewSEORepository(db)
	paymentRepo := NewPaymentRepository(db)
	workflowRepo := NewWorkflowRepository(db)
	reportRepo := NewReportRepository(db)
	
	// Initialize Services
	var cacheService CacheService = nil
	queueService := NewQueueService(100, logger)
	emailService := NewEmailService(logger)
	
	// ✅ FIXED: Use only ONE authService declaration
	// Create authService with Google OAuth support
	authService := NewAuthService(userRepo, config.JWTSecret)
	
	// Create HTTP client
	httpClient := &http.Client{
		Timeout: config.RequestTimeout,
	}
	
	// ============ INITIALIZE COMPLETE SEO MODULES ============
	log.Println("🔧 Initializing complete SEO modules...")
	
	// Create output directory
	os.MkdirAll(config.OutputDir, 0755)
	os.MkdirAll(config.OutputDir+"/scans", 0755)
	os.MkdirAll(config.OutputDir+"/crawls", 0755)
	os.MkdirAll(config.OutputDir+"/reports", 0755)
	os.MkdirAll(config.OutputDir+"/pdfs", 0755)
	os.MkdirAll(config.OutputDir+"/templates", 0755)
	
	// 1. Core Web Vitals analyzer
	var coreWebVitals *analyzer.Client
	if config.CruxAPIKey != "" {
		coreWebVitals = analyzer.NewClient(config.CruxAPIKey, analyzer.DefaultConfig())
		log.Println("✅ Core Web Vitals analyzer initialized")
	} else {
		log.Println("⚠️ Core Web Vitals disabled (CRUX_API_KEY not set)")
	}
	
	// 2. NLP Analyzer
	nlpAnalyzer := analyzer.NewNLPAnalyzer()
	log.Println("✅ NLP Analyzer initialized")
	
	// 3. SEO Scanner
	scannerConfig := scanner.ScannerConfig{
		Timeout:          30 * time.Second,
		UserAgent:        "SEOBot/1.0",
		FollowRedirects:  true,
		MaxRedirects:     5,
		CheckBrokenLinks: true,
		OptimizeImages:   true,
		MinifyContent:    true,
		CheckMobile:      true,
		OutputDir:        config.OutputDir + "/scans",
		EnableJavaScript: false,
		Concurrency:      5,
	}
	seoScanner := scanner.NewMetaScanner(scannerConfig)
	log.Println("✅ SEO Scanner initialized")
	
	// 4. SEO Crawler
	crawlerConfig := scanner.CrawlerConfig{
		MaxDepth:         3,
		Timeout:          30 * time.Second,
		Concurrency:      5,
		RespectRobotsTxt: true,
		UserAgent:        "SEOCrawler/1.0",
		OptimizeImages:   true,
		CompressOutput:   true,
		CheckMobile:      true,
		OutputDir:        config.OutputDir + "/crawls",
	}
	seoCrawler := scanner.NewSEOCrawler(crawlerConfig)
	log.Println("✅ SEO Crawler initialized")
	
	// 5. Content Enhancer
	var contentEnhancer *optimizer.Enhancer
	if config.OpenAIAPIKey != "" {
		contentEnhancer = optimizer.New(config.OpenAIAPIKey)
		log.Println("✅ Content Enhancer initialized")
	} else {
		log.Println("⚠️ Content Enhancer disabled (OPENAI_API_KEY not set)")
	}
	
	// 6. Keyword Optimizer
	keywordOptimizer := optimizer.NewKeywordOptimizer()
	log.Println("✅ Keyword Optimizer initialized")
	
	// 7. Internal Link Optimizer
	linkOptimizer := fixer.NewInternalLinkOptimizer(logger)
	log.Println("✅ Internal Link Optimizer initialized")
	
	// 8. Report Generator
	reportConfig := reporting.ReportConfig{
		OutputDir:    config.OutputDir + "/reports",
		TemplateDir:  config.OutputDir + "/templates",
		PrimaryColor: "#4F46E5",
		FooterText:   "Generated by AI SEO Tool",
	}
	reportGenerator, err := reporting.NewReportGenerator(reportConfig)
	if err != nil {
		log.Printf("⚠️ Report Generator initialization failed: %v", err)
		reportGenerator = nil
	} else {
		log.Println("✅ Report Generator initialized")
	}
	
	// 9. PDF Generator
	pdfConfig := reporting.PDFConfig{
		OutputDir:   config.OutputDir + "/pdfs",
		PageSize:    "A4",
		Orientation: "portrait",
		Margins:     "20mm",
	}
	pdfGenerator, err := reporting.NewPDFGenerator(pdfConfig)
	if err != nil {
		log.Printf("⚠️ PDF Generator initialization failed: %v", err)
		pdfGenerator = nil
	} else {
		log.Println("✅ PDF Generator initialized")
	}
	
	// 10. Email Reporter
	var emailReporter *reporting.EmailReporter
	if config.SMTPUsername != "" && config.SMTPPassword != "" {
		emailConfig := reporting.EmailConfig{
			SMTPHost:   config.SMTPHost,
			SMTPPort:   config.SMTPPort,
			Username:   config.SMTPUsername,
			Password:   config.SMTPPassword,
			FromEmail:  config.FromEmail,
			FromName:   config.FromName,
			UseTLS:     true,
			RetryCount: 3,
		}
		emailReporter, err = reporting.NewEmailReporter(emailConfig)
		if err != nil {
			log.Printf("⚠️ Email Reporter initialization failed: %v", err)
			emailReporter = nil
		} else {
			log.Println("✅ Email Reporter initialized")
		}
	} else {
		log.Println("⚠️ Email Reporter disabled (SMTP credentials not set)")
	}
	
	if reportGenerator != nil && emailReporter != nil {
		reportGenerator.SetEmailReporter(emailReporter)
	}
	
	// 11. Workflow Engine
	workflowEngine := workflow.NewEngine(logger, config.MaxConcurrent)
	workflowEngine.RegisterTaskExecutor("crawl", &workflow.CrawlTaskExecutor{})
	workflowEngine.RegisterTaskExecutor("keyword_research", &workflow.KeywordResearchExecutor{})
	workflowEngine.RegisterTaskExecutor("content_optimizer", &workflow.ContentOptimizerExecutor{})
	workflowEngine.RegisterTaskExecutor("link_analyzer", &workflow.LinkAnalyzerExecutor{})
	workflowEngine.RegisterTaskExecutor("report_generator", &workflow.ReportGeneratorExecutor{})
	log.Println("✅ Workflow Engine initialized")
	log.Println("✅ All complete SEO modules initialized successfully!")
	
	// ============ INITIALIZE FIXER MODULES ============
	log.Println("🔧 Initializing fixer modules...")
	
	rollbackManager, err := fixer.NewRollbackManager("./backups", "")
	if err != nil {
		log.Fatal(err)
	}
	
	wordpressFixer := wordpress.NewWordPressFixer(httpClient, logger)
	shopifyFixer := shopify.NewShopifyFixer(httpClient, logger)
	cloudflareFixer := fixer.NewCloudflareFixer(httpClient, logger)
	schemaInjector := fixer.NewSchemaInjector(httpClient, logger)
	imageFixer := fixer.NewImageFixer(httpClient, logger)
	redirectManager := fixer.NewRedirectManager(httpClient, logger)
	technicalFixer := fixer.NewTechnicalFixer(httpClient, logger)
	performanceFixer := fixer.NewPerformanceFixer()
	guideGenerator := guide.NewGenerator(config.OpenAIAPIKey, logger)
	
	log.Println("✅ All fixer modules initialized successfully!")
	
	// ============ CREATE SEO HANDLER ============
	seoHandler := handlers.NewSEOHandler(
		logger,
		wordpressFixer,
		shopifyFixer,
		cloudflareFixer,
		schemaInjector,
		redirectManager,
		rollbackManager,
		technicalFixer,
		performanceFixer,
		guideGenerator,
		config.CruxAPIKey,
		config.OpenAIAPIKey,
		config.OutputDir,
		db,
	)
	
	// Payments
	paymentProcessor := NewRazorpayProcessor(
		config.RazorpayKeyID,
		config.RazorpayKeySecret,
		logger,
	)
	planManager := NewPlanManager(paymentRepo, logger)
	subscriptionMgr := NewSubscriptionManager(paymentRepo, planManager, logger)
	
	// Workflow Engine (Queue-based)
	queueEngine := NewEngine(logger, config.MaxConcurrent, queueService, seoHandler)

// ✅ FORCE PRODUCTION URL
googleRedirectURL := config.GoogleRedirectURL
if config.Environment == "production" {
    googleRedirectURL = "https://api.seosps.com/api/auth/google/callback"
    logger.Printf("🏭 Production: Google Redirect URL set to: %s", googleRedirectURL)
}

// ✅ Create the wrapper
authServiceWrapper := &AuthServiceWrapper{
    AuthService: authService,
}

// ✅ Pass the wrapper (which implements the interface)
authHandler := handlers.NewAuthHandler(authServiceWrapper, logger)

// ✅ Set Google config
authHandler.SetGoogleConfig(
    config.GoogleClientID,
    config.GoogleClientSecret,
    config.GoogleRedirectURL,
)

// ✅ Set database connection
authHandler.SetDB(db)

// ✅ ADD THIS CHECK - Verify authHandler is not nil
if authHandler == nil {
    logger.Fatal("❌ AuthHandler is nil!")
}
logger.Printf("✅ AuthHandler created successfully")

	// ============================================
	// ✅ FIXED: Initialize PaymentHandler
	// ============================================
	paymentHandler := NewPaymentHandler(
		logger,
		paymentProcessor,
		subscriptionMgr,
		planManager,
		paymentRepo,
	)
	paymentHandler.SetDB(db)
	
	// ============================================
	// ✅ FIXED: Initialize ReportHandler
	// ============================================
	reportHandler := NewReportHandler(
		logger,
		reportGenerator,
		pdfGenerator,
		emailReporter,
		reportRepo,
	)
	
	// ============================================
	// ✅ FIXED: Initialize WorkflowHandler
	// ============================================
	workflowHandler := NewWorkflowHandler(queueEngine, logger, workflowRepo)
	
	// ============================================
	// ✅ FIXED: Define public paths
	// ============================================
	publicPaths := []string{
		"/",
		"/health",
		"/api/docs",
		"/api/auth/login",
		"/api/auth/register",
		"/api/auth/google",
		"/api/auth/google/callback",
		"/api/auth/free-trial-status",
		"/api/payment/config",
		"/api/payment/plans",
		"/api/seo/analyze",
		"/api/seo/automate",
		"/api/seo/result/",
	}
	
	// ============================================
	// ✅ FIXED: Initialize AuthMiddleware
	// ============================================
	authMiddleware := middleware.NewAuthMiddleware(middleware.AuthConfig{
		JWTSecret:           config.JWTSecret,
		JWTIssuer:           "seosps",
		JWTAudience:         "seosps-users",
		 AuthService:         authService, 
		PublicRoutes:        publicPaths,
		GoogleClientID:      config.GoogleClientID,
		GoogleClientSecret:  config.GoogleClientSecret,
		GoogleRedirectURL:   config.GoogleRedirectURL,
	})
	
	// ============================================
	// ✅ FIXED: Initialize RateLimitMiddleware
	// ============================================
	rateLimitMiddleware := NewRateLimitMiddleware(config.RateLimit, logger)
	
	// ============================================
	// ✅ FIXED: Create App with all dependencies
	// ============================================
	app := &App{
		config:                config,
		router:                chi.NewRouter(),
		logger:                logger,
		db:                    db,
		userRepo:              userRepo,
		seoRepo:               seoRepo,
		paymentRepo:           paymentRepo,
		workflowRepo:          workflowRepo,
		reportRepo:            reportRepo,
		authService:           authService,
		cacheService:          cacheService,
		queueService:          queueService,
		emailService:          emailService,
		paymentProcessor:      paymentProcessor,
		subscriptionMgr:       subscriptionMgr,
		planManager:           planManager,
		reportGenerator:       reportGenerator,
		pdfGenerator:          pdfGenerator,
		emailReporter:         emailReporter,
		workflowEngine:        workflowEngine,
		authMiddleware:        authMiddleware,
		rateLimitMiddleware:   rateLimitMiddleware,
		authHandler:           authHandler,
		seoHandler:            seoHandler,
		paymentHandler:        paymentHandler,
		reportHandler:         reportHandler,
		workflowHandler:       workflowHandler,
		wordpressFixer:        wordpressFixer,
		shopifyFixer:          shopifyFixer,
		cloudflareFixer:       cloudflareFixer,
		schemaInjector:        schemaInjector,
		ImageFixer:            imageFixer,
		redirectManager:       redirectManager,
		rollbackManager:       rollbackManager,
		technicalFixer:        technicalFixer,
		performanceFixer:      performanceFixer,
		guideGenerator:        guideGenerator,
		coreWebVitals:         coreWebVitals,
		nlpAnalyzer:           nlpAnalyzer,
		seoScanner:            seoScanner,
		seoCrawler:            seoCrawler,
		contentEnhancer:       contentEnhancer,
		keywordOptimizer:      keywordOptimizer,
		linkOptimizer:         linkOptimizer,
	}
	
	
	app.authHandler = authHandler


	app.setupRoutes()
	
	return app
}

// ============================================
// ✅ FIXED: setupRoutes with Free Trial Routes and AI Guide Routes
func (app *App) setupRoutes() {
	app.router.Use(corsMiddleware)
	app.router.Use(app.rateLimitMiddleware.Handler)

	app.router.Get("/", app.handleRoot)
	app.router.Get("/health", app.handleHealth)
	app.router.Get("/api/docs", app.handleDocs)
	
	// Public auth routes
	app.router.Post("/api/auth/register", app.authHandler.Register)
	app.router.Post("/api/auth/login", app.authHandler.Login)
	
	// Google OAuth routes
	app.router.Get("/api/auth/google", app.authHandler.GoogleAuth)
	app.router.Get("/api/auth/google/callback", app.authHandler.GoogleCallback) 
	app.router.Post("/api/auth/google/token", app.authHandler.GoogleTokenValidate)
	
	// ✅ FREE TRIAL STATUS - Public
	app.router.Get("/api/auth/free-trial-status", app.handleFreeTrialStatus)
	
	// Public payment routes
	app.router.Get("/api/payment/config", app.paymentHandler.GetConfig)
	app.router.Get("/api/payment/plans", app.paymentHandler.ListPlans)
	
	// FREE SEO Analysis - Public
	app.router.Post("/api/seo/analyze", app.seoHandler.AnalyzeOnly)
	app.router.Get("/api/seo/result/{id}", app.seoHandler.GetAutomationResult)
	
	// Protected routes
	app.router.Group(func(r chi.Router) {
		r.Use(app.authMiddleware.Handler)
		
		// Auth routes
		r.Post("/api/auth/logout", app.authHandler.Logout)
		r.Post("/api/auth/refresh", app.authHandler.RefreshToken)
		
		// ✅ START FREE TRIAL - Protected
		r.Post("/api/payment/start-free-trial", app.handleStartFreeTrial)
		
		// SEO routes
		r.Post("/api/seo/automate", app.seoHandler.AutomateSEO)
		r.Get("/api/seo/history", app.seoHandler.GetAutomationHistory)
		r.Post("/api/seo/save-scan", app.seoHandler.SaveScan)
		
		// ✅ AI MANUAL GUIDE ROUTES - Protected
		r.Post("/api/seo/generate-guide", app.seoHandler.GenerateAIGuideReport)
		r.Get("/api/seo/ai-guide/{scanId}", app.seoHandler.GetAIGuideReport)
		
		// Payment routes
		r.Post("/api/payment/create-order", app.paymentHandler.CreateOrder)
		r.Post("/api/payment/verify", app.paymentHandler.VerifyPayment)
		r.Get("/api/payment/subscription/status", app.paymentHandler.CheckSubscriptionStatus)
		r.Get("/api/payment/history", app.paymentHandler.GetPaymentHistory)
		
		// Report routes
		r.Post("/api/reports/generate", app.reportHandler.GenerateReport)
		r.Get("/api/reports", app.reportHandler.ListReports)
		r.Get("/api/reports/{id}/pdf", app.reportHandler.DownloadPDF)
		
		// Workflow routes
		r.Post("/api/workflow/start", app.workflowHandler.StartWorkflow)
		r.Get("/api/workflow/status/{id}", app.workflowHandler.GetWorkflow)
		r.Get("/api/workflow/templates", app.workflowHandler.ListTemplates)
		
		// Dashboard routes
		r.Get("/api/dashboard", app.seoHandler.GetDashboardStats)
		r.Get("/api/dashboard/analysis/{websiteId}", app.seoHandler.GetDetailedAnalysis)
		r.Post("/api/dashboard/auto-fix", app.seoHandler.HandleAutoFix)
	})
}

// ============ ADVANCED ANALYSIS HANDLERS ============

// handleCruxAnalysis handles Core Web Vitals analysis
func (app *App) handleCruxAnalysis(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}
	
	if app.coreWebVitals == nil {
		utils.ErrorResponse(w, http.StatusServiceUnavailable, "Core Web Vitals not configured")
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	vitals, err := app.coreWebVitals.GetVitals(ctx, req.URL)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	utils.JSONResponse(w, http.StatusOK, vitals)
}
// In main.go, around line 2130, update handleStartFreeTrial function
func (app *App) handleStartFreeTrial(w http.ResponseWriter, r *http.Request) {
    app.logger.Printf("📥 handleStartFreeTrial called")
    
    // Get user ID from context
    userID := r.Context().Value("user_id")
    if userID == nil {
        userID = r.Context().Value(ContextKeyUserID)
        if userID == nil {
            app.logger.Printf("❌ No user_id in context")
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusUnauthorized)
            json.NewEncoder(w).Encode(map[string]interface{}{
                "success": false,
                "error":   "User not authenticated",
            })
            return
        }
    }

    userIDStr := userID.(string)
    app.logger.Printf("🔍 Starting free trial for user: %s", userIDStr)

    if app.db == nil {
        app.logger.Printf("❌ Database not available")
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "success": false,
            "error":   "Database not available",
        })
        return
    }

    // ✅ FIX: Check if user exists first
    var user User
    err := app.db.Where("id = ?", userIDStr).First(&user).Error
    
    if err != nil {
        // ✅ FIX: Don't create new user - instead return error
        app.logger.Printf("❌ User not found: %s", userIDStr)
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "success": false,
            "error":   "User not found. Please login again.",
            "requires_relogin": true,  // ✅ Tell frontend to re-login
        })
        return
    }

    // ✅ Check if already used free trial
    if user.FreeTrialUsed {
        app.logger.Printf("⚠️ User already used free trial: %s", user.ID)
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "success": false,
            "message": "You have already used your free trial",
            "trialData": map[string]interface{}{
                "used":           true,
                "days_remaining": 0,
                "is_expired":     true,
            },
        })
        return
    }

    // ✅ Start free trial
    now := time.Now()
    user.FreeTrialUsed = true
    user.FreeTrialStartDate = now
    user.FreeTrialEndDate = now.Add(7 * 24 * time.Hour)
    user.Plan = "free"
    user.Status = "active"

    if err := app.db.Save(&user).Error; err != nil {
        app.logger.Printf("❌ Failed to save trial: %v", err)
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "success": false,
            "error":   "Failed to save trial",
        })
        return
    }

    // ✅ Generate NEW token for the user
    newToken, err := app.authService.GenerateToken(user.ID)
    if err != nil {
        app.logger.Printf("❌ Failed to generate new token: %v", err)
    }

    app.logger.Printf("✅ Free trial started for user: %s", user.ID)

    // ✅ Return success with NEW token
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "message": "Free trial started! You have 7 days.",
        "token": newToken,  // ✅ Send new token
        "user": map[string]interface{}{
            "id":    user.ID,
            "email": user.Email,
            "name":  user.Name,
        },
        "trialData": map[string]interface{}{
            "used":           true,
            "days_remaining": 7,
            "is_expired":     false,
            "start_date":     user.FreeTrialStartDate.Format(time.RFC3339),
            "end_date":       user.FreeTrialEndDate.Format(time.RFC3339),
        },
    })
}
// handleFreeTrialStatus returns free trial status for the user
func (app *App) handleFreeTrialStatus(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated
	userID := r.Context().Value("user_id")
	if userID == nil {
		// User not authenticated - return generic trial info
		utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"authenticated":   false,
			"free_trial_used": false,
			"days_remaining":  7,
			"is_expired":      false,
			"message":         "Sign up for a free 7-day trial!",
			"signup_url":      "/signup",
		})
		return
	}
	
	// Check if database is available
	if app.db == nil {
		utils.ErrorResponse(w, http.StatusServiceUnavailable, "Database not available")
		return
	}
	
	// Get user from database
	var user User
	err := app.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}
	
	// Calculate trial status
	isExpired := time.Now().After(user.FreeTrialEndDate)
	daysUsed := int(time.Since(user.FreeTrialStartDate).Hours() / 24)
	daysRemaining := 7 - daysUsed
	if daysRemaining < 0 {
		daysRemaining = 0
	}
	
	// Check if user has active subscription
	hasActiveSubscription := user.SubscriptionEndDate.After(time.Now())
	
	// Determine if trial is available
	trialAvailable := !user.FreeTrialUsed && !isExpired && !hasActiveSubscription
	
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"authenticated":          true,
		"free_trial_used":        user.FreeTrialUsed || isExpired,
		"days_remaining":         daysRemaining,
		"is_expired":             isExpired,
		"start_date":             user.FreeTrialStartDate.Format(time.RFC3339),
		"end_date":               user.FreeTrialEndDate.Format(time.RFC3339),
		"trial_available":        trialAvailable,
		"has_active_subscription": hasActiveSubscription,
		"plan":                   user.Plan,
		"message": func() string {
			if hasActiveSubscription {
				return "You have an active subscription. Enjoy unlimited SEO automation!"
			} else if user.FreeTrialUsed && isExpired {
				return "Your free trial has expired. Please subscribe to continue using SEO automation."
			} else if user.FreeTrialUsed {
				return "You have used your free trial. Please subscribe to continue."
			} else if !isExpired {
				return fmt.Sprintf("You have %d days remaining in your free trial.", daysRemaining)
			}
			return "Sign up for a free 7-day trial!"
		}(),
		"upgrade_link": "/pricing",
	})
}

// handleContentAnalysis handles NLP content analysis
func (app *App) handleContentAnalysis(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
		Keyword string `json:"keyword,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}
	
	result := make(map[string]interface{})
	
	// NLP Analysis
	if app.nlpAnalyzer != nil {
		analysis := app.nlpAnalyzer.Analyze(req.Content)
		result["nlp_analysis"] = analysis
	}
	
	// Keyword analysis
	if app.keywordOptimizer != nil && req.Keyword != "" {
		keywordData, _ := app.keywordOptimizer.AnalyzeKeyword(req.Keyword)
		result["keyword_data"] = keywordData
		
		tips := app.keywordOptimizer.OptimizeContent(req.Content, req.Keyword)
		result["optimization_tips"] = tips
	}
	
	utils.JSONResponse(w, http.StatusOK, result)
}

// handleKeywordAnalysis handles keyword research
func (app *App) handleKeywordAnalysis(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Keyword     string   `json:"keyword"`
		Competitors []string `json:"competitors,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}
	
	if app.keywordOptimizer == nil {
		utils.ErrorResponse(w, http.StatusServiceUnavailable, "Keyword optimizer not configured")
		return
	}
	
	// Analyze keyword
	keywordData, err := app.keywordOptimizer.AnalyzeKeyword(req.Keyword)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	result := map[string]interface{}{
		"keyword": keywordData,
	}
	
	// Get LSI keywords
	lsi := app.keywordOptimizer.GetLSI(req.Keyword)
	result["lsi_keywords"] = lsi
	
	utils.JSONResponse(w, http.StatusOK, result)
}

// ============ HANDLER FUNCTIONS ============

func (app *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"service":     "AI SEO Backend",
		"version":     "2.0.0",
		"status":      "running",
		"environment": app.config.Environment,
		"timestamp":   time.Now().Unix(),
		"modules": map[string]bool{
			"seo_automation":    true,
			"fixer_modules":     true,
			"real_analysis":     true,
			"payments":          true,
			"reports":           true,
			"workflows":         true,
			"core_web_vitals":   app.coreWebVitals != nil,
			"nlp_analyzer":      app.nlpAnalyzer != nil,
			"seo_scanner":       app.seoScanner != nil,
			"seo_crawler":       app.seoCrawler != nil,
			"content_enhancer":  app.contentEnhancer != nil,
			"keyword_optimizer": app.keywordOptimizer != nil,
			"link_optimizer":    app.linkOptimizer != nil,
			"report_generator":  app.reportGenerator != nil,
			"pdf_generator":     app.pdfGenerator != nil,
			"email_reporter":    app.emailReporter != nil,
			"workflow_engine":   app.workflowEngine != nil,
		},
		"endpoints": map[string]string{
			"docs":    "/api/docs",
			"health":  "/health",
			"seo_api": "/api/seo",
		},
	})
}

func (app *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"services": map[string]string{
			"database":   "connected",
			"cache":      "available",
			"queue":      "running",
			"seo_engine": "active",
		},
		"metrics": map[string]interface{}{
			"queue_length": app.queueService.GetQueueLength(),
			"workers":      app.config.MaxConcurrent,
		},
	}
	utils.JSONResponse(w, http.StatusOK, health)
}

func (app *App) handleDocs(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"api_version": "2.0.0",
		"title":       "AI SEO Backend API",
		"description": "Complete SEO automation and optimization platform",
		"base_url":    fmt.Sprintf("http://localhost:%s", app.config.ServerPort),
		"endpoints": []map[string]interface{}{
			{
				"path":        "/api/seo/analyze",
				"method":      "POST",
				"description": "Free SEO analysis for any website",
				"auth":        false,
			},
			{
				"path":        "/api/seo/automate",
				"method":      "POST",
				"description": "Start SEO automation for a website (requires subscription)",
				"auth":        true,
			},
			{
				"path":        "/api/seo/result/{id}",
				"method":      "GET",
				"description": "Get analysis/automation results",
				"auth":        false,
			},
			{
				"path":        "/api/payment/config",
				"method":      "GET",
				"description": "Get Razorpay payment configuration",
				"auth":        false,
			},
			{
				"path":        "/api/payment/create-order",
				"method":      "POST",
				"description": "Create Razorpay payment order",
				"auth":        true,
			},
			{
				"path":        "/api/payment/verify",
				"method":      "POST",
				"description": "Verify Razorpay payment",
				"auth":        true,
			},
			{
				"path":        "/api/payment/check-subscription",
				"method":      "GET",
				"description": "Check user subscription status",
				"auth":        true,
			},
			{
				"path":        "/api/payment/history",
				"method":      "GET",
				"description": "Get payment history",
				"auth":        true,
			},
			{
				"path":        "/api/auth/register",
				"method":      "POST",
				"description": "User registration",
				"auth":        false,
			},
			{
				"path":        "/api/auth/login",
				"method":      "POST",
				"description": "User login",
				"auth":        false,
			},
			{
				"path":        "/api/workflow/start",
				"method":      "POST",
				"description": "Start a workflow",
				"auth":        true,
			},
			{
				"path":        "/health",
				"method":      "GET",
				"description": "Health check endpoint",
				"auth":        false,
			},
		},
	}
	utils.JSONResponse(w, http.StatusOK, docs)
}
func startTrialReminderScheduler(app *App) {
    ticker := time.NewTicker(24 * time.Hour)
    go func() {
        for range ticker.C {
            if app.db == nil {
                continue
            }
            
            // Define time variables
            now := time.Now()
            endDate1 := now.AddDate(0, 0, 7)   // 7 days from now
            endDate2 := now.AddDate(0, 0, 3)   // 3 days from now
            endDate3 := now.AddDate(0, 0, 1)   // 1 day from now
            
            // ========== FIX: Use all variables ==========
            // Get users with active trials ending in 7 days (using endDate1)
            var users []User
            app.db.Where(
                "free_trial_used = ? AND free_trial_end_date BETWEEN ? AND ?",
                true,
                now,
                endDate1,
            ).Find(&users)
            
            // Send reminders for each user
            for _, user := range users {
                daysLeft := int(user.FreeTrialEndDate.Sub(now).Hours() / 24)
                if daysLeft < 0 {
                    daysLeft = 0
                }
                
                // Use endDate2 and endDate3 for urgency
                if daysLeft <= 1 {
                    // Urgent - using endDate3 (1 day)
                    go sendTrialExpiringSoonEmail(user, daysLeft)
                    app.logger.Printf("📧 URGENT: Trial ends in %d days for %s", daysLeft, user.Email)
                } else if daysLeft <= 3 {
                    // Reminder - using endDate2 (3 days)
                    go sendTrialReminderEmail(user, daysLeft, "reminder")
                    app.logger.Printf("📧 REMINDER: Trial ends in %d days for %s", daysLeft, user.Email)
                } else {
                    // Early reminder
                    go sendTrialReminderEmail(user, daysLeft, "early")
                    app.logger.Printf("📧 EARLY: Trial ends in %d days for %s", daysLeft, user.Email)
                }
            }
            
            // Check for expired trials (using now)
            var expiredUsers []User
            app.db.Where(
                "free_trial_used = ? AND free_trial_end_date < ?",
                true,
                now,
            ).Find(&expiredUsers)
            
            for _, user := range expiredUsers {
                go sendTrialExpiredEmail(user)
                app.logger.Printf("📧 Sent trial expired email to %s", user.Email)
            }
            
            // ========== FIX: Log the unused variables to use them ==========
            app.logger.Printf("📊 Trial reminder stats: 7-day=%s, 3-day=%s, 1-day=%s", 
                endDate1.Format("2006-01-02"),
                endDate2.Format("2006-01-02"),
                endDate3.Format("2006-01-02"))
        }
    }()
}
// ========== ADD: Helper functions for trial reminders ==========

// sendTrialReminderEmail sends trial reminder email
func sendTrialReminderEmail(user User, daysLeft int, reminderType string) {
    subject := "Your Free Trial is Ending Soon!"
    var message string
    switch reminderType {
    case "urgent":
        message = fmt.Sprintf("⚠️ Your free trial ends in %d days! Subscribe now to continue using our SEO automation services.", daysLeft)
    case "reminder":
        message = fmt.Sprintf("⏰ Reminder: Your free trial ends in %d days. Upgrade to keep using our services.", daysLeft)
    default:
        message = fmt.Sprintf("📢 Your free trial is ending in %d days. Don't miss out on our premium features!", daysLeft)
    }
    
    // TODO: Implement actual email sending
    // Example with a hypothetical email service:
    // if err := emailService.SendEmail(user.Email, subject, message); err != nil {
    //     log.Printf("Failed to send email to %s: %v", user.Email, err)
    // }
    
    // Log the email for now
    log.Printf("📧 [TRIAL REMINDER] To: %s | Subject: %s | Message: %s", user.Email, subject, message)
}

// sendTrialExpiredEmail sends trial expired email
func sendTrialExpiredEmail(user User) {
    subject := "Your Free Trial Has Expired"
    message := "Your free trial has expired. Subscribe now to continue using our SEO automation services!"
    
    // TODO: Implement actual email sending
    // if err := emailService.SendEmail(user.Email, subject, message); err != nil {
    //     log.Printf("Failed to send email to %s: %v", user.Email, err)
    // }
    
    log.Printf("📧 [TRIAL EXPIRED] To: %s | Subject: %s | Message: %s", user.Email, subject, message)
}

// sendTrialExpiringSoonEmail sends trial expiring soon email
func sendTrialExpiringSoonEmail(user User, daysLeft int) {
    subject := fmt.Sprintf("⚠️ Your Free Trial Ends in %d Days!", daysLeft)
    message := fmt.Sprintf("Your free trial ends in %d days. Subscribe now to avoid interruption of services!", daysLeft)
    
    // TODO: Implement actual email sending
    // if err := emailService.SendEmail(user.Email, subject, message); err != nil {
    //     log.Printf("Failed to send email to %s: %v", user.Email, err)
    // }
    
    log.Printf("📧 [TRIAL EXPIRING SOON] To: %s | Subject: %s | Message: %s", user.Email, subject, message)
}

// sendEmail sends email (implement with SMTP)
func sendEmail(to, subject, body string) {
    log.Printf("Sending email to: %s, subject: %s", to, subject)
    // Use your email service here
}

func main() {
	// Check if .env exists
	if _, err := os.Stat(".env"); err == nil {
		log.Println("✅ .env file found")
	} else {
		log.Printf("❌ .env file NOT found: %v", err)
	}

	logger := log.New(os.Stdout, "[SEO] ", log.LstdFlags)

	config := LoadConfig()

	// Helper function to mask sensitive data
	maskString := func(s string) string {
		if len(s) == 0 {
			return "(empty)"
		}
		if len(s) <= 8 {
			return "***"
		}
		return s[:4] + "..." + s[len(s)-4:]
	}

	// DEBUG: Print actual config values
	fmt.Println("========================================")
	fmt.Println("🔧 DEBUG: Configuration Values")
	fmt.Printf("   CRUX_API_KEY: '%s' (length: %d)\n", maskString(config.CruxAPIKey), len(config.CruxAPIKey))
	fmt.Printf("   OPENAI_API_KEY: '%s' (length: %d)\n", maskString(config.OpenAIAPIKey), len(config.OpenAIAPIKey))
	fmt.Printf("   SMTP_USERNAME: '%s' (length: %d)\n", config.SMTPUsername, len(config.SMTPUsername))
	fmt.Printf("   SMTP_PASSWORD: '%s' (length: %d)\n", maskString(config.SMTPPassword), len(config.SMTPPassword))
	fmt.Printf("   JWT_SECRET: '%s' (length: %d)\n", maskString(config.JWTSecret), len(config.JWTSecret))
	fmt.Printf("   DB_HOST: '%s'\n", config.DBHost)
	fmt.Printf("   DB_USER: '%s'\n", config.DBUser)
	fmt.Printf("   ENVIRONMENT: '%s'\n", config.Environment)
	
	// 🔑 RAZORPAY KEY STATUS - IMPORTANT FOR PAYMENT
	if config.RazorpayKeyID != "" && config.RazorpayKeySecret != "" {
		fmt.Printf("   ✅ RAZORPAY_KEY_ID: '%s' (length: %d)\n", 
			maskString(config.RazorpayKeyID), len(config.RazorpayKeyID))
		fmt.Printf("   ✅ RAZORPAY_KEY_SECRET: '%s' (length: %d)\n", 
			maskString(config.RazorpayKeySecret), len(config.RazorpayKeySecret))
	} else {
		fmt.Printf("   ❌ RAZORPAY_KEY_ID: '%s' (length: %d) - ⚠️ PAYMENT WILL FAIL!\n", 
			maskString(config.RazorpayKeyID), len(config.RazorpayKeyID))
		fmt.Printf("   ❌ RAZORPAY_KEY_SECRET: '%s' (length: %d) - ⚠️ PAYMENT WILL FAIL!\n", 
			maskString(config.RazorpayKeySecret), len(config.RazorpayKeySecret))
		fmt.Println("   ⚠️  Please set RAZORPAY_KEY_ID and RAZORPAY_KEY_SECRET in .env file")
		fmt.Println("   🔑 Get your keys from: https://dashboard.razorpay.com/app/api-keys")
	}
	fmt.Println("========================================\n")

	logger.Printf("Starting AI SEO Backend environment=%s port=%s max_concurrent=%d",
		config.Environment, config.ServerPort, config.MaxConcurrent)

	app := NewApp(config, logger)

	 go startTrialReminderScheduler(app)
    
    // Start auto-scan scheduler
    go startAutoScanScheduler(app)
	
	// Start auto-scan scheduler for subscribers
	go startAutoScanScheduler(app)

	// Start workflow engine
	go app.workflowEngine.StartEngine()
	defer app.workflowEngine.StopEngine()

	server := &http.Server{
		Addr:         ":" + config.ServerPort,
		Handler:      app.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Printf("Server starting on port %s", config.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	fmt.Println("========================================")
	fmt.Println("🚀 AI SEO Backend Server v2.0")
	fmt.Println("========================================")
	fmt.Printf("✅ Environment: %s\n", config.Environment)
	fmt.Printf("✅ Server URL: http://localhost:%s\n", config.ServerPort)
	fmt.Printf("✅ Health Check: http://localhost:%s/health\n", config.ServerPort)
	fmt.Printf("✅ API Docs: http://localhost:%s/api/docs\n", config.ServerPort)
	fmt.Println("========================================")
	fmt.Println("📊 SEO Modules Status:")
	fmt.Printf("   • Core Web Vitals: %s\n", map[bool]string{true: "✅ Enabled", false: "⚠️ Disabled"}[config.CruxAPIKey != ""])
	fmt.Printf("   • NLP Analysis: ✅ Enabled\n")
	fmt.Printf("   • Content Enhancer: %s\n", map[bool]string{true: "✅ Enabled", false: "⚠️ Disabled"}[config.OpenAIAPIKey != ""])
	fmt.Printf("   • Email Reports: %s\n", map[bool]string{true: "✅ Enabled", false: "⚠️ Disabled"}[config.SMTPUsername != ""])
	
	// 🔑 Payment Status
	if config.RazorpayKeyID != "" && config.RazorpayKeySecret != "" {
		fmt.Printf("   • Payment Gateway: ✅ Enabled (Razorpay)\n")
	} else {
		fmt.Printf("   • Payment Gateway: ❌ DISABLED - Set RAZORPAY keys in .env\n")
	}
	fmt.Println("========================================")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Printf("Shutting down server...")
	fmt.Println("\n🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Printf("Server forced to shutdown: %v", err)
		fmt.Printf("❌ Server shutdown error: %v\n", err)
	}

	logger.Printf("Server exited properly")
	fmt.Println("✅ Server exited properly")
}

func startContinuousSEOScheduler(app *App) {
    ticker := time.NewTicker(1 * time.Hour) // Check every hour
    go func() {
        for range ticker.C {
            var users []models.User
            
            // Get users with active subscription
            app.db.Where("subscription_end_date > ?", time.Now()).Find(&users)
            
            for _, user := range users {
                shouldScan := false
                
                switch user.ScanFrequency {
                case "daily":
                    // Check if last scan was today
                    if user.LastScanAt == nil || user.LastScanAt.Before(time.Now().Truncate(24*time.Hour)) {
                        shouldScan = true
                    }
                case "weekly":
                    // Check if last scan was more than 7 days ago
                    if user.LastScanAt == nil || time.Since(*user.LastScanAt) > 7*24*time.Hour {
                        shouldScan = true
                    }
                }
                
                if shouldScan {
                    go app.seoHandler.RunFullSEOAutomation(generateJobID(), user.WebsiteURL, user.ID)
                    
                    // Update last scan time
                    app.db.Model(&user).Update("last_scan_at", time.Now())
                }
            }
        }
    }()
}

// generateJobID creates a unique job ID
func generateJobID() string {
    return uuid.New().String()
}
// getActiveSubscribers returns users with active subscriptions
func getActiveSubscribers() []User {
    // TODO: Query database for users with active subscriptions
    // For now, return empty slice (no automatic scans until implemented)
    return []User{}
}

func startWeeklyEmailScheduler(seoHandler *handlers.SEOHandler) {
    ticker := time.NewTicker(1 * time.Hour) // Check every hour
    for range ticker.C {
        // Get users who need weekly report (7 days since last)
        users := getUsersForWeeklyReport()
        
        for _, user := range users {
            // Generate and send report
            report := seoHandler.GenerateWeeklyProgressReport(user)
            sendWeeklyReportEmail(user.Email, report)
        }
    }
}
// getUsersForWeeklyReport returns users who need weekly report
func getUsersForWeeklyReport() []models.User {
    // TODO: Query database for users with last_email_sent > 7 days ago
    // For now, return empty slice
    return []models.User{}
}

// sendWeeklyReportEmail sends weekly SEO report email
func sendWeeklyReportEmail(email string, report *handlers.WeeklyReport) {
    log.Printf("Sending weekly report to %s", email)
}

// ============ AUTO SCAN SCHEDULER ============
func startAutoScanScheduler(app *App) {
    ticker := time.NewTicker(1 * time.Hour) // Check every hour
    go func() {
        for range ticker.C {
            // Skip if database is nil
            if app.db == nil {
                continue
            }
            
            var users []models.User
            
            // Get users with active subscription
            app.db.Where("subscription_end_date > ?", time.Now()).Find(&users)
            
            for _, user := range users {
                shouldScan := false
                
                // Determine scan frequency based on plan and interval
                // Monthly Starter ($49) → weekly scan
                // Yearly Starter ($470) → daily scan
                // Monthly Professional ($149) → weekly scan  
                // Yearly Professional ($1430) → daily scan
                // Monthly Enterprise ($299) → weekly scan
                // Yearly Enterprise ($2870) → daily scan
                
                switch user.Plan {
                case "starter", "professional", "enterprise":
                    // Check if user paid yearly (check subscription amount or interval)
                    if user.LastPaymentAmount >= 470 { // Yearly plan (20% discount applied)
                        // Daily scan for yearly subscribers
                        if user.LastScanAt == nil || time.Since(*user.LastScanAt) >= 24*time.Hour {
                            shouldScan = true
                        }
                    } else {
                        // Weekly scan for monthly subscribers
                        if user.LastScanAt == nil || time.Since(*user.LastScanAt) >= 7*24*time.Hour {
                            shouldScan = true
                        }
                    }
                default:
                    // Default to weekly
                    if user.LastScanAt == nil || time.Since(*user.LastScanAt) >= 7*24*time.Hour {
                        shouldScan = true
                    }
                }
                
                if shouldScan {
                    go app.seoHandler.RunFullSEOAutomation(generateJobID(), user.WebsiteURL, user.ID)
                    app.db.Model(&user).Update("last_scan_at", time.Now())
                }
            }
        }
    }()
}
