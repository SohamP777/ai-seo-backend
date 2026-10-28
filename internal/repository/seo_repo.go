package repository

import (
    "database/sql"
    "errors"
    "fmt"
    "time"
)

type KeywordAnalysis struct {
    ID          int       `json:"id"`
    UserID      int       `json:"user_id"`
    URL         string    `json:"url"`
    Keyword     string    `json:"keyword"`
    Volume      int       `json:"volume"`
    Difficulty  float64   `json:"difficulty"`
    Density     float64   `json:"density"`
    Suggestions string    `json:"suggestions"` // JSON string
    LSITerms    string    `json:"lsi_terms"`   // JSON string
    CreatedAt   time.Time `json:"created_at"`
}

type CompetitorAnalysis struct {
    ID            int       `json:"id"`
    UserID        int       `json:"user_id"`
    YourURL       string    `json:"your_url"`
    CompetitorURL string    `json:"competitor_url"`
    MissingWords  string    `json:"missing_words"`  // JSON string
    OverlapWords  string    `json:"overlap_words"`  // JSON string
    Score         float64   `json:"score"`
    CreatedAt     time.Time `json:"created_at"`
}

type SEORepository struct {
    db *sql.DB
}

// NewSEORepository creates a new SEO repository
func NewSEORepository(dbPath string) (*SEORepository, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Create tables
    queries := []string{
        `CREATE TABLE IF NOT EXISTS keyword_analyses (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            url TEXT NOT NULL,
            keyword TEXT NOT NULL,
            volume INTEGER DEFAULT 0,
            difficulty REAL DEFAULT 0,
            density REAL DEFAULT 0,
            suggestions TEXT,
            lsi_terms TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(id)
        );`,
        `CREATE TABLE IF NOT EXISTS competitor_analyses (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            your_url TEXT NOT NULL,
            competitor_url TEXT NOT NULL,
            missing_words TEXT,
            overlap_words TEXT,
            score REAL DEFAULT 0,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (user_id) REFERENCES users(id)
        );`,
        `CREATE INDEX IF NOT EXISTS idx_keyword_user ON keyword_analyses(user_id);`,
        `CREATE INDEX IF NOT EXISTS idx_competitor_user ON competitor_analyses(user_id);`,
    }

    for _, query := range queries {
        _, err = db.Exec(query)
        if err != nil {
            return nil, fmt.Errorf("failed to create table: %w", err)
        }
    }

    return &SEORepository{db: db}, nil
}

