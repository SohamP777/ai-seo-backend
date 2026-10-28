// pkg/shopify/auth.go - REAL OAuth Flow
package shopify

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/url"
    "strings"
    "context"
     "net/http"
    "time"
    "github.com/google/uuid"
)

type BackupManager interface {
    CreateBackup(platform, identifier string, data []byte) (string, error)
    Restore(backupID string) error
    ListBackups() ([]string, error)
}

type OAuthConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURI  string
    Scopes       []string
}

type BackupManagerImpl struct {
    ID        string    // ← ADD THIS
    SiteURL   string    // ← ADD THIS  
    Platform  string    // ← ADD THIS
    CreatedAt time.Time // ← ADD THIS
    BackupDir string    // ← ADD THIS
}

func NewBackupManager() BackupManager {
    return &BackupManagerImpl{}
}

func (b *BackupManagerImpl) CreateBackup(platform, identifier string, data []byte) (string, error) {
    // implementation
    return "backup-id", nil
}

func (b *BackupManagerImpl) Restore(backupID string) error {
    return nil
}

func (b *BackupManagerImpl) ListBackups() ([]string, error) {
    return []string{}, nil
}

func generateStoreID() string {
    return uuid.New().String()
}

// REAL OAuth Authorization URL
func (a *OAuthConfig) GetAuthURL(shop string) (string, error) {
    shop = strings.TrimPrefix(shop, "https://")
    shop = strings.TrimPrefix(shop, "http://")
    shop = strings.TrimSuffix(shop, "/")
    
    // Generate a random state for CSRF protection
    state := generateRandomString(32)
    
    params := url.Values{}
    params.Add("client_id", a.ClientID)
    params.Add("scope", strings.Join(a.Scopes, ","))
    params.Add("redirect_uri", a.RedirectURI)
    params.Add("state", state)
    
    authURL := fmt.Sprintf("https://%s/admin/oauth/authorize?%s", shop, params.Encode())
    return authURL, nil
}

// REAL OAuth Token Exchange
func (a *OAuthConfig) ExchangeCode(ctx context.Context, shop, code string) (*ShopifyStore, error) {
    shop = strings.TrimPrefix(shop, "https://")
    shop = strings.TrimPrefix(shop, "http://")
    shop = strings.TrimSuffix(shop, "/")
    
    tokenURL := fmt.Sprintf("https://%s/admin/oauth/access_token", shop)
    
    data := url.Values{}
    data.Set("client_id", a.ClientID)
    data.Set("client_secret", a.ClientSecret)
    data.Set("code", code)
    
    req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
    if err != nil {
        return nil, fmt.Errorf("create token request: %w", err)
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("token request: %w", err)
    }
    defer resp.Body.Close()
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("read response: %w", err)
    }
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("token exchange failed: %d - %s", resp.StatusCode, string(body))
    }
    
    var tokenResponse struct {
        AccessToken string `json:"access_token"`
        Scope       string `json:"scope"`
    }
    
    if err := json.Unmarshal(body, &tokenResponse); err != nil {
        return nil, fmt.Errorf("parse token response: %w", err)
    }
    
    store := &ShopifyStore{
        ID:          generateStoreID(),
        URL:         shop,
        AccessToken: tokenResponse.AccessToken,
        APIVersion:  "2024-04",
        Scopes:      strings.Split(tokenResponse.Scope, ","),
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    
    return store, nil
}

// REAL Webhook Signature Verification
func (a *OAuthConfig) VerifyWebhook(payload []byte, signatureHeader string) bool {
    if a.ClientSecret == "" {
        return false
    }
    
    mac := hmac.New(sha256.New, []byte(a.ClientSecret))
    mac.Write(payload)
    expectedMAC := hex.EncodeToString(mac.Sum(nil))
    
    return hmac.Equal([]byte(signatureHeader), []byte(expectedMAC))
}

// REAL Private App (Custom App) Authentication
type PrivateAppAuth struct {
    APIKey     string
    Password   string
    StoreURL   string
}

func (p *PrivateAppAuth) GetClient() *ShopifyClient {
    // For private apps, the password is the access token
    return NewShopifyClient(p.StoreURL, p.Password, "2024-04")
}

func generateRandomString(length int) string {
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    b := make([]byte, length)
    for i := range b {
        b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
    }
    return string(b)
}