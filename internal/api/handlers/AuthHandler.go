package handlers

import (
	"net/http"
	"encoding/json"
	"time"
	"fmt" 
	"io"
	"log"
	"os"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"net/url"
	"sync"

	"github.com/google/uuid"  
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// =============================================
// INTERFACES
// =============================================

type AuthServiceInterface interface {
	ValidateToken(token string) (string, error)
	GenerateToken(userID string) (string, error)
	FindOrCreateGoogleUser(email, name, googleID string) (string, error)
}

// =============================================
// STRUCTS
// =============================================

type AuthHandler struct {
	authService        AuthServiceInterface
	logger             *log.Logger
	jwtSecret          []byte
	userStore          map[string]User
	googleClientID     string
	googleClientSecret string
	googleRedirectURL  string
	db                 *gorm.DB
	mu                 sync.RWMutex
	oauthStates        map[string]time.Time
}

type User struct {
	ID                   string    `json:"id"`
	Email                string    `json:"email"`
	Password             string    `json:"-"`
	Name                 string    `json:"name"`
	CreatedAt            time.Time `json:"created_at"`
	Status               string    `json:"status"`
	Plan                 string    `json:"plan"`
	SubscriptionEndDate  time.Time `json:"subscription_end_date"`
	MaxWebsites          int       `json:"max_websites"`
	GoogleID             string    `json:"google_id,omitempty"`
	Avatar               string    `json:"avatar,omitempty"`
	Provider             string    `json:"provider"`
	FreeTrialUsed        bool      `json:"free_trial_used"`
	FreeTrialStartDate   time.Time `json:"free_trial_start_date"`
	FreeTrialEndDate     time.Time `json:"free_trial_end_date"`
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type GoogleAuthRequest struct {
	IDToken string `json:"id_token"`
}

type AuthResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message,omitempty"`
	User    struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Avatar   string `json:"avatar,omitempty"`
		Provider string `json:"provider,omitempty"`
		Plan     string `json:"plan,omitempty"`
	} `json:"user,omitempty"`
}

// =============================================
// CONSTRUCTOR
// =============================================

func NewAuthHandler(authService AuthServiceInterface, logger *log.Logger) *AuthHandler {
	if logger == nil {
		logger = log.New(os.Stdout, "[SEO] ", log.LstdFlags)
	}
	
	return &AuthHandler{
		authService:        authService,
		logger:             logger,
		jwtSecret:          []byte("default-secret-change-me"),
		userStore:          make(map[string]User),
		googleClientID:     "",
		googleClientSecret: "",
		googleRedirectURL:  "",
		db:                 nil,
		oauthStates:        make(map[string]time.Time),
	}
}

// SetDB sets the database connection
func (h *AuthHandler) SetDB(db *gorm.DB) {
	if db == nil {
		h.logger.Printf("⚠️ Warning: Attempted to set nil database connection")
		return
	}
	h.db = db
	h.logger.Printf("✅ Database connection set for AuthHandler")
}

// SetGoogleConfig sets Google OAuth configuration
func (h *AuthHandler) SetGoogleConfig(clientID, clientSecret, redirectURL string) {
	h.googleClientID = clientID
	h.googleClientSecret = clientSecret
	h.googleRedirectURL = redirectURL
	h.logger.Printf("✅ Google OAuth configured with RedirectURL: %s", redirectURL)
}

