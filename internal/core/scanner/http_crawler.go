package scanner

import (
    "net/http"
    "time"
)

type HTTPCrawler struct {
    client *http.Client
}

func NewHTTPCrawler() *HTTPCrawler {
    return &HTTPCrawler{
        client: &http.Client{
            Timeout: 30 * time.Second,
            CheckRedirect: func(req *http.Request, via []*http.Request) error {
                if len(via) >= 10 {
                    return http.ErrUseLastResponse
                }
                return nil
            },
        },
    }
}

func (c *HTTPCrawler) Crawl(url string, maxPages int) (map[string]*PageData, error) {
    // Simple implementation
    results := make(map[string]*PageData)
    
    // Fetch the main page
    resp, err := c.client.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    // Create page data
    pageData := &PageData{
        URL:        url,
        StatusCode: resp.StatusCode,
    }
    
    results[url] = pageData
    
    return results, nil
}