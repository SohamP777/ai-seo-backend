package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Truncate truncates a string to the given length
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Slugify converts a string to a URL-friendly slug
func Slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)
	
	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	
	// Remove all non-alphanumeric characters except hyphens
	reg := regexp.MustCompile("[^a-z0-9-]+")
	s = reg.ReplaceAllString(s, "")
	
	// Remove multiple hyphens
	reg = regexp.MustCompile("-+")
	s = reg.ReplaceAllString(s, "-")
	
	// Trim hyphens from ends
	s = strings.Trim(s, "-")
	
	return s
}

// GenerateRandomString generates a random string of given length
func GenerateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)[:length]
}

// IsEmail validates if a string is an email
func IsEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(email)
}

// IsURL validates if a string is a URL
func IsURL(url string) bool {
	pattern := `^(https?://)?([\da-z.-]+)\.([a-z.]{2,6})([/\w .-]*)*/?$`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(url)
}

// ExtractDomain extracts domain from URL
func ExtractDomain(url string) string {
	// Remove protocol
	url = strings.ReplaceAll(url, "http://", "")
	url = strings.ReplaceAll(url, "https://", "")
	url = strings.ReplaceAll(url, "www.", "")
	
	// Split by path
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return url
}

// ToTitle converts string to title case
func ToTitle(s string) string {
	return cases.Title(language.English).String(s)
}

// ToCamelCase converts string to camelCase
func ToCamelCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if i == 0 {
			words[i] = strings.ToLower(word)
		} else {
			words[i] = ToTitle(word)
		}
	}
	return strings.Join(words, "")
}

// ToSnakeCase converts string to snake_case
func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToKebabCase converts string to kebab-case
func ToKebabCase(s string) string {
	return strings.ReplaceAll(ToSnakeCase(s), "_", "-")
}

// StripHTML removes HTML tags from string
func StripHTML(html string) string {
	re := regexp.MustCompile("<[^>]*>")
	return re.ReplaceAllString(html, "")
}

// ExtractKeywords extracts keywords from text
func ExtractKeywords(text string, minLength int) []string {
	// Convert to lowercase
	text = strings.ToLower(text)
	
	// Remove punctuation
	reg := regexp.MustCompile("[^a-zA-Z0-9\\s]+")
	text = reg.ReplaceAllString(text, "")
	
	// Split into words
	words := strings.Fields(text)
	
	// Filter words by length and remove duplicates
	keywordMap := make(map[string]bool)
	keywords := make([]string, 0)
	
	for _, word := range words {
		if len(word) >= minLength && !isStopWord(word) {
			if !keywordMap[word] {
				keywordMap[word] = true
				keywords = append(keywords, word)
			}
		}
	}
	
	return keywords
}

// Common stop words to filter out
var stopWords = map[string]bool{
	"the": true, "and": true, "for": true, "you": true, "are": true,
	"but": true, "not": true, "have": true, "with": true, "this": true,
	"that": true, "from": true, "they": true, "will": true, "your": true,
	"can": true, "all": true, "has": true, "was": true, "were": true,
}

// MetaDescription generates a meta description from text
func GenerateMetaDescription(text string, maxLength int) string {
	// Strip HTML and normalize spaces
	text = StripHTML(text)
	text = strings.Join(strings.Fields(text), " ")
	
	// Truncate to max length
	if len(text) > maxLength {
		// Try to cut at last space
		lastSpace := strings.LastIndex(text[:maxLength], " ")
		if lastSpace > 0 {
			text = text[:lastSpace] + "..."
		} else {
			text = text[:maxLength-3] + "..."
		}
	}
	
	return text
}

// CountWords counts words in text
func CountWords(text string) int {
	text = StripHTML(text)
	words := strings.Fields(text)
	return len(words)
}

// CalculateReadingTime calculates reading time in minutes
func CalculateReadingTime(text string, wordsPerMinute int) int {
	wordCount := CountWords(text)
	if wordCount == 0 {
		return 0
	}
	
	minutes := wordCount / wordsPerMinute
	if minutes == 0 {
		minutes = 1
	}
	
	return minutes
}

// ExtractSentences extracts sentences from text
func ExtractSentences(text string) []string {
	// Simple sentence splitting based on punctuation
	re := regexp.MustCompile(`[.!?]+`)
	sentences := re.Split(text, -1)
	
	// Clean up sentences
	result := make([]string, 0)
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	
	return result
}

// NormalizeSpaces normalizes whitespace in text
func NormalizeSpaces(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// IsEmpty checks if a string is empty or only whitespace
func IsEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

// MaskString masks part of a string (useful for emails, credit cards, etc.)
func MaskString(s string, visibleChars int, maskChar rune) string {
	if len(s) <= visibleChars {
		return s
	}
	
	masked := strings.Repeat(string(maskChar), len(s)-visibleChars)
	return masked + s[len(s)-visibleChars:]
}

// MaskEmail masks an email address
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	
	name := parts[0]
	domain := parts[1]
	
	if len(name) <= 2 {
		return email
	}
	
	maskedName := name[:2] + strings.Repeat("*", len(name)-2)
	return maskedName + "@" + domain
}

// Pluralize returns plural form of a word
func Pluralize(word string) string {
	// Basic pluralization rules
	if strings.HasSuffix(word, "y") {
		return word[:len(word)-1] + "ies"
	}
	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "sh") || strings.HasSuffix(word, "ch") {
		return word + "es"
	}
	return word + "s"
}

// Singularize returns singular form of a word
func Singularize(word string) string {
	// Basic singularization rules
	if strings.HasSuffix(word, "ies") {
		return word[:len(word)-3] + "y"
	}
	if strings.HasSuffix(word, "ses") || strings.HasSuffix(word, "shes") || strings.HasSuffix(word, "ches") {
		return word[:len(word)-2]
	}
	if strings.HasSuffix(word, "s") {
		return word[:len(word)-1]
	}
	return word
}

// FormatBytes formats bytes to human readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ContainsAny checks if string contains any of the given substrings
func ContainsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// ReplaceMultiple replaces multiple substrings at once
func ReplaceMultiple(s string, replacements map[string]string) string {
	for old, new := range replacements {
		s = strings.ReplaceAll(s, old, new)
	}
	return s
}