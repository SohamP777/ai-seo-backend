// Package models contains data structures for user management
package models

import (
    "time"
)

// UserRole defines user permission levels
type UserRole string

const (
    RoleAdmin   UserRole = "admin"
    RoleUser    UserRole = "user"
    RoleViewer  UserRole = "viewer"
    RoleApiOnly UserRole = "api_only"
)

// UserStatus defines account status
type UserStatus string

const (
    StatusActive    UserStatus = "active"
    StatusInactive  UserStatus = "inactive"
    StatusSuspended UserStatus = "suspended"
    StatusPending   UserStatus = "pending_verification"
)

// SubscriptionTier defines plan levels
type SubscriptionTier string

const (
    TierFree       SubscriptionTier = "free"
    TierBasic      SubscriptionTier = "basic"
    TierPro        SubscriptionTier = "pro"
    TierEnterprise SubscriptionTier = "enterprise"
)

// AuthProvider defines authentication provider
type AuthProvider string

const (
    ProviderLocal  AuthProvider = "local"
    ProviderGoogle AuthProvider = "google"
)

type User struct {
    // Basic Info
    ID          string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    Email       string     `gorm:"uniqueIndex;not null" json:"email"`
    Password    string     `gorm:"not null" json:"-"`
    Name        string     `json:"name"`
    FirstName   string     `json:"first_name"`
    LastName    string     `json:"last_name"`
    Role        string     `gorm:"default:user" json:"role"`
    WebsiteURL  string     `json:"website_url"`
    
    // Google OAuth Fields
    GoogleID    string     `gorm:"uniqueIndex" json:"google_id,omitempty"`
    Avatar      string     `json:"avatar,omitempty"`
    Provider    string     `gorm:"default:local" json:"provider"` // local, google
    
    // Subscription
    Plan                string     `gorm:"default:starter" json:"plan"`
    SubscriptionEndDate time.Time  `gorm:"column:subscription_end_date" json:"subscription_end_date"`
    MaxWebsites         int        `gorm:"default:1" json:"max_websites"`
    ScanFrequency       string     `gorm:"default:weekly" json:"scan_frequency"`
    Status              string     `gorm:"default:active" json:"status"`
    LastPaymentAmount   float64    `gorm:"type:decimal(10,2);default:0" json:"last_payment_amount"`
    
    // ✅ NEW FIELDS - Used by auth_service.go and handlers
    Credits      int        `gorm:"default:100" json:"credits"`
    IsActive     bool       `gorm:"default:true" json:"is_active"`
    PasswordHash string     `gorm:"not null" json:"-"` // For hashed passwords (bcrypt)
    APIKey       string     `gorm:"uniqueIndex" json:"api_key,omitempty"`
    LastLogin    time.Time  `json:"last_login"`
    
    // Limits
    MaxScansPerDay     int `gorm:"default:10" json:"max_scans_per_day"`
    MaxPagesPerScan    int `gorm:"default:100" json:"max_pages_per_scan"`
    MaxDomains         int `gorm:"default:1" json:"max_domains"`
    ApiRequestsPerHour int `gorm:"default:100" json:"api_requests_per_hour"`
    
    // Usage Tracking
    ScansToday int        `gorm:"default:0" json:"scans_today"`
    LastScanAt *time.Time `json:"last_scan_at,omitempty"`
    
    // Security
    EmailVerified    bool   `gorm:"default:false" json:"email_verified"`
    TwoFactorEnabled bool   `gorm:"default:false" json:"two_factor_enabled"`
    TwoFactorSecret  string `json:"-"`
    
    // API Access
    ApiKeyExpiresAt *time.Time `json:"api_key_expires_at,omitempty"`
    
    // Timestamps
    CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
    LastLoginAt *time.Time `json:"last_login_at,omitempty"`
    DeletedAt   *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// UserSession represents a user login session
type UserSession struct {
    ID           string     `json:"id" db:"id"`
    UserID       string     `json:"user_id" db:"user_id"`
    Token        string     `json:"token" db:"token"`
    RefreshToken string     `json:"refresh_token" db:"refresh_token"`
    IPAddress    string     `json:"ip_address" db:"ip_address"`
    UserAgent    string     `json:"user_agent" db:"user_agent"`
    DeviceType   string     `json:"device_type" db:"device_type"`
    Location     string     `json:"location,omitempty" db:"location"`
    ExpiresAt    time.Time  `json:"expires_at" db:"expires_at"`
    CreatedAt    time.Time  `json:"created_at" db:"created_at"`
    LastActiveAt *time.Time `json:"last_active_at" db:"last_active_at"`
    IsActive     bool       `json:"is_active" db:"is_active"`
}

// LoginRequest represents a login attempt
type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Remember bool   `json:"remember"`
}

