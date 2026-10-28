package models

import "time"

type ScanRequest struct {
    URL string `json:"url"`
}

type Page struct {
    URL          string   `json:"url"`
    Title        string   `json:"title"`
    TitleLength  int      `json:"title_length"`
    Description  string   `json:"description"`
    DescLength   int      `json:"desc_length"`
    Headings     []string `json:"headings"`
    Images       []Image  `json:"images"`
    Links        []Link   `json:"links"`
    WordCount    int      `json:"word_count"`
}

type Image struct {
    Src     string `json:"src"`
    Alt     string `json:"alt"`
    HasAlt  bool   `json:"has_alt"`
}

type Link struct {
    Href  string `json:"href"`
    Text  string `json:"text"`
    Title string `json:"title"`
}

type ScanResult struct {
    URL            string   `json:"url"`
    PagesScanned   int      `json:"pages_scanned"`
    SEOScore       int      `json:"seo_score"`
    Issues         []Issue  `json:"issues"`
    Fixes          []string `json:"fixes"`
    Recommendations []string `json:"recommendations"`
    ScanTime       time.Time `json:"scan_time"`
}

type Issue struct {
    Type     string `json:"type"`
    Severity string `json:"severity"` // critical, warning, info
    Message  string `json:"message"`
    Fix      string `json:"fix"`
}

type FixRequest struct {
    URL  string   `json:"url"`
    Fixes []string `json:"fixes"` // which fixes to apply
}