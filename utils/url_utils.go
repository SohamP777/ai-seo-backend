package utils

import (
	"net/url"
	"path"
	"strings"
)

// NormalizeURL normalizes a URL
func NormalizeURL(rawURL string) (string, error) {
	// Add scheme if missing
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}
	
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	
	// Remove default ports
	if parsed.Port() == "80" && parsed.Scheme == "http" {
		parsed.Host = strings.TrimSuffix(parsed.Host, ":80")
	}
	if parsed.Port() == "443" && parsed.Scheme == "https" {
		parsed.Host = strings.TrimSuffix(parsed.Host, ":443")
	}
	
	// Remove trailing slash
	parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	
	// Lowercase scheme and host
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	
	return parsed.String(), nil
}

// GetBaseURL returns the base URL (scheme + host)
func GetBaseURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	
	return parsed.Scheme + "://" + parsed.Host, nil
}

// IsSameDomain checks if two URLs are from the same domain
func IsSameDomain(url1, url2 string) bool {
	base1, err1 := GetBaseURL(url1)
	base2, err2 := GetBaseURL(url2)
	
	if err1 != nil || err2 != nil {
		return false
	}
	
	return base1 == base2
}

// ResolveURL resolves a relative URL to an absolute URL
func ResolveURL(base, relative string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	
	relativeURL, err := url.Parse(relative)
	if err != nil {
		return "", err
	}
	
	return baseURL.ResolveReference(relativeURL).String(), nil
}

// GetPathDepth returns the depth of a URL path
func GetPathDepth(rawURL string) int {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return 0
	}
	
	// Clean path
	cleanPath := path.Clean(parsed.Path)
	if cleanPath == "/" || cleanPath == "." {
		return 0
	}
	
	// Count segments
	segments := strings.Split(strings.Trim(cleanPath, "/"), "/")
	return len(segments)
}

// GetURLParameters returns query parameters as a map
func GetURLParameters(rawURL string) (map[string]string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	
	params := make(map[string]string)
	for key, values := range parsed.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	
	return params, nil
}

// IsInternalLink checks if a link is internal to the domain
func IsInternalLink(link, domain string) bool {
	// Handle relative links
	if strings.HasPrefix(link, "/") {
		return true
	}
	
	// Check if it's the same domain
	parsedLink, err := url.Parse(link)
	if err != nil {
		return false
	}
	
	return parsedLink.Host == domain || strings.HasSuffix(parsedLink.Host, "."+domain)
}