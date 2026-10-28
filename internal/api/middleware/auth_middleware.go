// File: internal/api/middleware/auth.go
package middleware

import (
    "context"
    "fmt"
    "net/http"
    "strings"
    "time"
    "encoding/json"
    "regexp"

    "github.com/golang-jwt/jwt/v5"
)

// =============================================
// CONSTANTS
// =============================================

const (
    // Context Keys
    ContextKeyUserID          = "user_id"
    ContextKeyUserEmail       = "user_email"
    ContextKeyUserName        = "user_name"
    ContextKeyUserRoles       = "user_roles"
    ContextKeyUserPermissions = "user_permissions"
    ContextKeyOrganizationID  = "organization_id"
    ContextKeyTokenID         = "token_id"
    ContextKeyJWTClaims       = "jwt_claims"
    ContextKeyUserProvider    = "user_provider"
    ContextKeyUserAvatar      = "user_avatar"
)

// =============================================
// TYPES
// =============================================

// AuthMiddleware provides JWT authentication
type AuthMiddleware struct {
    jwtSecret    string
    jwtIssuer    string
    jwtAudience  string
    authService  AuthService
    publicRoutes map[string]bool
}

// AuthService defines authentication service interface
type AuthService interface {
    ValidateToken(tokenString string) (*Claims, error)
    GetUserPermissions(userID string) ([]string, error)
    IsTokenBlacklisted(tokenID string) (bool, error)
    GetUserByGoogleID(googleID string) (interface{}, error)
    // ✅ FIXED: Match the actual AuthService implementation
    FindOrCreateGoogleUser(email, name, googleID string) (string, error)
}

// Claims represents JWT claims
type Claims struct {
    UserID         string   `json:"user_id"`
    Email          string   `json:"email"`
    Name           string   `json:"name"`
    Roles          []string `json:"roles"`
    Permissions    []string `json:"permissions"`
    OrganizationID *string  `json:"organization_id,omitempty"`
    TokenID        string   `json:"token_id"`
    Provider       string   `json:"provider,omitempty"`
    Avatar         string   `json:"avatar,omitempty"`
    jwt.RegisteredClaims
}

