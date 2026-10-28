// pkg/wordpress/client.go
package wordpress

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
)

type Client struct {
     Client   *http.Client 
    baseURL     string
    httpClient  *http.Client
    auth        *Auth
    rateLimiter *RateLimiter
    logger      *Logger
}

type RateLimiter struct {
    lastRequest time.Time
    minInterval time.Duration
}

func NewRateLimiter(requestsPerSecond int) *RateLimiter {
    return &RateLimiter{
        minInterval: time.Second / time.Duration(requestsPerSecond),
    }
}

func (r *RateLimiter) Wait() {
    if r.lastRequest.IsZero() {
        r.lastRequest = time.Now()
        return
    }
    
    elapsed := time.Since(r.lastRequest)
    if elapsed < r.minInterval {
        time.Sleep(r.minInterval - elapsed)
    }
    r.lastRequest = time.Now()
}

type Logger struct {
    debug bool
}

func NewLogger(debug bool) *Logger {
    return &Logger{debug: debug}
}

func (l *Logger) Debug(format string, args ...interface{}) {
    if l.debug {
        fmt.Printf("[DEBUG] "+format+"\n", args...)
    }
}

func (l *Logger) Info(format string, args ...interface{}) {
    fmt.Printf("[INFO] "+format+"\n", args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
    fmt.Printf("[ERROR] "+format+"\n", args...)
}

func NewClient(siteURL, username, password string, debug bool) (*Client, error) {
    baseURL := strings.TrimSuffix(siteURL, "/")
    
    auth, err := NewAuth(baseURL, username, password)
    if err != nil {
        return nil, fmt.Errorf("failed to create auth: %w", err)
    }
    
    client := &Client{
        baseURL:    baseURL,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        auth:        auth,
        rateLimiter: NewRateLimiter(5), // 5 requests per second
        logger:      NewLogger(debug),
    }
    
    // Test connection
    if err := client.TestConnection(context.Background()); err != nil {
        return nil, fmt.Errorf("connection test failed: %w", err)
    }
    
    return client, nil
}

func (c *Client) TestConnection(ctx context.Context) error {
    _, err := c.Get(ctx, "/wp-json/wp/v2/users/me")
    return err
}

func (c *Client) Get(ctx context.Context, endpoint string) (map[string]interface{}, error) {
    return c.doRequest(ctx, "GET", endpoint, nil)
}

func (c *Client) Post(ctx context.Context, endpoint string, body interface{}) (map[string]interface{}, error) {
    return c.doRequest(ctx, "POST", endpoint, body)
}

func (c *Client) Put(ctx context.Context, endpoint string, body interface{}) (map[string]interface{}, error) {
    return c.doRequest(ctx, "PUT", endpoint, body)
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}) (map[string]interface{}, error) {
    c.rateLimiter.Wait()
    
    fullURL := c.baseURL + endpoint
    
    var reqBody io.Reader
    if body != nil {
        jsonBody, err := json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal body: %w", err)
        }
        reqBody = bytes.NewBuffer(jsonBody)
    }
    
    req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    c.auth.Authenticate(req)
    
    c.logger.Debug("%s %s", method, fullURL)
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == http.StatusTooManyRequests {
        // Handle rate limiting
        retryAfter := resp.Header.Get("Retry-After")
        if retryAfter != "" {
            if duration, err := time.ParseDuration(retryAfter + "s"); err == nil {
                time.Sleep(duration)
                return c.doRequest(ctx, method, endpoint, body)
            }
        }
        time.Sleep(2 * time.Second)
        return c.doRequest(ctx, method, endpoint, body)
    }
    
    if resp.StatusCode >= 400 {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
    }
    
    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }
    
    return result, nil
}

func (c *Client) GetPosts(ctx context.Context, page, perPage int) ([]WPPost, error) {
    endpoint := fmt.Sprintf("/wp-json/wp/v2/posts?page=%d&per_page=%d", page, perPage)
    
    resp, err := c.Client.Get(endpoint)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var posts []WPPost
    if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
        return nil, err
    }
    
    return posts, nil
}

func (c *Client) UpdatePost(ctx context.Context, postID int, updates map[string]interface{}) error {
    endpoint := fmt.Sprintf("/wp-json/wp/v2/posts/%d", postID)
    _, err := c.Post(ctx, endpoint, updates)
    return err
}

func (c *Client) GetSiteInfo(ctx context.Context) (map[string]interface{}, error) {
    return c.Get(ctx, "/wp-json/wp/v2")
}

func (c *Client) GetSettings(ctx context.Context) (map[string]interface{}, error) {
    return c.Get(ctx, "/wp-json/wp/v2/settings")
}

func (c *Client) UpdateSettings(ctx context.Context, settings map[string]interface{}) error {
    _, err := c.Post(ctx, "/wp-json/wp/v2/settings", settings)
    return err
}

func (c *Client) GetYoastMeta(ctx context.Context, postID int) (*YoastMeta, error) {
    endpoint := fmt.Sprintf("/wp-json/yoast/v1/get_head?post_id=%d", postID)
    result, err := c.Get(ctx, endpoint)
    if err != nil {
        return nil, err
    }
    
    meta := &YoastMeta{}
    if title, ok := result["title"].(string); ok {
        meta.Title = title
    }
    if desc, ok := result["description"].(string); ok {
        meta.Description = desc
    }
    if canonical, ok := result["canonical"].(string); ok {
        meta.Canonical = canonical
    }
    
    return meta, nil
}

func (c *Client) UpdateYoastMeta(ctx context.Context, postID int, meta *YoastMeta) error {
    endpoint := fmt.Sprintf("/wp-json/yoast/v1/update_meta/%d", postID)
    updates := map[string]interface{}{
        "yoast_wpseo_title":       meta.Title,
        "yoast_wpseo_metadesc":    meta.Description,
        "yoast_wpseo_canonical":   meta.Canonical,
        "yoast_wpseo_noindex":     meta.NoIndex,
        "yoast_wpseo_nofollow":    meta.NoFollow,
        "yoast_wpseo_opengraph-title": meta.OgTitle,
        "yoast_wpseo_opengraph-description": meta.OgDesc,
        "yoast_wpseo_opengraph-image": meta.OgImage,
        "yoast_wpseo_twitter-title": meta.TwitterTitle,
        "yoast_wpseo_twitter-description": meta.TwitterDesc,
        "yoast_wpseo_twitter-image": meta.TwitterImage,
    }
    
    _, err := c.Post(ctx, endpoint, updates)
    return err
}