package utils

import (
    "fmt"
    "math"
    "net/http"
    "regexp"
    "sort"
    "strings"
    "time"
    "unicode"

    "github.com/PuerkitoBio/goquery"
    "golang.org/x/text/transform"
    "golang.org/x/text/unicode/norm"
)

// SEOUtils provides SEO-related utility functions
type SEOUtils struct {
    stopWords map[string]bool
}

// NewSEOUtils creates a new SEO utils instance
func NewSEOUtils() *SEOUtils {
    return &SEOUtils{
        stopWords: loadStopWords(),
    }
}

// KeywordExtractor extracts keywords from text
type KeywordExtractor struct {
    MinWordLength  int
    MaxKeywords    int
    IncludePhrases bool
    PhraseLength   int
}

// KeywordScore represents a keyword with its score
type KeywordScore struct {
    Keyword string  `json:"keyword"`
    Score   float64 `json:"score"`
    Count   int     `json:"count"`
    Density float64 `json:"density"`
}

// NewKeywordExtractor creates a new keyword extractor with defaults
func NewKeywordExtractor() *KeywordExtractor {
    return &KeywordExtractor{
        MinWordLength:  3,
        MaxKeywords:    20,
        IncludePhrases: true,
        PhraseLength:   3,
    }
}

// ExtractKeywords extracts keywords from text with TF-IDF scoring
func (ke *KeywordExtractor) ExtractKeywords(text string, totalDocs int, docFreq map[string]int) []KeywordScore {
    // Normalize text
    text = normalizeText(text)

    // Tokenize
    words := tokenize(text)

    // Count frequencies
    wordFreq := make(map[string]int)
    totalWords := 0

    for _, word := range words {
        if len(word) >= ke.MinWordLength && !isStopWord(word) {
            wordFreq[word]++
            totalWords++
        }
    }

    // Calculate scores
    var scores []KeywordScore
    for word, count := range wordFreq {
        if totalWords == 0 {
            continue
        }

        // TF (Term Frequency)
        tf := float64(count) / float64(totalWords)

        // IDF (Inverse Document Frequency)
        idf := 1.0
        if totalDocs > 0 {
            df := docFreq[word]
            if df == 0 {
                df = 1
            }
            idf = math.Log(float64(totalDocs) / float64(df))
        }

        // TF-IDF Score
        score := tf * idf

        // Density
        density := (float64(count) / float64(totalWords)) * 100

        scores = append(scores, KeywordScore{
            Keyword: word,
            Score:   score,
            Count:   count,
            Density: density,
        })
    }

    // Sort by score
    sort.Slice(scores, func(i, j int) bool {
        return scores[i].Score > scores[j].Score
    })

    // Limit results
    if len(scores) > ke.MaxKeywords {
        scores = scores[:ke.MaxKeywords]
    }

    // Extract phrases if enabled
    if ke.IncludePhrases && ke.PhraseLength > 1 {
        phrases := ke.extractPhrases(text, ke.PhraseLength)
        scores = append(scores, phrases...)
    }

    return scores
}

// ExtractFromHTML extracts keywords from HTML content
func (ke *KeywordExtractor) ExtractFromHTML(html string, totalDocs int, docFreq map[string]int) []KeywordScore {
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
    if err != nil {
        return []KeywordScore{}
    }

    // Extract text from important elements
    var contentBuilder strings.Builder

    // Title (highest weight)
    doc.Find("title").Each(func(i int, s *goquery.Selection) {
        contentBuilder.WriteString(" " + strings.Repeat(s.Text()+" ", 3))
    })

    // Headings (high weight)
    doc.Find("h1").Each(func(i int, s *goquery.Selection) {
        contentBuilder.WriteString(" " + strings.Repeat(s.Text()+" ", 2))
    })
    doc.Find("h2, h3").Each(func(i int, s *goquery.Selection) {
        contentBuilder.WriteString(" " + s.Text())
    })

    // Meta description
    doc.Find("meta[name='description']").Each(func(i int, s *goquery.Selection) {
        if metaContent, exists := s.Attr("content"); exists {
            contentBuilder.WriteString(" " + metaContent)
        }
    })

    // Main content
    doc.Find("p, li, article, .content, #content").Each(func(i int, s *goquery.Selection) {
        contentBuilder.WriteString(" " + s.Text())
    })

    return ke.ExtractKeywords(contentBuilder.String(), totalDocs, docFreq)
}