// GoogleUserInfo represents Google OAuth user info
type GoogleUserInfo struct {
    ID            string `json:"id"`
    Email         string `json:"email"`
    Name          string `json:"name"`
    GivenName     string `json:"given_name"`
    FamilyName    string `json:"family_name"`
    Picture       string `json:"picture"`
    VerifiedEmail bool   `json:"verified_email"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
    JWTSecret          string
    JWTIssuer          string
    JWTAudience        string
    AuthService        AuthService
    PublicRoutes       []string
    GoogleClientID     string
    GoogleClientSecret string
    GoogleRedirectURL  string
}

// =============================================
// CONSTRUCTOR
// =============================================

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(config AuthConfig) *AuthMiddleware {
    publicRoutes := make(map[string]bool)
    for _, route := range config.PublicRoutes {
        publicRoutes[route] = true
    }

    return &AuthMiddleware{
        jwtSecret:    config.JWTSecret,
        jwtIssuer:    config.JWTIssuer,
        jwtAudience:  config.JWTAudience,
        authService:  config.AuthService,
        publicRoutes: publicRoutes,
    }
}

// =============================================
// MAIN HANDLER
// =============================================

// Handler returns the authentication middleware handler
func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // ========== FIX: Add nil check for authService ==========
        if m.authService == nil {
            m.loggerError("❌ CRITICAL: AuthService is nil in AuthMiddleware!")
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(map[string]interface{}{
                "error":   "server_error",
                "message": "Authentication service not available",
            })
            return
        }

        // Skip authentication for public routes
        if m.isPublicRoute(r) {
            next.ServeHTTP(w, r)
            return
        }

        // Extract token from header
        tokenString, err := m.extractToken(r)
        if err != nil {
            m.unauthorized(w, "missing_authorization", "Authorization header is required")
            return
        }

        // Validate token
        claims, err := m.validateToken(tokenString)
        if err != nil {
            m.unauthorized(w, "invalid_token", "Invalid or expired token")
            return
        }

        // ========== FIX: Add nil check for claims ==========
        if claims == nil {
            m.unauthorized(w, "invalid_claims", "Invalid token claims")
            return
        }

        // Check if token is blacklisted
        blacklisted, err := m.authService.IsTokenBlacklisted(claims.TokenID)
        if err != nil {
            // Log the error but don't block the request if blacklist check fails
            m.loggerError(fmt.Sprintf("⚠️ Token blacklist check failed for token %s: %v", claims.TokenID, err))
        }
        if blacklisted {
            m.unauthorized(w, "token_revoked", "Token has been revoked")
            return
        }

        // Add claims to context
        ctx := r.Context()

        // ========== FIX: Add safe context value setting ==========
        if claims.UserID != "" {
            ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
        }
        if claims.Email != "" {
            ctx = context.WithValue(ctx, ContextKeyUserEmail, claims.Email)
        }
        if claims.Name != "" {
            ctx = context.WithValue(ctx, ContextKeyUserName, claims.Name)
        }
        if claims.Roles != nil {
            ctx = context.WithValue(ctx, ContextKeyUserRoles, claims.Roles)
        }
        if claims.Permissions != nil {
            ctx = context.WithValue(ctx, ContextKeyUserPermissions, claims.Permissions)
        }
        if claims.OrganizationID != nil {
            ctx = context.WithValue(ctx, ContextKeyOrganizationID, claims.OrganizationID)
        }
        if claims.TokenID != "" {
            ctx = context.WithValue(ctx, ContextKeyTokenID, claims.TokenID)
        }

        ctx = context.WithValue(ctx, ContextKeyJWTClaims, claims)
        ctx = context.WithValue(ctx, ContextKeyUserProvider, claims.Provider)
        ctx = context.WithValue(ctx, ContextKeyUserAvatar, claims.Avatar)

        // Update request with context
        r = r.WithContext(ctx)

        next.ServeHTTP(w, r)
    })
}

// =============================================
// TOKEN EXTRACTION & VALIDATION
// =============================================

// extractToken extracts token from Authorization header
func (m *AuthMiddleware) extractToken(r *http.Request) (string, error) {
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        return "", fmt.Errorf("authorization header is missing")
    }

    parts := strings.Split(authHeader, " ")
    if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
        return "", fmt.Errorf("authorization header format must be Bearer {token}")
    }

    return parts[1], nil
}

// validateToken validates JWT token
func (m *AuthMiddleware) validateToken(tokenString string) (*Claims, error) {
    claims := &Claims{}

    token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(m.jwtSecret), nil
    })

    if err != nil {
        return nil, fmt.Errorf("token parsing failed: %w", err)
    }

    if !token.Valid {
        return nil, fmt.Errorf("invalid token")
    }

    // Validate issuer
    if claims.Issuer != m.jwtIssuer {
        return nil, fmt.Errorf("invalid issuer")
    }

    // Validate audience
    audience, err := claims.GetAudience()
    if err != nil || len(audience) == 0 || !contains(audience, m.jwtAudience) {
        return nil, fmt.Errorf("invalid audience")
    }

    // Validate expiration
    if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
        return nil, fmt.Errorf("token has expired")
    }

    return claims, nil
}

// =============================================
// ROUTE CHECKING
// =============================================

// isPublicRoute checks if route is public
func (m *AuthMiddleware) isPublicRoute(r *http.Request) bool {
    path := r.URL.Path
    method := r.Method

    // Check exact path
    if m.publicRoutes[fmt.Sprintf("%s %s", method, path)] {
        return true
    }

    // Check wildcard paths
    for route := range m.publicRoutes {
        if strings.Contains(route, "*") {
            routePattern := strings.ReplaceAll(route, "*", ".*")
            routePattern = strings.ReplaceAll(routePattern, "/", "\\/")
            if match, _ := regexp.MatchString(routePattern, fmt.Sprintf("%s %s", method, path)); match {
                return true
            }
        }
    }

    return false
}

// =============================================
// RESPONSE HELPERS
// =============================================

// unauthorized sends unauthorized response
func (m *AuthMiddleware) unauthorized(w http.ResponseWriter, errorCode, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="%s"`, m.jwtIssuer))
    w.WriteHeader(http.StatusUnauthorized)

    json.NewEncoder(w).Encode(map[string]interface{}{
        "error":   errorCode,
        "message": message,
    })
}

// forbidden sends forbidden response
func (m *AuthMiddleware) forbidden(w http.ResponseWriter, errorCode, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusForbidden)

    json.NewEncoder(w).Encode(map[string]interface{}{
        "error":   errorCode,
        "message": message,
    })
}