// =============================================
// AUTH HANDLERS
// =============================================

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	
	var req RegisterRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Printf("❌ JSON decode error: %v", err)
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "Email, password, and name are required")
		return
	}

	if h.db != nil {
		var existingUser User
		err := h.db.Where("email = ?", req.Email).First(&existingUser).Error
		if err == nil {
			h.sendErrorResponse(w, http.StatusConflict, "User with this email already exists. Please login.")
			return
		}
	} else {
		for _, user := range h.userStore {
			if user.Email == req.Email {
				h.sendErrorResponse(w, http.StatusConflict, "User already exists")
				return
			}
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Printf("❌ Password hashing error: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to create account")
		return
	}

	userID := uuid.New().String()
	googleID := "local_" + uuid.New().String()

	user := User{
		ID:                 userID,
		Email:              req.Email,
		Password:           string(hashedPassword),
		Name:               req.Name,
		GoogleID:           googleID,
		Provider:           "local",
		Status:             "active",
		Plan:               "free",
		CreatedAt:          time.Now(),
		MaxWebsites:        1,
		FreeTrialUsed:      false,
		FreeTrialStartDate: time.Now(),
		FreeTrialEndDate:   time.Now().Add(7 * 24 * time.Hour),
	}

	if h.db != nil {
		err = h.db.Create(&user).Error
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				h.sendErrorResponse(w, http.StatusConflict, "User already exists. Please login.")
				return
			}
			h.logger.Printf("❌ Failed to create user: %v", err)
			h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to create user")
			return
		}
	} else {
		h.userStore[user.ID] = user
	}

	token, err := h.authService.GenerateToken(user.ID)
	if err != nil {
		h.logger.Printf("❌ Token generation error: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response := h.createAuthResponse(true, token, "Registration successful", &user)
	h.sendJSONResponse(w, http.StatusCreated, response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	var foundUser *User

	if h.db != nil {
		var user User
		err := h.db.Where("email = ?", req.Email).First(&user).Error
		if err == nil {
			foundUser = &user
		}
	} else {
		for _, user := range h.userStore {
			if user.Email == req.Email {
				foundUser = &user
				break
			}
		}
	}

	if foundUser == nil {
		h.sendErrorResponse(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if foundUser.Provider == "google" {
		h.sendErrorResponse(w, http.StatusBadRequest, "This account uses Google login. Please use 'Continue with Google'.")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(req.Password)); err != nil {
		h.sendErrorResponse(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token, err := h.authService.GenerateToken(foundUser.ID)
	if err != nil {
		h.logger.Printf("❌ Token generation error: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response := h.createAuthResponse(true, token, "Login successful", foundUser)
	h.sendJSONResponse(w, http.StatusOK, response)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.sendJSONResponse(w, http.StatusOK, AuthResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	token, err := h.authService.GenerateToken("user-id")
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	h.sendJSONResponse(w, http.StatusOK, AuthResponse{
		Success: true,
		Token:   token,
		Message: "Token refreshed successfully",
	})
}

// =============================================
// GOOGLE OAUTH HANDLERS - FIXED (NO COOKIES)
// =============================================

func (h *AuthHandler) GoogleAuth(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("🔑 GoogleAuth: Initiating Google OAuth flow")
	
	isProduction := os.Getenv("RENDER") != "" || os.Getenv("ENVIRONMENT") == "production"
	
	var frontendURL, backendURL string
	
	if isProduction {
		frontendURL = os.Getenv("FRONTEND_URL")
		if frontendURL == "" {
			frontendURL = "https://www.seosps.com"
		}
		backendURL = os.Getenv("BACKEND_URL")
		if backendURL == "" {
			backendURL = "https://api.seosps.com"
		}
		h.logger.Printf("🏭 Running in PRODUCTION mode")
	} else {
		frontendURL = os.Getenv("FRONTEND_URL")
		if frontendURL == "" {
			frontendURL = "http://localhost:3000"
		}
		backendURL = os.Getenv("BACKEND_URL")
		if backendURL == "" {
			backendURL = "http://localhost:8080"
		}
		h.logger.Printf("💻 Running in LOCAL mode")
	}

	// ✅ Remove trailing slash from frontendURL
	frontendURL = strings.TrimSuffix(frontendURL, "/")
	
	h.googleRedirectURL = backendURL + "/api/auth/google/callback"

	h.logger.Printf("🔑 GoogleAuth: RedirectURL: %s", h.googleRedirectURL)
	h.logger.Printf("🔑 GoogleAuth: FrontendURL: %s", frontendURL)

	if h.googleClientID == "" || h.googleClientSecret == "" {
		h.logger.Printf("❌ GoogleAuth: Google OAuth not configured")
		h.sendErrorResponse(w, http.StatusInternalServerError, "Google OAuth not configured")
		return
	}

	state, err := generateRandomString(32)
	if err != nil {
		h.logger.Printf("❌ GoogleAuth: Failed to generate state: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to generate state")
		return
	}

	// ✅ Store state in memory (NO COOKIES!)
	h.mu.Lock()
	h.oauthStates[state] = time.Now().Add(10 * time.Minute)
	h.mu.Unlock()
	
	h.logger.Printf("✅ State stored in memory: %s", state[:8]+"...")

	// ✅ NO COOKIES - REMOVED ALL COOKIE CODE

	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=email%%20profile&state=%s&access_type=offline&prompt=select_account",
		h.googleClientID,
		h.googleRedirectURL,
		state,
	)

	h.logger.Printf("✅ GoogleAuth: Redirecting to Google OAuth")
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("🚨 Google Callback executing")
	
	defer func() {
		if err := recover(); err != nil {
			h.logger.Printf("❌ PANIC in GoogleCallback: %v", err)
			http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		}
	}()

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "https://www.seosps.com"
	}
	// ✅ Remove trailing slash
	frontendURL = strings.TrimSuffix(frontendURL, "/")

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	
	h.logger.Printf("🔍 Code: %s", code[:20]+"...")
	h.logger.Printf("🔍 State from URL: %s", state[:8]+"...")
	
	// ✅ Validate state from memory (NOT from cookie)
	h.mu.RLock()
	expiry, exists := h.oauthStates[state]
	h.mu.RUnlock()
	
	if !exists {
		h.logger.Printf("❌ State not found in memory")
		errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("Invalid state parameter"))
		http.Redirect(w, r, errorURL, http.StatusFound)
		return
	}
	
	if time.Now().After(expiry) {
		h.logger.Printf("❌ State expired")
		errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("State expired"))
		http.Redirect(w, r, errorURL, http.StatusFound)
		return
	}
	
	// ✅ Clean up used state
	h.mu.Lock()
	delete(h.oauthStates, state)
	h.mu.Unlock()
	
	h.logger.Printf("✅ State validated successfully from memory")
	
	errorParam := r.URL.Query().Get("error")
	if errorParam != "" {
		h.logger.Printf("❌ Google returned error: %s", errorParam)
		errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("Google authentication failed: "+errorParam))
		http.Redirect(w, r, errorURL, http.StatusFound)
		return
	}
	
	if code == "" {
		h.logger.Printf("❌ No code found in URL")
		errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("No authorization code"))
		http.Redirect(w, r, errorURL, http.StatusFound)
		return
	}

	h.logger.Printf("✅ Code found, exchanging for token...")
	
	accessToken, err := h.exchangeCodeForToken(code)
	if err != nil {
		h.logger.Printf("❌ Token exchange error: %v", err)
		errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("Authentication failed"))
		http.Redirect(w, r, errorURL, http.StatusFound)
		return
	}

	userInfo, err := h.getGoogleUserInfo(accessToken)
	if err != nil {
		h.logger.Printf("❌ Get user info error: %v", err)
		errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("Failed to get user info"))
		http.Redirect(w, r, errorURL, http.StatusFound)
		return
	}

	user, err := h.findOrCreateGoogleUser(userInfo)
	if err != nil {
		h.logger.Printf("❌ Find/create user error: %v", err)
		errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("Failed to create user"))
		http.Redirect(w, r, errorURL, http.StatusFound)
		return
	}
	
	if user == nil {
		h.logger.Printf("❌ User is nil after findOrCreateGoogleUser")
		errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("User creation failed"))
		http.Redirect(w, r, errorURL, http.StatusFound)
		return
	}
	
	if h.authService == nil {
		h.logger.Printf("❌ AuthService is nil")
		errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("Authentication service unavailable"))
		http.Redirect(w, r, errorURL, http.StatusFound)
		return
	}

	token, err := h.authService.GenerateToken(user.ID)
	if err != nil {
		h.logger.Printf("❌ Token generation error: %v", err)
		errorURL := fmt.Sprintf("%s/signup?error=%s", frontendURL, url.QueryEscape("Failed to generate token"))
		http.Redirect(w, r, errorURL, http.StatusFound)
		return
	}

	// ✅ NO COOKIE CLEANUP NEEDED

	redirectURL := fmt.Sprintf("%s/signup?token=%s&success=true", frontendURL, token)
	h.logger.Printf("✅ Redirecting to: %s", redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *AuthHandler) GoogleTokenValidate(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("🔑 GoogleTokenValidate: Validating Google ID token")
	
	var req GoogleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.IDToken == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "ID token required")
		return
	}

	resp, err := http.Get(fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", req.IDToken))
	if err != nil {
		h.logger.Printf("❌ Token validation error: %v", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to validate token")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.sendErrorResponse(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	var tokenInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to parse token")
		return
	}

	userInfo := &GoogleUserInfo{
		ID:            getString(tokenInfo, "sub"),
		Email:         getString(tokenInfo, "email"),
		Name:          getString(tokenInfo, "name"),
		Picture:       getString(tokenInfo, "picture"),
		VerifiedEmail: getBool(tokenInfo, "email_verified"),
	}

	if userInfo.Email == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid token: email missing")
		return
	}

	user, err := h.findOrCreateGoogleUser(userInfo)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to find/create user")
		return
	}
	
	if user == nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, "User creation failed")
		return
	}

	token, err := h.authService.GenerateToken(user.ID)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response := h.createAuthResponse(true, token, "Google token validated", user)
	h.sendJSONResponse(w, http.StatusOK, response)
}

