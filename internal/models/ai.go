
package models

import (
    "time"
)

// AIAnalysisResult - Main storage model for AI analysis
type AIAnalysisResult struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	URL         string    `json:"url" gorm:"index"`
	ScanID      string    `json:"scan_id" gorm:"uniqueIndex"`
	UserID      string    `json:"user_id" gorm:"index"`
	ProjectID   string    `json:"project_id" gorm:"index"`
	
	// Analysis Results - Store as JSON strings
	AEO         string    `json:"aeo" gorm:"type:json"`      // JSON string of AEOAnalysisResult
	GEO         string    `json:"geo" gorm:"type:json"`      // JSON string of GEOAnalysisResult
	AIO         string    `json:"aio" gorm:"type:json"`      // JSON string of AIOAnalysisResult
	
	// Combined Metrics
	OverallScore int      `json:"overall_score"`
	Priority     string   `json:"priority"` // high/medium/low
	
	// Metadata
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	AnalyzedAt  time.Time `json:"analyzed_at"`
	Status      string    `json:"status"` // pending/processing/completed/failed
	
	// Versioning
	Version     int       `json:"version"`
	PreviousID  string    `json:"previous_id,omitempty"`
}

// AEOAnalysis with enhanced fields
type AEOAnalysis struct {
	Score                int                    `json:"score" bson:"score"`
	FeaturedSnippetReady bool                   `json:"featured_snippet_ready" bson:"featured_snippet_ready"`
	FAQOptimized         bool                   `json:"faq_optimized" bson:"faq_optimized"`
	QAScore              int                    `json:"qa_score" bson:"qa_score"`
	StructuredData       []string               `json:"structured_data" bson:"structured_data"`
	AnswerClarity        int                    `json:"answer_clarity" bson:"answer_clarity"`
	MissingElements      []string               `json:"missing_elements" bson:"missing_elements"`
	Recommendations      []Recommendation       `json:"recommendations" bson:"recommendations"`
	QA_Pairs             []QuestionAnswer       `json:"qa_pairs" bson:"qa_pairs"`
	SchemaTypes          []string               `json:"schema_types" bson:"schema_types"`
	CompetitorScore      int                    `json:"competitor_score" bson:"competitor_score"`
}

// GEOAnalysis with enhanced fields
type GEOAnalysis struct {
	Score               int                    `json:"score" bson:"score"`
	EntityRich          bool                   `json:"entity_rich" bson:"entity_rich"`
	SemanticMarkup      bool                   `json:"semantic_markup" bson:"semantic_markup"`
	KnowledgeGraph      bool                   `json:"knowledge_graph" bson:"knowledge_graph"`
	SchemaOrg           []string               `json:"schema_org" bson:"schema_org"`
	EntityCount         int                    `json:"entity_count" bson:"entity_count"`
	ContextualDepth     int                    `json:"contextual_depth" bson:"contextual_depth"`
	MissingElements     []string               `json:"missing_elements" bson:"missing_elements"`
	Recommendations     []Recommendation       `json:"recommendations" bson:"recommendations"`
	Entities            []Entity               `json:"entities" bson:"entities"`
	InternalLinks       int                    `json:"internal_links" bson:"internal_links"`
	ExternalLinks       int                    `json:"external_links" bson:"external_links"`
	SemanticTags        []string               `json:"semantic_tags" bson:"semantic_tags"`
}

// AIOAnalysis with enhanced fields
type AIOAnalysis struct {
	Score                int                    `json:"score" bson:"score"`
	PromptOptimized      bool                   `json:"prompt_optimized" bson:"prompt_optimized"`
	ContentStructured    bool                   `json:"content_structured" bson:"content_structured"`
	LLMFriendly          bool                   `json:"llm_friendly" bson:"llm_friendly"`
	SemanticSections     int                    `json:"semantic_sections" bson:"semantic_sections"`
	Readability          int                    `json:"readability" bson:"readability"`
	MissingElements      []string               `json:"missing_elements" bson:"missing_elements"`
	Recommendations      []Recommendation       `json:"recommendations" bson:"recommendations"`
	WordCount            int                    `json:"word_count" bson:"word_count"`
	AvgSentenceLength    int                    `json:"avg_sentence_length" bson:"avg_sentence_length"`
	UniqueWords          int                    `json:"unique_words" bson:"unique_words"`
	ReadabilityLevel     string                 `json:"readability_level" bson:"readability_level"` // easy/medium/hard
	Topics               []string               `json:"topics" bson:"topics"`
}

// Recommendation with priority
type Recommendation struct {
	ID          string   `json:"id" bson:"id"`
	Title       string   `json:"title" bson:"title"`
	Description string   `json:"description" bson:"description"`
	Priority    string   `json:"priority" bson:"priority"` // critical/high/medium/low
	Category    string   `json:"category" bson:"category"`
	Impact      string   `json:"impact" bson:"impact"` // high/medium/low
	Effort      string   `json:"effort" bson:"effort"` // easy/medium/hard
	Steps       []string `json:"steps" bson:"steps"`
	CodeSnippet string   `json:"code_snippet,omitempty" bson:"code_snippet,omitempty"`
}

// QuestionAnswer with metadata
type QuestionAnswer struct {
	Question    string   `json:"question" bson:"question"`
	Answer      string   `json:"answer" bson:"answer"`
	Score       int      `json:"score" bson:"score"`
	Confidence  float64  `json:"confidence" bson:"confidence"`
	Position    int      `json:"position" bson:"position"`
	IsFeatured  bool     `json:"is_featured" bson:"is_featured"`
}

// Entity with enhanced fields
type Entity struct {
	Name        string    `json:"name" bson:"name"`
	Type        string    `json:"type" bson:"type"`
	Confidence  float64   `json:"confidence" bson:"confidence"`
	Relations   []string  `json:"relations" bson:"relations"`
	Frequency   int       `json:"frequency" bson:"frequency"`
	Sentiment   string    `json:"sentiment,omitempty" bson:"sentiment,omitempty"`
}

// BatchAnalysisRequest
type BatchAnalysisRequest struct {
	URLs        []string `json:"urls"`
	ProjectID   string   `json:"project_id"`
	WebhookURL  string   `json:"webhook_url,omitempty"`
	Priority    string   `json:"priority"`
}

// WebhookPayload
type WebhookPayload struct {
	Event       string                 `json:"event"` // analysis.completed, analysis.failed
	ScanID      string                 `json:"scan_id"`
	URL         string                 `json:"url"`
	Status      string                 `json:"status"`
	Data        interface{}            `json:"data,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}