// loggerError logs error messages
func (m *AuthMiddleware) loggerError(message string) {
    // Simple log output
    fmt.Println(message)
}

// =============================================
// PERMISSION & ROLE MIDDLEWARES
// =============================================

// RequirePermission middleware checks for specific permission
func (m *AuthMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID, ok := r.Context().Value(ContextKeyUserID).(string)
            if !ok || userID == "" {
                m.unauthorized(w, "missing_user", "User not authenticated")
                return
            }

            permissions, ok := r.Context().Value(ContextKeyUserPermissions).([]string)
            if !ok {
                var err error
                permissions, err = m.authService.GetUserPermissions(userID)
                if err != nil {
                    m.forbidden(w, "permission_check_failed", "Failed to check permissions")
                    return
                }
            }

            if !contains(permissions, permission) {
                m.forbidden(w, "insufficient_permissions",
                    fmt.Sprintf("Required permission: %s", permission))
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// RequireRole middleware checks for specific role
func (m *AuthMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            roles, ok := r.Context().Value(ContextKeyUserRoles).([]string)
            if !ok {
                m.forbidden(w, "missing_roles", "User roles not found")
                return
            }

            if !contains(roles, role) {
                m.forbidden(w, "insufficient_role",
                    fmt.Sprintf("Required role: %s", role))
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// RequireOrganization middleware checks organization access
func (m *AuthMiddleware) RequireOrganization() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            orgID := r.Context().Value(ContextKeyOrganizationID)
            if orgID == nil {
                m.forbidden(w, "missing_organization", "Organization access required")
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// =============================================
// GOOGLE OAUTH MIDDLEWARES
// =============================================

// RequireGoogleUser middleware ensures user is authenticated via Google
func (m *AuthMiddleware) RequireGoogleUser() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            provider := GetUserProvider(r.Context())
            if provider != "google" {
                m.forbidden(w, "google_user_required", "This endpoint requires Google authentication")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// RequireLocalUser middleware ensures user is authenticated via email/password
func (m *AuthMiddleware) RequireLocalUser() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            provider := GetUserProvider(r.Context())
            if provider != "local" && provider != "" {
                m.forbidden(w, "local_user_required", "This endpoint requires email/password authentication")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// HasVerifiedEmail middleware checks if email is verified
func (m *AuthMiddleware) HasVerifiedEmail() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // For Google users, email is always verified
            provider := GetUserProvider(r.Context())
            if provider == "google" {
                next.ServeHTTP(w, r)
                return
            }

            // For local users, check verification status from claims
            claims, ok := r.Context().Value(ContextKeyJWTClaims).(*Claims)
            if ok && claims != nil {
                next.ServeHTTP(w, r)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// =============================================
// CONTEXT HELPERS
// =============================================

// GetUserProvider returns the authentication provider from context
func GetUserProvider(ctx context.Context) string {
    if provider, ok := ctx.Value(ContextKeyUserProvider).(string); ok {
        return provider
    }
    return ""
}

// GetUserAvatar returns the user avatar from context
func GetUserAvatar(ctx context.Context) string {
    if avatar, ok := ctx.Value(ContextKeyUserAvatar).(string); ok {
        return avatar
    }
    return ""
}

// IsGoogleUser checks if the user is authenticated via Google
func IsGoogleUser(ctx context.Context) bool {
    return GetUserProvider(ctx) == "google"
}

// IsLocalUser checks if the user is authenticated via email/password
func IsLocalUser(ctx context.Context) bool {
    provider := GetUserProvider(ctx)
    return provider == "local" || provider == ""
}

// GetUserID returns the user ID from context
func GetUserID(ctx context.Context) string {
    if id, ok := ctx.Value(ContextKeyUserID).(string); ok {
        return id
    }
    return ""
}

// GetUserEmail returns the user email from context
func GetUserEmail(ctx context.Context) string {
    if email, ok := ctx.Value(ContextKeyUserEmail).(string); ok {
        return email
    }
    return ""
}

// GetUserName returns the user name from context
func GetUserName(ctx context.Context) string {
    if name, ok := ctx.Value(ContextKeyUserName).(string); ok {
        return name
    }
    return ""
}

// =============================================
// UTILITY FUNCTIONS
// =============================================

// contains checks if slice contains string
func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}