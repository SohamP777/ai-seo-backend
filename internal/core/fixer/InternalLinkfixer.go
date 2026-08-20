package fixer

import (
	"time"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"log"

)

	type Page struct {
    URL          string   `json:"url"`
    Title        string   `json:"title"`
    Depth        int      `json:"depth"`
    InboundLinks int      `json:"inbound_links"`
    OutboundLinks int     `json:"outbound_links"`
    Keywords     []string `json:"keywords"`
}

type LinkIssue struct {
    Type     string `json:"type"`
    Page     string `json:"page"`
    Severity string `json:"severity"`
    Fix      string `json:"fix"`
}

type LinkSuggestion struct {
    From      string  `json:"from"`
    To        string  `json:"to"`
    Anchor    string  `json:"anchor"`
    Relevance float64 `json:"relevance"`
}

type Report struct {
    URL         string           `json:"url"`
    TotalPages  int              `json:"total_pages"`
    OrphanPages []string         `json:"orphan_pages"`
    DeepPages   []string         `json:"deep_pages"`
    Issues      []LinkIssue      `json:"issues"`
    Suggestions []LinkSuggestion `json:"suggestions"`
    Score       int              `json:"score"`
}

type Crawler struct {
    baseURL  string
    pages    map[string]*Page
    links    map[string][]string
    mu       sync.Mutex
    client   *http.Client
}

type InternalLinkOptimizer struct {
    Client *http.Client
	logger *log.Logger
}


func NewInternalLinkOptimizer(logger *log.Logger) *InternalLinkOptimizer {
    return &InternalLinkOptimizer{logger: logger}
}