// extractPhrases extracts multi-word phrases
func (ke *KeywordExtractor) extractPhrases(text string, phraseLength int) []KeywordScore {
    words := tokenize(text)
    if len(words) < phraseLength {
        return []KeywordScore{}
    }

    phraseFreq := make(map[string]int)

    for i := 0; i <= len(words)-phraseLength; i++ {
        phrase := strings.Join(words[i:i+phraseLength], " ")
        if !containsStopWord(phrase) {
            phraseFreq[phrase]++
        }
    }

    var phrases []KeywordScore
    totalWords := len(words)

    for phrase, count := range phraseFreq {
        density := (float64(count) / float64(totalWords)) * 100
        phrases = append(phrases, KeywordScore{
            Keyword: phrase,
            Count:   count,
            Density: density,
            Score:   float64(count) * density, // Simple scoring for phrases
        })
    }

    // Sort by frequency
    sort.Slice(phrases, func(i, j int) bool {
        return phrases[i].Count > phrases[j].Count
    })

    if len(phrases) > 5 {
        phrases = phrases[:5]
    }

    return phrases
}

// KeywordDensityAnalyzer analyzes keyword density
type KeywordDensityAnalyzer struct {
    MinDensity float64
    MaxDensity float64
}

// NewKeywordDensityAnalyzer creates a new density analyzer
func NewKeywordDensityAnalyzer() *KeywordDensityAnalyzer {
    return &KeywordDensityAnalyzer{
        MinDensity: 1.0,
        MaxDensity: 2.5,
    }
}

// DensityAnalysis represents keyword density analysis results
type DensityAnalysis struct {
    Keyword        string   `json:"keyword"`
    Count          int      `json:"count"`
    TotalWords     int      `json:"total_words"`
    Density        float64  `json:"density"`
    IsOptimal      bool     `json:"is_optimal"`
    Suggestions    []string `json:"suggestions"`
    RelatedKeywords []string `json:"related_keywords"`
}

// Analyze analyzes keyword density in text
func (da *KeywordDensityAnalyzer) Analyze(text, targetKeyword string) *DensityAnalysis {
    words := tokenize(text)
    targetWords := tokenize(targetKeyword)

    // Count exact matches
    exactMatches := 0
    partialMatches := 0

    for i := 0; i <= len(words)-len(targetWords); i++ {
        match := true
        for j, word := range targetWords {
            if i+j >= len(words) || words[i+j] != word {
                match = false
                break
            }
        }
        if match {
            exactMatches++
        }
    }

    // Count partial matches (for related terms)
    for _, word := range words {
        for _, targetWord := range targetWords {
            if strings.Contains(word, targetWord) || strings.Contains(targetWord, word) {
                partialMatches++
                break
            }
        }
    }

    totalWords := len(words)
    density := 0.0
    if totalWords > 0 {
        density = (float64(exactMatches) / float64(totalWords)) * 100
    }

    analysis := &DensityAnalysis{
        Keyword:     targetKeyword,
        Count:       exactMatches,
        TotalWords:  totalWords,
        Density:     density,
        IsOptimal:   density >= da.MinDensity && density <= da.MaxDensity,
        Suggestions: []string{},
    }

    // Generate suggestions
    if density < da.MinDensity {
        analysis.Suggestions = append(analysis.Suggestions,
            fmt.Sprintf("Increase '%s' usage (current: %.1f%%, target: %.1f-%.1f%%)",
                targetKeyword, density, da.MinDensity, da.MaxDensity))
        analysis.Suggestions = append(analysis.Suggestions,
            "Add to H1 tag and first paragraph")
        analysis.Suggestions = append(analysis.Suggestions,
            "Use in image alt text and meta description")
    } else if density > da.MaxDensity {
        analysis.Suggestions = append(analysis.Suggestions,
            fmt.Sprintf("Reduce '%s' usage to avoid over-optimization", targetKeyword))
        analysis.Suggestions = append(analysis.Suggestions,
            "Use synonyms and related terms instead")
    } else {
        analysis.Suggestions = append(analysis.Suggestions,
            "Keyword density is optimal")
    }

    // Add related keywords suggestion
    if partialMatches > exactMatches*2 {
        analysis.Suggestions = append(analysis.Suggestions,
            "Consider using more exact matches of your target keyword")
    }

    return analysis
}

// SERPAnalyzer analyzes SERP features
type SERPAnalyzer struct {
    client *http.Client
}

// SERPFeature represents a SERP feature
type SERPFeature struct {
    Type        string `json:"type"`
    Present     bool   `json:"present"`
    Opportunity bool   `json:"opportunity"`
    Suggestion  string `json:"suggestion"`
}

