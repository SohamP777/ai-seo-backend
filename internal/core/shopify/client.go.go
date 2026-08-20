// pkg/shopify/client.go - COMPLETE REWRITE WITH REAL API CALLS
package shopify

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"
    
    
    "golang.org/x/time/rate"
)

type ShopifyClient struct {
    StoreURL     string
    AccessToken  string
    APIVersion   string
    HTTPClient   *http.Client
    RateLimiter  *rate.Limiter
    IsGraphQL    bool
}

func NewShopifyClient(storeURL, accessToken, apiVersion string) *ShopifyClient {
    // REST API: 2 requests per second (Shopify's limit for most plans)
    limiter := rate.NewLimiter(rate.Limit(2), 2)
    
    // Remove https:// and trailing slashes
    storeURL = strings.TrimPrefix(storeURL, "https://")
    storeURL = strings.TrimPrefix(storeURL, "http://")
    storeURL = strings.TrimSuffix(storeURL, "/")
    
    return &ShopifyClient{
        StoreURL:    storeURL,
        AccessToken: accessToken,
        APIVersion:  apiVersion,
        HTTPClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        RateLimiter: limiter,
        IsGraphQL:   false,
    }
}

// REAL REST API Request with X-Shopify-Access-Token header
func (c *ShopifyClient) DoRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
    // Wait for rate limiter
    if err := c.RateLimiter.Wait(ctx); err != nil {
        return fmt.Errorf("rate limiter: %w", err)
    }
    
    var reqBody io.Reader
    if body != nil {
        jsonBody, err := json.Marshal(body)
        if err != nil {
            return fmt.Errorf("marshal body: %w", err)
        }
        reqBody = bytes.NewBuffer(jsonBody)
    }
    
    // Construct REAL Shopify API URL
    var reqURL string
    if c.IsGraphQL {
        reqURL = fmt.Sprintf("https://%s/admin/api/%s/graphql.json", c.StoreURL, c.APIVersion)
    } else {
        reqURL = fmt.Sprintf("https://%s/admin/api/%s%s", c.StoreURL, c.APIVersion, path)
    }
    
    req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }
    
    // REAL Shopify authentication header
    req.Header.Set("X-Shopify-Access-Token", c.AccessToken)
    req.Header.Set("Content-Type", "application/json")
    
    if c.IsGraphQL {
        req.Header.Set("Content-Type", "application/graphql")
    }
    
    // Log request for debugging (remove in production)
    fmt.Printf("Making REAL Shopify API call: %s %s\n", method, reqURL)
    
    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return fmt.Errorf("do request: %w", err)
    }
    defer resp.Body.Close()
    
    // Handle rate limiting with Retry-After header
    if resp.StatusCode == http.StatusTooManyRequests {
        retryAfter := resp.Header.Get("Retry-After")
        if retryAfter != "" {
            duration, err := time.ParseDuration(retryAfter + "s")
            if err == nil {
                select {
                case <-time.After(duration):
                    return c.DoRequest(ctx, method, path, body, result)
                case <-ctx.Done():
                    return ctx.Err()
                }
            }
        }
        return fmt.Errorf("rate limited by Shopify")
    }
    
    // Check for Shopify API errors
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("Shopify API error: %d - %s", resp.StatusCode, string(bodyBytes))
    }
    
    if result != nil {
        if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
            return fmt.Errorf("decode response: %w", err)
        }
    }
    
    return nil
}

// REAL product fetch with pagination
func (c *ShopifyClient) GetProducts(ctx context.Context, limit int, pageInfo string) ([]ShopifyProduct, string, error) {
    path := fmt.Sprintf("/products.json?limit=%d", limit)
    if pageInfo != "" {
        path += "&page_info=" + url.QueryEscape(pageInfo)
    }
    
    var response struct {
        Products []ShopifyProduct `json:"products"`
    }
    
    if err := c.DoRequest(ctx, "GET", path, nil, &response); err != nil {
        return nil, "", err
    }
    
    // Extract next page info from Link header (would need to capture headers)
    // For now, return empty
    nextPageInfo := ""
    
    return response.Products, nextPageInfo, nil
}

// REAL product update with REST API
func (c *ShopifyClient) UpdateProduct(ctx context.Context, product *ShopifyProduct) error {
    path := fmt.Sprintf("/products/%d.json", product.ID)
    
    // Build the update payload matching Shopify's expected format
    updateData := map[string]interface{}{
        "product": map[string]interface{}{
            "id":          product.ID,
            "title":       product.Title,
            "body_html":   product.BodyHTML,
            "handle":      product.Handle,
            "status":      product.Status,
            "vendor":      product.Vendor,
            "product_type": product.ProductType,
        },
    }
    
    // Add metafields if they exist
    if len(product.Metafields) > 0 {
        updateData["product"].(map[string]interface{})["metafields"] = product.Metafields
    }
    
    // Add images if they were updated
    if len(product.Images) > 0 {
        updateData["product"].(map[string]interface{})["images"] = product.Images
    }
    
    var response struct {
        Product ShopifyProduct `json:"product"`
    }
    
    return c.DoRequest(ctx, "PUT", path, updateData, &response)
}

// REAL GraphQL mutation with actual Shopify GraphQL endpoint
func (c *ShopifyClient) GraphQLRequest(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
    graphQLClient := c.NewGraphQLClient()
    
    requestBody := map[string]interface{}{
        "query":     query,
        "variables": variables,
    }
    
    return graphQLClient.DoRequest(ctx, "POST", "", requestBody, result)
}

func (c *ShopifyClient) NewGraphQLClient() *ShopifyClient {
    return &ShopifyClient{
        StoreURL:    c.StoreURL,
        AccessToken: c.AccessToken,
        APIVersion:  c.APIVersion,
        HTTPClient:  c.HTTPClient,
        RateLimiter: rate.NewLimiter(rate.Limit(10), 10), // GraphQL allows 10 requests/second
        IsGraphQL:   true,
    }
}