// SaveKeywordAnalysis saves a keyword analysis result
func (r *SEORepository) SaveKeywordAnalysis(analysis *KeywordAnalysis) error {
    if analysis.UserID == 0 || analysis.URL == "" || analysis.Keyword == "" {
        return errors.New("user_id, url, and keyword are required")
    }

    query := `
    INSERT INTO keyword_analyses (user_id, url, keyword, volume, difficulty, density, suggestions, lsi_terms, created_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

    result, err := r.db.Exec(query,
        analysis.UserID, analysis.URL, analysis.Keyword,
        analysis.Volume, analysis.Difficulty, analysis.Density,
        analysis.Suggestions, analysis.LSITerms, time.Now(),
    )
    if err != nil {
        return fmt.Errorf("failed to save analysis: %w", err)
    }

    id, err := result.LastInsertId()
    if err != nil {
        return fmt.Errorf("failed to get last insert ID: %w", err)
    }

    analysis.ID = int(id)
    analysis.CreatedAt = time.Now()
    return nil
}

// GetKeywordAnalyses retrieves all analyses for a user
func (r *SEORepository) GetKeywordAnalyses(userID int, limit, offset int) ([]*KeywordAnalysis, error) {
    query := `SELECT id, user_id, url, keyword, volume, difficulty, density, suggestions, lsi_terms, created_at 
              FROM keyword_analyses WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`

    rows, err := r.db.Query(query, userID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get analyses: %w", err)
    }
    defer rows.Close()

    var analyses []*KeywordAnalysis
    for rows.Next() {
        a := &KeywordAnalysis{}
        err := rows.Scan(
            &a.ID, &a.UserID, &a.URL, &a.Keyword,
            &a.Volume, &a.Difficulty, &a.Density,
            &a.Suggestions, &a.LSITerms, &a.CreatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan analysis: %w", err)
        }
        analyses = append(analyses, a)
    }

    return analyses, nil
}

// SaveCompetitorAnalysis saves a competitor analysis result
func (r *SEORepository) SaveCompetitorAnalysis(analysis *CompetitorAnalysis) error {
    if analysis.UserID == 0 || analysis.YourURL == "" || analysis.CompetitorURL == "" {
        return errors.New("user_id and urls are required")
    }

    query := `
    INSERT INTO competitor_analyses (user_id, your_url, competitor_url, missing_words, overlap_words, score, created_at)
    VALUES (?, ?, ?, ?, ?, ?, ?)`

    result, err := r.db.Exec(query,
        analysis.UserID, analysis.YourURL, analysis.CompetitorURL,
        analysis.MissingWords, analysis.OverlapWords, analysis.Score, time.Now(),
    )
    if err != nil {
        return fmt.Errorf("failed to save competitor analysis: %w", err)
    }

    id, err := result.LastInsertId()
    if err != nil {
        return fmt.Errorf("failed to get last insert ID: %w", err)
    }

    analysis.ID = int(id)
    analysis.CreatedAt = time.Now()
    return nil
}

// GetCompetitorAnalyses retrieves competitor analyses for a user
func (r *SEORepository) GetCompetitorAnalyses(userID int, limit, offset int) ([]*CompetitorAnalysis, error) {
    query := `SELECT id, user_id, your_url, competitor_url, missing_words, overlap_words, score, created_at 
              FROM competitor_analyses WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`

    rows, err := r.db.Query(query, userID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get competitor analyses: %w", err)
    }
    defer rows.Close()

    var analyses []*CompetitorAnalysis
    for rows.Next() {
        a := &CompetitorAnalysis{}
        err := rows.Scan(
            &a.ID, &a.UserID, &a.YourURL, &a.CompetitorURL,
            &a.MissingWords, &a.OverlapWords, &a.Score, &a.CreatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan competitor analysis: %w", err)
        }
        analyses = append(analyses, a)
    }

    return analyses, nil
}

// GetKeywordStats gets keyword statistics for a user
func (r *SEORepository) GetKeywordStats(userID int) (map[string]interface{}, error) {
    stats := make(map[string]interface{})

    // Total analyses
    var total int
    err := r.db.QueryRow("SELECT COUNT(*) FROM keyword_analyses WHERE user_id = ?", userID).Scan(&total)
    if err != nil {
        return nil, fmt.Errorf("failed to get total: %w", err)
    }
    stats["total_analyses"] = total

    // Average difficulty
    var avgDifficulty sql.NullFloat64
    err = r.db.QueryRow("SELECT AVG(difficulty) FROM keyword_analyses WHERE user_id = ?", userID).Scan(&avgDifficulty)
    if err != nil {
        return nil, fmt.Errorf("failed to get avg difficulty: %w", err)
    }
    stats["avg_difficulty"] = avgDifficulty.Float64

    // Most analyzed keywords
    rows, err := r.db.Query(`
        SELECT keyword, COUNT(*) as count 
        FROM keyword_analyses 
        WHERE user_id = ? 
        GROUP BY keyword 
        ORDER BY count DESC 
        LIMIT 5`, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get top keywords: %w", err)
    }
    defer rows.Close()

    var topKeywords []map[string]interface{}
    for rows.Next() {
        var keyword string
        var count int
        err := rows.Scan(&keyword, &count)
        if err != nil {
            continue
        }
        topKeywords = append(topKeywords, map[string]interface{}{
            "keyword": keyword,
            "count":   count,
        })
    }
    stats["top_keywords"] = topKeywords

    return stats, nil
}

// DeleteOldAnalyses deletes analyses older than days
func (r *SEORepository) DeleteOldAnalyses(days int) (int64, error) {
    cutoff := time.Now().AddDate(0, 0, -days)
    
    result, err := r.db.Exec("DELETE FROM keyword_analyses WHERE created_at < ?", cutoff)
    if err != nil {
        return 0, fmt.Errorf("failed to delete old analyses: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return 0, fmt.Errorf("failed to get rows affected: %w", err)
    }

    return rows, nil
}

// Close closes the database connection
func (r *SEORepository) Close() error {
    return r.db.Close()
}