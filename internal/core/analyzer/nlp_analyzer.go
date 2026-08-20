package analyzer

import (
	"strings"
)

// NLPAnalyzer provides simple NLP for SEO content analysis
type NLPAnalyzer struct {
	commonWords map[string]bool
}

// NewNLPAnalyzer creates a new NLP analyzer
func NewNLPAnalyzer() *NLPAnalyzer {
	// Common English words to ignore in keyword analysis
	commonWords := map[string]bool{
		"the": true, "be": true, "to": true, "of": true, "and": true,
		"a": true, "in": true, "that": true, "have": true, "i": true,
		"it": true, "for": true, "not": true, "on": true, "with": true,
		"he": true, "as": true, "you": true, "do": true, "at": true,
		"this": true, "but": true, "his": true, "by": true, "from": true,
		"they": true, "we": true, "say": true, "her": true, "she": true,
		"or": true, "an": true, "will": true, "my": true, "one": true,
		"all": true, "would": true, "there": true, "their": true,
		"about": true, "which": true, "get": true, "has": true,
	}

	return &NLPAnalyzer{
		commonWords: commonWords,
	}
}

// AnalysisResult contains NLP analysis results
type AnalysisResult struct {
	WordCount        int                 `json:"word_count"`
	KeywordDensity   map[string]float64  `json:"keyword_density"`
	TopKeywords      []Keyword           `json:"top_keywords"`
	Sentences        []string            `json:"sentences"`
	ReadabilityScore int                 `json:"readability_score"`
	Issues           []string            `json:"issues"`
	Fixes            []string            `json:"fixes"`
}

// Keyword represents a keyword with count
type Keyword struct {
	Word  string  `json:"word"`
	Count int     `json:"count"`
	Density float64 `json:"density"`
}

// Analyze performs NLP analysis on content
func (n *NLPAnalyzer) Analyze(content string) *AnalysisResult {
	result := &AnalysisResult{
		KeywordDensity: make(map[string]float64),
		Issues:         []string{},
		Fixes:          []string{},
	}

	// Clean content
	content = strings.ToLower(content)
	
	// Count words
	words := strings.Fields(content)
	result.WordCount = len(words)

	// Extract sentences
	result.Sentences = n.extractSentences(content)
	
	// Count keywords (excluding common words)
	wordCount := make(map[string]int)
	totalSignificantWords := 0
	
	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:\"''()[]{}")
		
		if word == "" || n.commonWords[word] {
			continue
		}
		
		wordCount[word]++
		totalSignificantWords++
	}

	// Calculate density and get top keywords
	keywords := []Keyword{}
	for word, count := range wordCount {
		density := float64(count) / float64(totalSignificantWords) * 100
		result.KeywordDensity[word] = density
		
		keywords = append(keywords, Keyword{
			Word:    word,
			Count:   count,
			Density: density,
		})
	}

	// Sort and get top 10 keywords
	result.TopKeywords = n.getTopKeywords(keywords, 10)
	
	// Calculate readability score
	result.ReadabilityScore = n.calculateReadability(content, result.Sentences)
	
	// Identify SEO issues
	n.identifyIssues(result)
	
	return result
}

// extractSentences splits content into sentences
func (n *NLPAnalyzer) extractSentences(content string) []string {
	var sentences []string
	
	// Simple sentence splitting on .!?
	current := ""
	for _, char := range content {
		current += string(char)
		if char == '.' || char == '!' || char == '?' {
			if trimmed := strings.TrimSpace(current); trimmed != "" {
				sentences = append(sentences, trimmed)
				current = ""
			}
		}
	}
	
	// Add last sentence if any
	if trimmed := strings.TrimSpace(current); trimmed != "" {
		sentences = append(sentences, trimmed)
	}
	
	return sentences
}

// getTopKeywords returns top N keywords by count
func (n *NLPAnalyzer) getTopKeywords(keywords []Keyword, limit int) []Keyword {
	// Simple bubble sort (enough for small sets)
	for i := 0; i < len(keywords)-1; i++ {
		for j := i + 1; j < len(keywords); j++ {
			if keywords[j].Count > keywords[i].Count {
				keywords[i], keywords[j] = keywords[j], keywords[i]
			}
		}
	}
	
	if len(keywords) > limit {
		keywords = keywords[:limit]
	}
	
	return keywords
}

// calculateReadability gives a simple score
func (n *NLPAnalyzer) calculateReadability(content string, sentences []string) int {
	words := strings.Fields(content)
	
	if len(words) == 0 || len(sentences) == 0 {
		return 50 // Default score
	}
	
	avgWordsPerSentence := float64(len(words)) / float64(len(sentences))
	
	// Score based on sentence length (lower is better)
	score := 100
	if avgWordsPerSentence > 25 {
		score -= 20
	} else if avgWordsPerSentence > 20 {
		score -= 10
	} else if avgWordsPerSentence > 15 {
		score -= 5
	}
	
	// Penalize very long words
	longWords := 0
	for _, word := range words {
		if len(word) > 12 {
			longWords++
		}
	}
	
	if longWords > len(words)/10 {
		score -= 10
	}
	
	if score < 0 {
		score = 0
	}
	
	return score
}