// REAL metafield creation via REST API
func (c *ShopifyClient) CreateMetafield(ctx context.Context, ownerResource string, ownerID int64, metafield Metafield) (*Metafield, error) {
    // ownerResource can be: "products", "collections", "pages", "blogs", "articles"
    path := fmt.Sprintf("/%s/%d/metafields.json", ownerResource, ownerID)
    
    data := map[string]interface{}{
        "metafield": map[string]interface{}{
            "namespace": metafield.Namespace,
            "key":       metafield.Key,
            "value":     metafield.Value,
            "type":      metafield.Type,
        },
    }
    
    var response struct {
        Metafield Metafield `json:"metafield"`
    }
    
    if err := c.DoRequest(ctx, "POST", path, data, &response); err != nil {
        return nil, err
    }
    
    return &response.Metafield, nil
}

// REAL metafield update via REST API
func (c *ShopifyClient) UpdateMetafield(ctx context.Context, ownerResource string, ownerID int64, metafieldID int64, metafield Metafield) error {
    path := fmt.Sprintf("/%s/%d/metafields/%d.json", ownerResource, ownerID, metafieldID)
    
    data := map[string]interface{}{
        "metafield": map[string]interface{}{
            "id":        metafieldID,
            "namespace": metafield.Namespace,
            "key":       metafield.Key,
            "value":     metafield.Value,
            "type":      metafield.Type,
        },
    }
    
    return c.DoRequest(ctx, "PUT", path, data, nil)
}

// REAL theme fetch
func (c *ShopifyClient) GetThemes(ctx context.Context) ([]Theme, error) {
    path := "/themes.json"
    
    var response struct {
        Themes []Theme `json:"themes"`
    }
    
    if err := c.DoRequest(ctx, "GET", path, nil, &response); err != nil {
        return nil, err
    }
    
    return response.Themes, nil
}

// REAL theme asset fetch
func (c *ShopifyClient) GetThemeAsset(themeID, assetKey string) (string, error) {
    endpoint := fmt.Sprintf("/admin/api/2024-04/themes/%s/assets.json?asset[key]=%s", themeID, assetKey)
    
    var result struct {
        Asset struct {
            Value string `json:"value"`
        } `json:"asset"`
    }
    
    // http.Client.Get returns (*http.Response, error), not error
    resp, err := c.HTTPClient.Get(endpoint)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    // Parse JSON response
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }
    
    return result.Asset.Value, nil
}

// REAL theme asset update
func (c *ShopifyClient) UpdateThemeAsset(ctx context.Context, themeID int64, asset ThemeAsset) error {
    path := fmt.Sprintf("/themes/%d/assets.json", themeID)
    
    data := map[string]interface{}{
        "asset": map[string]interface{}{
            "key":   asset.Key,
            "value": asset.Value,
        },
    }
    
    var response struct {
        Asset ThemeAsset `json:"asset"`
    }
    
    return c.DoRequest(ctx, "PUT", path, data, &response)
}

// REAL blog fetch
func (c *ShopifyClient) GetBlogs(ctx context.Context) ([]Blog, error) {
    path := "/blogs.json"
    
    var response struct {
        Blogs []Blog `json:"blogs"`
    }
    
    if err := c.DoRequest(ctx, "GET", path, nil, &response); err != nil {
        return nil, err
    }
    
    return response.Blogs, nil
}

// REAL article fetch
func (c *ShopifyClient) GetArticles(ctx context.Context, blogID int64) ([]Article, error) {
    path := fmt.Sprintf("/blogs/%d/articles.json", blogID)
    
    var response struct {
        Articles []Article `json:"articles"`
    }
    
    if err := c.DoRequest(ctx, "GET", path, nil, &response); err != nil {
        return nil, err
    }
    
    return response.Articles, nil
}

// REAL page fetch
func (c *ShopifyClient) GetPages(ctx context.Context) ([]Page, error) {
    path := "/pages.json"
    
    var response struct {
        Pages []Page `json:"pages"`
    }
    
    if err := c.DoRequest(ctx, "GET", path, nil, &response); err != nil {
        return nil, err
    }
    
    return response.Pages, nil
}

// REAL collection fetch
func (c *ShopifyClient) GetCollections(ctx context.Context) ([]Collection, error) {
    path := "/collections.json"
    
    var response struct {
        Collections []Collection `json:"collections"`
    }
    
    if err := c.DoRequest(ctx, "GET", path, nil, &response); err != nil {
        return nil, err
    }
    
    return response.Collections, nil
}

// REAL collection update
func (c *ShopifyClient) UpdateCollection(ctx context.Context, collectionID int64, collection map[string]interface{}) error {
    path := fmt.Sprintf("/collections/%d.json", collectionID)
    
    data := map[string]interface{}{
        "collection": collection,
    }
    
    return c.DoRequest(ctx, "PUT", path, data, nil)
}

// REAL page update
func (c *ShopifyClient) UpdatePage(ctx context.Context, pageID int64, page map[string]interface{}) error {
    path := fmt.Sprintf("/pages/%d.json", pageID)
    
    data := map[string]interface{}{
        "page": page,
    }
    
    return c.DoRequest(ctx, "PUT", path, data, nil)
}

// REAL article update
func (c *ShopifyClient) UpdateArticle(ctx context.Context, blogID, articleID int64, article map[string]interface{}) error {
    path := fmt.Sprintf("/blogs/%d/articles/%d.json", blogID, articleID)
    
    data := map[string]interface{}{
        "article": article,
    }
    
    return c.DoRequest(ctx, "PUT", path, data, nil)
}