// LoginResponse represents successful login
type LoginResponse struct {
    User         *User  `json:"user"`
    Token        string `json:"token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int    `json:"expires_in"`
}

// RegisterRequest represents a new user registration
type RegisterRequest struct {
    Email           string `json:"email" validate:"required,email"`
    Password        string `json:"password" validate:"required,min=8"`
    ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
    FirstName       string `json:"first_name" validate:"required"`
    LastName        string `json:"last_name" validate:"required"`
    Company         string `json:"company"`
    Website         string `json:"website"`
    AcceptTerms     bool   `json:"accept_terms" validate:"required,eq=true"`
}

// GoogleAuthRequest represents Google OAuth login request
type GoogleAuthRequest struct {
    IDToken string `json:"id_token" validate:"required"`
}

// GoogleUserInfo represents user info from Google
type GoogleUserInfo struct {
    ID            string `json:"id"`
    Email         string `json:"email"`
    Name          string `json:"name"`
    GivenName     string `json:"given_name"`
    FamilyName    string `json:"family_name"`
    Picture       string `json:"picture"`
    VerifiedEmail bool   `json:"verified_email"`
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
    Email string `json:"email" validate:"required,email"`
}

// PasswordReset represents a password reset with token
type PasswordReset struct {
    Token     string     `json:"token" db:"token"`
    UserID    string     `json:"user_id" db:"user_id"`
    ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
    UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
    CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// UserDomain represents a domain owned by a user
type UserDomain struct {
    ID                string     `json:"id" db:"id"`
    UserID            string     `json:"user_id" db:"user_id"`
    Domain            string     `json:"domain" db:"domain"`
    Verified          bool       `json:"verified" db:"verified"`
    VerifiedAt        *time.Time `json:"verified_at,omitempty" db:"verified_at"`
    VerificationToken string     `json:"-" db:"verification_token"`
    LastScannedAt     *time.Time `json:"last_scanned_at,omitempty" db:"last_scanned_at"`
    ScanCount         int        `json:"scan_count" db:"scan_count"`
    CreatedAt         time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// UserAPIKey represents an API key for programmatic access
type UserAPIKey struct {
    ID          string     `json:"id" db:"id"`
    UserID      string     `json:"user_id" db:"user_id"`
    Name        string     `json:"name" db:"name"`
    Key         string     `json:"key" db:"key"`
    LastChars   string     `json:"last_chars" db:"last_chars"`
    Permissions []string   `json:"permissions" db:"permissions"`
    LastUsedAt  *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
    ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
    CreatedAt   time.Time  `json:"created_at" db:"created_at"`
    IsActive    bool       `json:"is_active" db:"is_active"`
}

// UserPreferences represents user settings
type UserPreferences struct {
    UserID             string          `json:"user_id" db:"user_id"`
    EmailNotifications bool            `json:"email_notifications" db:"email_notifications"`
    ReportFrequency    string          `json:"report_frequency" db:"report_frequency"`
    Theme              string          `json:"theme" db:"theme"`
    Timezone           string          `json:"timezone" db:"timezone"`
    DateFormat         string          `json:"date_format" db:"date_format"`
    DashboardWidgets   map[string]bool `json:"dashboard_widgets" db:"dashboard_widgets"`
    UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`
}

// ==================== HELPER METHODS ====================

// FullName returns user's full name
func (u *User) FullName() string {
    if u.FirstName != "" && u.LastName != "" {
        return u.FirstName + " " + u.LastName
    }
    return u.Name
}

// IsAdmin checks if user is admin
func (u *User) IsAdmin() bool {
    return u.Role == string(RoleAdmin)
}

// IsGoogleUser checks if user authenticated with Google
func (u *User) IsGoogleUser() bool {
    return u.Provider == string(ProviderGoogle) && u.GoogleID != ""
}

// IsLocalUser checks if user authenticated with email/password
func (u *User) IsLocalUser() bool {
    return u.Provider == string(ProviderLocal) || u.Provider == ""
}

// CanScan checks if user can perform another scan
func (u *User) CanScan() bool {
    return u.ScansToday < u.MaxScansPerDay
}

// HasReachedDomainLimit checks if user can add more domains
func (u *User) HasReachedDomainLimit(currentDomains int) bool {
    return currentDomains >= u.MaxDomains
}

// GetTierLimits returns limits for a subscription tier
func GetTierLimits(tier SubscriptionTier) (scansPerDay int, pagesPerScan int, maxDomains int, apiRate int) {
    switch tier {
    case TierFree:
        return 10, 100, 1, 60
    case TierBasic:
        return 50, 500, 5, 300
    case TierPro:
        return 200, 2000, 20, 1000
    case TierEnterprise:
        return 1000, 10000, 100, 5000
    default:
        return 10, 100, 1, 60
    }
}

// ValidatePassword checks password strength
func ValidatePassword(password string) []string {
    var errors []string
    
    if len(password) < 8 {
        errors = append(errors, "Password must be at least 8 characters")
    }
    
    hasUpper := false
    hasLower := false
    hasNumber := false
    hasSpecial := false
    
    for _, char := range password {
        switch {
        case 'A' <= char && char <= 'Z':
            hasUpper = true
        case 'a' <= char && char <= 'z':
            hasLower = true
        case '0' <= char && char <= '9':
            hasNumber = true
        default:
            hasSpecial = true
        }
    }
    
    if !hasUpper {
        errors = append(errors, "Password must contain at least one uppercase letter")
    }
    if !hasLower {
        errors = append(errors, "Password must contain at least one lowercase letter")
    }
    if !hasNumber {
        errors = append(errors, "Password must contain at least one number")
    }
    if !hasSpecial {
        errors = append(errors, "Password must contain at least one special character")
    }
    
    return errors
}