func NewCrawler(baseURL string) *Crawler {
	return &Crawler{
		baseURL: baseURL,
		pages:   make(map[string]*Page),
		links:   make(map[string][]string),
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Crawler) Crawl(maxPages int) {
	c.crawlPage(c.baseURL, 0, maxPages)
}

func (c *Crawler) crawlPage(pageURL string, depth int, maxPages int) {
	c.mu.Lock()
	if len(c.pages) >= maxPages {
		c.mu.Unlock()
		return
	}
	if _, exists := c.pages[pageURL]; exists {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	resp, err := c.client.Get(pageURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	content := string(body)
	
	titleRegex := regexp.MustCompile(`<title>(.*?)</title>`)
	title := ""
	if matches := titleRegex.FindStringSubmatch(content); len(matches) > 1 {
		title = matches[1]
	}

	linkRegex := regexp.MustCompile(`<a[^>]+href=["']([^"']+)["']`)
	matches := linkRegex.FindAllStringSubmatch(content, -1)
	
	foundLinks := make([]string, 0)
	for _, match := range matches {
		if len(match) > 1 {
			link := match[1]
			if strings.HasPrefix(link, "/") {
				link = c.baseURL + link
			}
			if strings.Contains(link, strings.Split(c.baseURL, "://")[1]) {
				foundLinks = append(foundLinks, link)
			}
		}
	}

	words := strings.Fields(strings.ToLower(content))
	wordCount := make(map[string]int)
	stopWords := map[string]bool{"the":true, "a":true, "an":true, "and":true, "or":true, "but":true, "in":true, "on":true, "at":true, "to":true, "for":true, "of":true, "with":true, "is":true, "are":true}
	
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:()[]{}")
		if len(word) > 3 && !stopWords[word] {
			wordCount[word]++
		}
	}
	
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range wordCount {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})
	
	keywords := make([]string, 0, 5)
	for i := 0; i < len(sorted) && i < 5; i++ {
		keywords = append(keywords, sorted[i].Key)
	}

	c.mu.Lock()
	c.pages[pageURL] = &Page{
		URL:           pageURL,
		Title:         title,
		Depth:         depth,
		OutboundLinks: len(foundLinks),
		Keywords:      keywords,
	}
	c.links[pageURL] = foundLinks
	c.mu.Unlock()

	for _, link := range foundLinks {
		c.crawlPage(link, depth+1, maxPages)
	}
}

func (c *Crawler) calculateInbound() {
	for _, links := range c.links {
		for _, link := range links {
			if page, exists := c.pages[link]; exists {
				page.InboundLinks++
			}
		}
	}
}

func (c *Crawler) calculateRelevance(page1, page2 *Page) float64 {
	matches := 0
	for _, kw1 := range page1.Keywords {
		for _, kw2 := range page2.Keywords {
			if kw1 == kw2 {
				matches++
			}
		}
	}
	
	if len(page1.Keywords) == 0 || len(page2.Keywords) == 0 {
		return 0
	}
	
	return float64(matches) / float64(len(page1.Keywords)+len(page2.Keywords)) * 2
}

func (c *Crawler) GenerateReport() *Report {
	c.calculateInbound()
	
	orphans := make([]string, 0)
	deepPages := make([]string, 0)
	issues := make([]LinkIssue, 0)
	suggestions := make([]LinkSuggestion, 0)
	
	for url, page := range c.pages {
		if page.InboundLinks == 0 && url != c.baseURL {
			orphans = append(orphans, url)
			issues = append(issues, LinkIssue{
				Type:     "orphan",
				Page:     url,
				Severity: "high",
				Fix:      fmt.Sprintf("Add internal links from related pages to %s", url),
			})
		}
	}
	
	for url, page := range c.pages {
		if page.Depth > 3 {
			deepPages = append(deepPages, url)
			issues = append(issues, LinkIssue{
				Type:     "deep",
				Page:     url,
				Severity: "medium",
				Fix:      fmt.Sprintf("Add links from homepage or category pages to %s", url),
			})
		}
	}
	
	for url, page := range c.pages {
		if page.OutboundLinks < 2 && len(c.pages) > 10 {
			issues = append(issues, LinkIssue{
				Type:     "low_outbound",
				Page:     url,
				Severity: "low",
				Fix:      fmt.Sprintf("Add 2-3 relevant internal links from %s to related content", url),
			})
		}
	}
	
	pages := make([]*Page, 0, len(c.pages))
	for _, page := range c.pages {
		pages = append(pages, page)
	}
	
	for i := 0; i < len(pages) && len(suggestions) < 20; i++ {
		for j := i + 1; j < len(pages); j++ {
			page1 := pages[i]
			page2 := pages[j]
			
			linked := false
			for _, link := range c.links[page1.URL] {
				if link == page2.URL {
					linked = true
					break
				}
			}
			if linked {
				continue
			}
			
			relevance := c.calculateRelevance(page1, page2)
			
			if relevance > 0.3 {
				anchor := ""
				for _, kw1 := range page1.Keywords {
					for _, kw2 := range page2.Keywords {
						if kw1 == kw2 {
							anchor = kw1
							break
						}
					}
					if anchor != "" {
						break
					}
				}
				if anchor == "" && len(page2.Keywords) > 0 {
					anchor = page2.Keywords[0]
				}
				
				if page1.InboundLinks > page2.InboundLinks && page2.InboundLinks < 3 {
					suggestions = append(suggestions, LinkSuggestion{
						From:      page1.URL,
						To:        page2.URL,
						Anchor:    anchor,
						Relevance: relevance,
					})
				} else if page2.InboundLinks > page1.InboundLinks && page1.InboundLinks < 3 {
					suggestions = append(suggestions, LinkSuggestion{
						From:      page2.URL,
						To:        page1.URL,
						Anchor:    anchor,
						Relevance: relevance,
					})
				}
			}
		}
	}
	
	score := 100
	score -= len(orphans) * 5
	score -= len(deepPages) * 3
	if score < 0 {
		score = 0
	}
	
	return &Report{
		URL:         c.baseURL,
		TotalPages:  len(c.pages),
		OrphanPages: orphans,
		DeepPages:   deepPages,
		Issues:      issues,
		Suggestions: suggestions,
		Score:       score,
	}
}