// NewSERPAnalyzer creates a new SERP analyzer
func NewSERPAnalyzer() *SERPAnalyzer {
    return &SERPAnalyzer{
        client: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

// AnalyzeSERPFeatures analyzes SERP features for a keyword
func (sa *SERPAnalyzer) AnalyzeSERPFeatures(keyword string) ([]SERPFeature, error) {
    // This would normally query a SERP API
    // For now, return simulated analysis
    features := []SERPFeature{
        {
            Type:        "featured_snippet",
            Present:     false,
            Opportunity: true,
            Suggestion:  "Create a concise answer to a common question about " + keyword,
        },
        {
            Type:        "people_also_ask",
            Present:     true,
            Opportunity: true,
            Suggestion:  "Address related questions in your content",
        },
        {
            Type:        "image_pack",
            Present:     false,
            Opportunity: true,
            Suggestion:  "Add relevant images with alt text containing " + keyword,
        },
        {
            Type:        "video_carousel",
            Present:     false,
            Opportunity: false,
            Suggestion:  "Consider creating video content for better visibility",
        },
        {
            Type:        "local_pack",
            Present:     false,
            Opportunity: false,
            Suggestion:  "Optimize for local SEO if applicable",
        },
        {
            Type:        "knowledge_panel",
            Present:     false,
            Opportunity: false,
            Suggestion:  "Build authority to appear in knowledge panels",
        },
    }

    return features, nil
}

// CompetitorGapAnalyzer analyzes competitor keyword gaps
type CompetitorGapAnalyzer struct {
    utils *SEOUtils
}

// Gap represents a keyword gap
type Gap struct {
    Keyword         string  `json:"keyword"`
    YourCount       int     `json:"your_count"`
    CompetitorCount int     `json:"competitor_count"`
    Opportunity     float64 `json:"opportunity"`
}

// NewCompetitorGapAnalyzer creates a new gap analyzer
func NewCompetitorGapAnalyzer() *CompetitorGapAnalyzer {
    return &CompetitorGapAnalyzer{
        utils: NewSEOUtils(),
    }
}

// AnalyzeGaps analyzes keyword gaps between you and competitors
func (cga *CompetitorGapAnalyzer) AnalyzeGaps(yourKeywords, competitorKeywords map[string]int) []Gap {
    var gaps []Gap

    // Find keywords competitors have but you don't
    for kw, compCount := range competitorKeywords {
        yourCount := yourKeywords[kw]

        if yourCount == 0 {
            opportunity := float64(compCount) * 0.5
            if isCommercialKeyword(kw) {
                opportunity *= 1.5
            }

            gaps = append(gaps, Gap{
                Keyword:         kw,
                YourCount:       0,
                CompetitorCount: compCount,
                Opportunity:     opportunity,
            })
        }
    }

    // Sort by opportunity
    sort.Slice(gaps, func(i, j int) bool {
        return gaps[i].Opportunity > gaps[j].Opportunity
    })

    // Limit to top 20
    if len(gaps) > 20 {
        gaps = gaps[:20]
    }

    return gaps
}

// Helper functions

func normalizeText(text string) string {
    // Remove accents/diacritics
    t := transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
        return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
    }), norm.NFC)

    result, _, _ := transform.String(t, text)

    // Convert to lowercase
    result = strings.ToLower(result)

    // Remove special characters
    re := regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
    result = re.ReplaceAllString(result, " ")

    // Remove extra spaces
    re = regexp.MustCompile(`\s+`)
    result = re.ReplaceAllString(result, " ")

    return strings.TrimSpace(result)
}

func tokenize(text string) []string {
    return strings.Fields(text)
}

func loadStopWords() map[string]bool {
    return map[string]bool{
        "the": true, "and": true, "for": true, "you": true, "with": true,
        "are": true, "this": true, "that": true, "have": true, "from": true,
        "was": true, "were": true, "will": true, "been": true, "has": true,
        "had": true, "can": true, "all": true, "any": true, "each": true,
        "some": true, "such": true, "then": true, "them": true,
        "these": true, "they": true, "their": true, "there": true, "what": true,
        "which": true, "when": true, "where": true, "who": true, "whom": true,
        "would": true, "could": true, "should": true, "may": true, "might": true,
        "must": true, "shall": true, "about": true, "into": true, "through": true,
        "during": true, "before": true, "after": true, "above": true, "below": true,
        "up": true, "down": true, "out": true, "off": true, "over": true,
        "under": true, "again": true, "further": true, "once": true,
    }
}

func isStopWord(word string) bool {
    stopWords := loadStopWords()
    return stopWords[word]
}

func containsStopWord(phrase string) bool {
    words := strings.Fields(phrase)
    stopWords := loadStopWords()

    for _, word := range words {
        if stopWords[word] {
            return true
        }
    }
    return false
}

func isCommercialKeyword(keyword string) bool {
    commercialTerms := []string{"buy", "price", "cost", "cheap", "discount", "deal", "offer"}
    keyword = strings.ToLower(keyword)

    for _, term := range commercialTerms {
        if strings.Contains(keyword, term) {
            return true
        }
    }
    return false
}