// =============================================
// HELPER FUNCTIONS
// =============================================

func (h *AuthHandler) exchangeCodeForToken(code string) (string, error) {
	tokenURL := "https://oauth2.googleapis.com/token"
	data := fmt.Sprintf(
		"client_id=%s&client_secret=%s&code=%s&redirect_uri=%s&grant_type=authorization_code",
		h.googleClientID,
		h.googleClientSecret,
		code,
		h.googleRedirectURL,
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

func (h *AuthHandler) getGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
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

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func (h *AuthHandler) findOrCreateGoogleUser(userInfo *GoogleUserInfo) (*User, error) {
    if userInfo == nil {
        return nil, fmt.Errorf("userInfo is nil")
    }
    
    h.logger.Printf("🔄 Finding/creating user: %s", userInfo.Email)

    if h.db == nil {
        h.logger.Printf("⚠️ Database not available, using in-memory store")
        
        for _, u := range h.userStore {
            if u.Email == userInfo.Email {
                return &u, nil
            }
        }
        
        newUser := User{
            ID:                 uuid.New().String(),
            Email:              userInfo.Email,
            Name:               userInfo.Name,
            GoogleID:           userInfo.ID,
            Avatar:             userInfo.Picture,
            Provider:           "google",
            Status:             "active",
            Plan:               "free",
            CreatedAt:          time.Now(),
            MaxWebsites:        1,
            FreeTrialUsed:      false,
            FreeTrialStartDate: time.Now(),
            FreeTrialEndDate:   time.Now().Add(7 * 24 * time.Hour),
        }
        h.userStore[newUser.ID] = newUser
        h.logger.Printf("✅ New in-memory user created: %s", newUser.ID)
        return &newUser, nil
    }

    var user User

    err := h.db.Where("google_id = ?", userInfo.ID).First(&user).Error
    if err == nil {
        h.logger.Printf("✅ User found by Google ID: %s", user.Email)
        if user.Avatar != userInfo.Picture && userInfo.Picture != "" {
            user.Avatar = userInfo.Picture
            h.db.Save(&user)
        }
        return &user, nil
    }

    err = h.db.Where("email = ?", userInfo.Email).First(&user).Error
    if err == nil {
        h.logger.Printf("✅ User found by email: %s", user.Email)
        user.GoogleID = userInfo.ID
        user.Avatar = userInfo.Picture
        user.Provider = "google"
        h.db.Save(&user)
        return &user, nil
    }

    newUser := User{
        ID:                 uuid.New().String(),
        Email:              userInfo.Email,
        Name:               userInfo.Name,
        GoogleID:           userInfo.ID,
        Avatar:             userInfo.Picture,
        Provider:           "google",
        Status:             "active",
        Plan:               "free",
        CreatedAt:          time.Now(),
        MaxWebsites:        1,
        FreeTrialUsed:      false,
        FreeTrialStartDate: time.Now(),
        FreeTrialEndDate:   time.Now().Add(7 * 24 * time.Hour),
    }

    err = h.db.Create(&newUser).Error
    if err != nil {
        h.logger.Printf("❌ Failed to create Google user: %v", err)
        return nil, err
    }

    h.logger.Printf("✅ New Google user created with UUID: %s", newUser.ID)
    return &newUser, nil
}

func (h *AuthHandler) createAuthResponse(success bool, token, message string, user *User) AuthResponse {
	response := AuthResponse{
		Success: success,
		Token:   token,
		Message: message,
	}
	if user != nil {
		response.User.ID = user.ID
		response.User.Email = user.Email
		response.User.Name = user.Name
		response.User.Avatar = user.Avatar
		response.User.Provider = user.Provider
		response.User.Plan = user.Plan
	}
	return response
}

func (h *AuthHandler) sendJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Printf("❌ Error encoding JSON response: %v", err)
	}
}

func (h *AuthHandler) sendErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(AuthResponse{
		Success: false,
		Message: message,
	})
}

// =============================================
// UTILITY FUNCTIONS
// =============================================

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length/2+1)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return false
}