package services

import (
    "context"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "strings"
    "time"
    "strconv"
    "io/ioutil"
    "net/http"

    "ai-seo-backend/internal/repository"
    "golang.org/x/crypto/bcrypt"
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
)

type AuthService struct {
    userRepo      *repository.UserRepository
    jwtSecret     []byte
    tokenExpiry   time.Duration
    googleOAuth   *oauth2.Config
}

type TokenDetails struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresAt    time.Time `json:"expires_at"`
    TokenType    string    `json:"token_type"`
}

type Claims struct {
    UserID    int       `json:"user_id"`
    Email     string    `json:"email"`
    Name      string    `json:"name"`
    Plan      string    `json:"plan"`
    Provider  string    `json:"provider"`
    ExpiresAt time.Time `json:"expires_at"`
}

// GoogleUserInfo represents the user info from Google
type GoogleUserInfo struct {
    ID            string `json:"id"`
    Email         string `json:"email"`
    Name          string `json:"name"`
    GivenName     string `json:"given_name"`
    FamilyName    string `json:"family_name"`
    Picture       string `json:"picture"`
    VerifiedEmail bool   `json:"verified_email"`
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, googleClientID, googleClientSecret, redirectURL string) *AuthService {
    return &AuthService{
        userRepo:    userRepo,
        jwtSecret:   []byte(jwtSecret),
        tokenExpiry: 24 * time.Hour,
        googleOAuth: &oauth2.Config{
            ClientID:     googleClientID,
            ClientSecret: googleClientSecret,
            RedirectURL:  redirectURL,
            Scopes: []string{
                "https://www.googleapis.com/auth/userinfo.email",
                "https://www.googleapis.com/auth/userinfo.profile",
            },
            Endpoint: google.Endpoint,
        },
    }
}

// =============================================
// EXISTING AUTH METHODS (Keep as they are)
// =============================================

// Register registers a new user
func (s *AuthService) Register(email, name, password string) (*repository.User, error) {
    // Validate input
    if email == "" || name == "" || password == "" {
        return nil, errors.New("email, name, and password are required")
    }

    if len(password) < 8 {
        return nil, errors.New("password must be at least 8 characters")
    }

    // Check if user exists
    existing, _ := s.userRepo.GetByEmail(email)
    if existing != nil {
        return nil, errors.New("user already exists")
    }

    // Hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return nil, fmt.Errorf("failed to hash password: %w", err)
    }

    // Generate API key
    apiKey, err := s.generateAPIKey()
    if err != nil {
        return nil, fmt.Errorf("failed to generate API key: %w", err)
    }

    // Create user
    user := &repository.User{
        Email:         email,
        Name:          name,
        PasswordHash:  string(hashedPassword),
        APIKey:        apiKey,
        Plan:          "free",
        Credits:       100,
        IsActive:      true,
        Provider:      "local",
        IsVerified:    false,
        LastLogin:     time.Now(),
        CreatedAt:     time.Now(),
        UpdatedAt:     time.Now(),
    }

    err = s.userRepo.Create(user)
    if err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }

    return user, nil
}

// Login authenticates a user
func (s *AuthService) Login(email, password string) (*TokenDetails, error) {
    // Get user
    user, err := s.userRepo.GetByEmail(email)
    if err != nil {
        return nil, errors.New("invalid credentials")
    }

    if !user.IsActive {
        return nil, errors.New("account is deactivated")
    }

    // Check if user is a Google user
    if user.Provider == "google" {
        return nil, errors.New("please use Google login for this account")
    }

    // Verify password
    if user.PasswordHash == "" {
        return nil, errors.New("invalid credentials")
    }

    err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
    if err != nil {
        return nil, errors.New("invalid credentials")
    }

    // Update last login
    s.userRepo.UpdateLastLogin(user.ID)

    // Generate tokens
    tokens, err := s.generateTokens(user)
    if err != nil {
        return nil, fmt.Errorf("failed to generate tokens: %w", err)
    }

    return tokens, nil
}

// =============================================
// NEW GOOGLE OAUTH METHODS
// =============================================

// GetGoogleAuthURL returns the Google OAuth URL
func (s *AuthService) GetGoogleAuthURL(state string) string {
    return s.googleOAuth.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// HandleGoogleCallback handles the Google OAuth callback
func (s *AuthService) HandleGoogleCallback(code string) (*TokenDetails, *repository.User, error) {
    // Exchange code for token
    ctx := context.Background()
    oauthToken, err := s.googleOAuth.Exchange(ctx, code)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to exchange token: %w", err)
    }

    // Get user info from Google
    userInfo, err := s.getGoogleUserInfo(oauthToken.AccessToken)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to get user info: %w", err)
    }

    // Find or create user
    user, err := s.findOrCreateGoogleUser(userInfo)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to find or create user: %w", err)
    }

    // Update last login
    s.userRepo.UpdateLastLogin(user.ID)

    // Generate tokens
    tokens, err := s.generateTokens(user)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
    }

    return tokens, user, nil
}

