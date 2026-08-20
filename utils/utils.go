package utils

import (
	"context"
	"io"
	"net/http"
	"time"
	"log"
    "os"
)

type Logger struct {
    *log.Logger
}

// NewLogger creates a new logger
func NewLogger(prefix string) *Logger {
    return &Logger{
        Logger: log.New(os.Stdout, prefix+" ", log.LstdFlags|log.Lshortfile),
    }
}

// HTMLFetcher with retry logic
type HTMLFetcher struct {
	client  *http.Client
	logger  *Logger
	retries int
}

func NewHTMLFetcher(logger *Logger) *HTMLFetcher {
	return &HTMLFetcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:  logger,
		retries: 3,
	}
}

func (f *HTMLFetcher) FetchHTML(ctx context.Context, url string) (string, error) {
	var lastErr error
	
	for i := 0; i < f.retries; i++ {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return "", err
		}
		
		req.Header.Set("User-Agent", "SEO-SPS-Bot/1.0")
		
		resp, err := f.client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		defer resp.Body.Close()
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}
		
		return string(body), nil
	}
	
	return "", lastErr
}