// identifyIssues finds SEO content problems
func (n *NLPAnalyzer) identifyIssues(result *AnalysisResult) {
	// Check content length
	if result.WordCount < 300 {
		result.Issues = append(result.Issues, "Content too short (minimum 300 words recommended)")
		result.Fixes = append(result.Fixes, "Add more detailed content (aim for 500-1000 words)")
	} else if result.WordCount < 500 {
		result.Issues = append(result.Issues, "Content could be longer for better SEO")
		result.Fixes = append(result.Fixes, "Expand content to 500+ words for better ranking")
	}
	
	// Check keyword density
	hasGoodDensity := false
	for _, keyword := range result.TopKeywords {
		if keyword.Density > 0.5 && keyword.Density < 3.0 {
			hasGoodDensity = true
			break
		}
	}
	
	if !hasGoodDensity && len(result.TopKeywords) > 0 {
		mainKeyword := result.TopKeywords[0].Word
		result.Issues = append(result.Issues, "Poor keyword density")
		result.Fixes = append(result.Fixes, 
			"Include '"+mainKeyword+"' 2-3 times per 100 words for optimal density")
	}
	
	// Check sentence length
	longSentences := 0
	for _, sentence := range result.Sentences {
		words := strings.Fields(sentence)
		if len(words) > 30 {
			longSentences++
		}
	}
	
	if longSentences > len(result.Sentences)/3 {
		result.Issues = append(result.Issues, "Too many long sentences")
		result.Fixes = append(result.Fixes, "Break long sentences into shorter ones (max 25 words)")
	}
	
	// Check readability
	if result.ReadabilityScore < 60 {
		result.Issues = append(result.Issues, "Poor readability score")
		result.Fixes = append(result.Fixes, "Use simpler words and shorter sentences")
	}
}

// GenerateMetaDescription creates SEO-friendly description
func (n *NLPAnalyzer) GenerateMetaDescription(content string, keywords []string) string {
	// Get first 2 sentences
	sentences := n.extractSentences(content)
	if len(sentences) == 0 {
		return ""
	}
	
	// Build description from first sentences
	description := ""
	for i, sentence := range sentences {
		if i >= 2 {
			break
		}
		description += sentence + " "
	}
	
	// Truncate to ~155 chars
	if len(description) > 155 {
		description = description[:152] + "..."
	}
	
	// Ensure keywords are included
	needsKeyword := true
	for _, keyword := range keywords {
		if strings.Contains(strings.ToLower(description), strings.ToLower(keyword)) {
			needsKeyword = false
			break
		}
	}
	
	// Add keyword if missing and first keyword exists
	if needsKeyword && len(keywords) > 0 && len(description) < 140 {
		description = keywords[0] + ": " + strings.ToLower(description[:1]) + description[1:]
	}
	
	return strings.TrimSpace(description)
}

// ExtractKeywords returns main keywords from content
func (n *NLPAnalyzer) ExtractKeywords(content string, limit int) []string {
	result := n.Analyze(content)
	
	keywords := []string{}
	for i, kw := range result.TopKeywords {
		if i >= limit {
			break
		}
		keywords = append(keywords, kw.Word)
	}
	
	return keywords
}

// CheckKeywordUsage validates keyword placement
func (n *NLPAnalyzer) CheckKeywordUsage(content string, targetKeyword string) map[string]interface{} {
	result := map[string]interface{}{
		"keyword":      targetKeyword,
		"in_title":     false,
		"in_first_100": false,
		"in_h1":        false,
		"in_h2":        false,
		"count":        0,
		"density":      0.0,
		"issues":       []string{},
	}
	
	contentLower := strings.ToLower(content)
	targetLower := strings.ToLower(targetKeyword)
	
	// Count occurrences
	result["count"] = strings.Count(contentLower, targetLower)
	
	// Calculate density
	words := strings.Fields(contentLower)
	if len(words) > 0 {
		wordCount := make(map[string]int)
		for _, word := range words {
			wordCount[word]++
		}
		totalWords := len(words)
		if count, exists := wordCount[targetLower]; exists {
			result["density"] = float64(count) / float64(totalWords) * 100
		}
	}
	
	// Check first 100 words
	first100 := strings.Join(words[:min(100, len(words))], " ")
	if strings.Contains(first100, targetLower) {
		result["in_first_100"] = true
	}
	
	// Check issues
	count := result["count"].(int)
	if count < 2 {
		result["issues"] = append(result["issues"].([]string), 
			"Keyword used too few times (minimum 2-3 recommended)")
	}
	
	density := result["density"].(float64)
	if density > 3.0 {
		result["issues"] = append(result["issues"].([]string), 
			"Keyword density too high (over 3% - risk of keyword stuffing)")
	} else if density < 0.5 && count > 0 {
		result["issues"] = append(result["issues"].([]string), 
			"Keyword density too low (aim for 1-2%)")
	}
	
	return result
}

// SuggestImprovements provides actionable SEO content fixes
func (n *NLPAnalyzer) SuggestImprovements(content string) []string {
	result := n.Analyze(content)
	return result.Fixes
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}