// findOrCreateGoogleUser finds existing user or creates new one
func (s *AuthService) findOrCreateGoogleUser(googleUser *GoogleUserInfo) (*repository.User, error) {
    // Check if user exists with Google ID
    user, err := s.userRepo.GetByGoogleID(googleUser.ID)
    if err == nil && user != nil {
        // User exists with Google ID
        return user, nil
    }

    // Check if user exists with same email
    user, err = s.userRepo.GetByEmail(googleUser.Email)
    if err == nil && user != nil {
        // User exists with email - link Google account
        user.GoogleID = googleUser.ID
        user.Provider = "google"
        user.IsVerified = googleUser.VerifiedEmail
        user.Avatar = googleUser.Picture
        user.UpdatedAt = time.Now()
        
        err = s.userRepo.Update(user)
        if err != nil {
            return nil, fmt.Errorf("failed to link Google account: %w", err)
        }
        return user, nil
    }

    // Create new user with Google
    apiKey, err := s.generateAPIKey()
    if err != nil {
        return nil, fmt.Errorf("failed to generate API key: %w", err)
    }

    newUser := &repository.User{
        GoogleID:      googleUser.ID,
        Email:         googleUser.Email,
        Name:          googleUser.Name,
        Avatar:        googleUser.Picture,
        APIKey:        apiKey,
        Plan:          "free",
        Credits:       100,
        IsActive:      true,
        Provider:      "google",
        IsVerified:    googleUser.VerifiedEmail,
        PasswordHash:  "", // No password for Google users
        LastLogin:     time.Now(),
        CreatedAt:     time.Now(),
        UpdatedAt:     time.Now(),
    }

    err = s.userRepo.Create(newUser)
    if err != nil {
        return nil, fmt.Errorf("failed to create Google user: %w", err)
    }

    return newUser, nil
}

// getGoogleUserInfo fetches user info from Google API
func (s *AuthService) getGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
    url := "https://www.googleapis.com/oauth2/v2/userinfo"
    
    req, err := http.NewRequest("GET", url, nil)
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
        return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
    }
    
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    
    var userInfo GoogleUserInfo
    err = json.Unmarshal(body, &userInfo)
    if err != nil {
        return nil, err
    }
    
    return &userInfo, nil
}

// ValidateGoogleToken validates Google ID token (for mobile apps)
func (s *AuthService) ValidateGoogleToken(idToken string) (*TokenDetails, *repository.User, error) {
    // Verify the token with Google
    url := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken)
    
    resp, err := http.Get(url)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to validate token: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, nil, errors.New("invalid Google token")
    }
    
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, nil, err
    }
    
    var tokenInfo map[string]interface{}
    err = json.Unmarshal(body, &tokenInfo)
    if err != nil {
        return nil, nil, err
    }
    
    // Extract user info from token
    googleUser := &GoogleUserInfo{
        ID:            tokenInfo["sub"].(string),
        Email:         tokenInfo["email"].(string),
        Name:          tokenInfo["name"].(string),
        Picture:       tokenInfo["picture"].(string),
        VerifiedEmail: tokenInfo["email_verified"].(bool),
    }
    
    // Find or create user
    user, err := s.findOrCreateGoogleUser(googleUser)
    if err != nil {
        return nil, nil, err
    }
    
    // Update last login
    s.userRepo.UpdateLastLogin(user.ID)
    
    // Generate tokens
    tokens, err := s.generateTokens(user)
    if err != nil {
        return nil, nil, err
    }
    
    return tokens, user, nil
}

// =============================================
// EXISTING METHODS (Keep as they are)
// =============================================

// ValidateToken validates an access token
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
    // Simple token validation (in production, use JWT library)
    parts := strings.Split(tokenString, ".")
    if len(parts) != 3 {
        return nil, errors.New("invalid token format")
    }

    // Decode payload (simplified - use proper JWT in production)
    payload, err := base64.RawURLEncoding.DecodeString(parts[1])
    if err != nil {
        return nil, errors.New("invalid token payload")
    }

    // Parse claims (simplified)
    var claims Claims
    fmt.Sscanf(string(payload), "%d|%s|%s|%s|%d", &claims.UserID, &claims.Email, &claims.Name, &claims.Plan, &claims.ExpiresAt)

    if time.Now().After(claims.ExpiresAt) {
        return nil, errors.New("token expired")
    }

    return &claims, nil
}

// RefreshToken generates new access token using refresh token
func (s *AuthService) RefreshToken(refreshToken string) (*TokenDetails, error) {
    // Validate refresh token (simplified)
    // In production, store refresh tokens in database
    
    // Extract user ID from refresh token
    data, err := base64.RawURLEncoding.DecodeString(refreshToken)
    if err != nil {
        return nil, errors.New("invalid refresh token")
    }

    var userID int
    fmt.Sscanf(string(data), "%d", &userID)

    // Get user
    userIDStr := strconv.Itoa(userID)
    user, err := s.userRepo.GetByID(userIDStr)
    if err != nil {
        return nil, errors.New("user not found")
    }

    // Generate new tokens
    return s.generateTokens(user)
}

// ChangePassword changes user password
func (s *AuthService) ChangePassword(userID int, oldPassword, newPassword string) error {
    if len(newPassword) < 8 {
        return errors.New("new password must be at least 8 characters")
    }

    // Get user
    userIDStr := strconv.Itoa(userID)
    user, err := s.userRepo.GetByID(userIDStr)
    if err != nil {
        return errors.New("user not found")
    }

    // Check if user is Google user
    if user.Provider == "google" {
        return errors.New("Google users can't change password. Please use Google to login")
    }

    // Verify old password
    if user.PasswordHash == "" {
        return errors.New("invalid current password")
    }

    err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword))
    if err != nil {
        return errors.New("invalid old password")
    }

    // Hash new password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
    if err != nil {
        return fmt.Errorf("failed to hash password: %w", err)
    }

    // Update password
    user.PasswordHash = string(hashedPassword)
    user.UpdatedAt = time.Now()
    
    err = s.userRepo.Update(user)
    if err != nil {
        return fmt.Errorf("failed to update password: %w", err)
    }

    return nil
}

// ResetPassword initiates password reset
func (s *AuthService) ResetPassword(email string) (string, error) {
    user, err := s.userRepo.GetByEmail(email)
    if err != nil {
        return "", errors.New("user not found")
    }

    // Check if user is Google user
    if user.Provider == "google" {
        return "", errors.New("Google users can't reset password. Please use Google to login")
    }

    // Generate reset token
    token, err := s.generateRandomString(32)
    if err != nil {
        return "", fmt.Errorf("failed to generate reset token: %w", err)
    }

    // Store reset token with expiry (implement this)
    // s.storeResetToken(user.ID, token, time.Now().Add(1*time.Hour))

    return token, nil
}

// ValidateAPIKey validates an API key
func (s *AuthService) ValidateAPIKey(apiKey string) (*repository.User, error) {
    return s.userRepo.GetByAPIKey(apiKey)
}

// GenerateAPIKey generates a new API key
func (s *AuthService) GenerateAPIKey(userID int) (string, error) {
    apiKey, err := s.generateAPIKey()
    if err != nil {
        return "", err
    }

    // Update user's API key (you'd need to add this method)
    // err = s.userRepo.UpdateAPIKey(userID, apiKey)
    
    return apiKey, nil
}

// =============================================
// HELPER FUNCTIONS
// =============================================

// generateTokens generates access and refresh tokens
func (s *AuthService) generateTokens(user *repository.User) (*TokenDetails, error) {
    // Generate access token (simplified - use proper JWT in production)
    expiresAt := time.Now().Add(s.tokenExpiry)
    
    // Create claims
    claims := fmt.Sprintf("%d|%s|%s|%s|%d", user.ID, user.Email, user.Name, user.Plan, expiresAt.Unix())
    
    // Simple token creation (in production, use proper JWT library)
    header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
    payload := base64.RawURLEncoding.EncodeToString([]byte(claims))
    
    // Create signature (simplified)
    signature := sha256.Sum256([]byte(header + "." + payload + string(s.jwtSecret)))
    sigBase64 := base64.RawURLEncoding.EncodeToString(signature[:])
    
    accessToken := header + "." + payload + "." + sigBase64

    // Generate refresh token
    refreshData := fmt.Sprintf("%d|%d", user.ID, time.Now().Add(7*24*time.Hour).Unix())
    refreshToken := base64.RawURLEncoding.EncodeToString([]byte(refreshData))

    return &TokenDetails{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresAt:    expiresAt,
        TokenType:    "Bearer",
    }, nil
}

func (s *AuthService) generateAPIKey() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return "key_" + hex.EncodeToString(bytes), nil
}

func (s *AuthService) generateRandomString(length int) (string, error) {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes)[:length], nil
}