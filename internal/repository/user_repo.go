package repository

import (
    "database/sql"
    "errors"
    "fmt"
    "time"

    "github.com/google/uuid"
    _ "github.com/mattn/go-sqlite3"
)

type User struct {
    ID          string    `json:"id"`
    Email       string    `json:"email"`
    Name        string    `json:"name"`
    APIKey      string    `json:"api_key"`
    Plan        string    `json:"plan"`
    Credits     int       `json:"credits"`
    CreatedAt   time.Time `json:"created_at"`
    LastLoginAt time.Time `json:"last_login_at"`
    IsActive    bool      `json:"is_active"`
    // Google OAuth Fields
    GoogleID     string    `json:"google_id,omitempty"`
    Avatar       string    `json:"avatar,omitempty"`
    Provider     string    `json:"provider"` // "local" or "google"
    IsVerified   bool      `json:"is_verified"`
    PasswordHash string    `json:"-"` // Don't expose in JSON
    // ✅ ADD THESE MISSING FIELDS
    LastLogin    time.Time `json:"last_login"`
    UpdatedAt    time.Time `json:"updated_at"`
}

type UserRepository struct {
    db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(dbPath string) (*UserRepository, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Create users table if not exists with Google OAuth columns
    query := `
    CREATE TABLE IF NOT EXISTS users (
        id TEXT PRIMARY KEY,
        email TEXT UNIQUE NOT NULL,
        name TEXT NOT NULL,
        api_key TEXT UNIQUE NOT NULL,
        plan TEXT DEFAULT 'free',
        credits INTEGER DEFAULT 100,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        last_login_at DATETIME,
        is_active BOOLEAN DEFAULT 1,
        google_id TEXT UNIQUE,
        avatar TEXT,
        provider TEXT DEFAULT 'local',
        is_verified BOOLEAN DEFAULT 0,
        password_hash TEXT,
        last_login DATETIME,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

    _, err = db.Exec(query)
    if err != nil {
        return nil, fmt.Errorf("failed to create table: %w", err)
    }

    // Add indexes for performance
    _, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_email ON users(email)")
    if err != nil {
        return nil, fmt.Errorf("failed to create email index: %w", err)
    }

    _, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_google_id ON users(google_id)")
    if err != nil {
        return nil, fmt.Errorf("failed to create google_id index: %w", err)
    }

    _, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_api_key ON users(api_key)")
    if err != nil {
        return nil, fmt.Errorf("failed to create api_key index: %w", err)
    }

    return &UserRepository{db: db}, nil
}

// =============================================
// CREATE
// =============================================

// Create inserts a new user
func (r *UserRepository) Create(user *User) error {
    // Generate UUID if not set
    if user.ID == "" {
        user.ID = uuid.New().String()
    }

    if user.Email == "" || user.Name == "" || user.APIKey == "" {
        return errors.New("email, name, and API key are required")
    }

    // Set defaults if not provided
    if user.Provider == "" {
        user.Provider = "local"
    }
    if user.Plan == "" {
        user.Plan = "free"
    }
    if user.Credits == 0 {
        user.Credits = 100
    }
    if !user.IsActive {
        user.IsActive = true
    }

    now := time.Now()
    user.CreatedAt = now
    user.UpdatedAt = now

    query := `
    INSERT INTO users (
        id, email, name, api_key, plan, credits, created_at, is_active,
        google_id, avatar, provider, is_verified, password_hash, last_login, updated_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

    _, err := r.db.Exec(
        query,
        user.ID,
        user.Email,
        user.Name,
        user.APIKey,
        user.Plan,
        user.Credits,
        user.CreatedAt,
        user.IsActive,
        user.GoogleID,
        user.Avatar,
        user.Provider,
        user.IsVerified,
        user.PasswordHash,
        user.LastLogin,
        user.UpdatedAt,
    )
    if err != nil {
        return fmt.Errorf("failed to create user: %w", err)
    }

    return nil
}

// =============================================
// READ (GET)
// =============================================

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id string) (*User, error) {
    user := &User{}
    query := `SELECT id, email, name, api_key, plan, credits, created_at, last_login_at, is_active,
              google_id, avatar, provider, is_verified, password_hash, last_login, updated_at
              FROM users WHERE id = ? AND is_active = 1`

    err := r.db.QueryRow(query, id).Scan(
        &user.ID, &user.Email, &user.Name, &user.APIKey,
        &user.Plan, &user.Credits, &user.CreatedAt, &user.LastLoginAt, &user.IsActive,
        &user.GoogleID, &user.Avatar, &user.Provider, &user.IsVerified, &user.PasswordHash,
        &user.LastLogin, &user.UpdatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, errors.New("user not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    return user, nil
}

// GetByIDWithInactive retrieves a user by ID including inactive users
func (r *UserRepository) GetByIDWithInactive(id string) (*User, error) {
    user := &User{}
    query := `SELECT id, email, name, api_key, plan, credits, created_at, last_login_at, is_active,
              google_id, avatar, provider, is_verified, password_hash, last_login, updated_at
              FROM users WHERE id = ?`

    err := r.db.QueryRow(query, id).Scan(
        &user.ID, &user.Email, &user.Name, &user.APIKey,
        &user.Plan, &user.Credits, &user.CreatedAt, &user.LastLoginAt, &user.IsActive,
        &user.GoogleID, &user.Avatar, &user.Provider, &user.IsVerified, &user.PasswordHash,
        &user.LastLogin, &user.UpdatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, errors.New("user not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(email string) (*User, error) {
    user := &User{}
    query := `SELECT id, email, name, api_key, plan, credits, created_at, last_login_at, is_active,
              google_id, avatar, provider, is_verified, password_hash, last_login, updated_at
              FROM users WHERE email = ?`

    err := r.db.QueryRow(query, email).Scan(
        &user.ID, &user.Email, &user.Name, &user.APIKey,
        &user.Plan, &user.Credits, &user.CreatedAt, &user.LastLoginAt, &user.IsActive,
        &user.GoogleID, &user.Avatar, &user.Provider, &user.IsVerified, &user.PasswordHash,
        &user.LastLogin, &user.UpdatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, errors.New("user not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    return user, nil
}

// GetByGoogleID retrieves a user by Google ID
func (r *UserRepository) GetByGoogleID(googleID string) (*User, error) {
    user := &User{}
    query := `SELECT id, email, name, api_key, plan, credits, created_at, last_login_at, is_active,
              google_id, avatar, provider, is_verified, password_hash, last_login, updated_at
              FROM users WHERE google_id = ?`

    err := r.db.QueryRow(query, googleID).Scan(
        &user.ID, &user.Email, &user.Name, &user.APIKey,
        &user.Plan, &user.Credits, &user.CreatedAt, &user.LastLoginAt, &user.IsActive,
        &user.GoogleID, &user.Avatar, &user.Provider, &user.IsVerified, &user.PasswordHash,
        &user.LastLogin, &user.UpdatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, nil // Return nil, nil if not found (not an error)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user by Google ID: %w", err)
    }

    return user, nil
}

// GetByAPIKey retrieves a user by API key
func (r *UserRepository) GetByAPIKey(apiKey string) (*User, error) {
    user := &User{}
    query := `SELECT id, email, name, api_key, plan, credits, created_at, last_login_at, is_active,
              google_id, avatar, provider, is_verified, password_hash, last_login, updated_at
              FROM users WHERE api_key = ? AND is_active = 1`

    err := r.db.QueryRow(query, apiKey).Scan(
        &user.ID, &user.Email, &user.Name, &user.APIKey,
        &user.Plan, &user.Credits, &user.CreatedAt, &user.LastLoginAt, &user.IsActive,
        &user.GoogleID, &user.Avatar, &user.Provider, &user.IsVerified, &user.PasswordHash,
        &user.LastLogin, &user.UpdatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, errors.New("user not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    return user, nil
}

// =============================================
// UPDATE
// =============================================

// Update updates user information
func (r *UserRepository) Update(user *User) error {
    user.UpdatedAt = time.Now()

    query := `
    UPDATE users 
    SET email = ?, name = ?, plan = ?, credits = ?, last_login_at = ?, is_active = ?,
        google_id = ?, avatar = ?, provider = ?, is_verified = ?, password_hash = ?,
        last_login = ?, updated_at = ?
    WHERE id = ?`

    result, err := r.db.Exec(
        query,
        user.Email,
        user.Name,
        user.Plan,
        user.Credits,
        user.LastLoginAt,
        user.IsActive,
        user.GoogleID,
        user.Avatar,
        user.Provider,
        user.IsVerified,
        user.PasswordHash,
        user.LastLogin,
        user.UpdatedAt,
        user.ID,
    )
    if err != nil {
        return fmt.Errorf("failed to update user: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    if rows == 0 {
        return errors.New("user not found")
    }

    return nil
}

// UpdateLastLogin updates user's last login time
func (r *UserRepository) UpdateLastLogin(userID string) error {
    now := time.Now()
    query := `UPDATE users SET last_login = ?, last_login_at = ?, updated_at = ? WHERE id = ? AND is_active = 1`
    result, err := r.db.Exec(query, now, now, now, userID)
    if err != nil {
        return fmt.Errorf("failed to update last login: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    if rows == 0 {
        return errors.New("user not found or inactive")
    }

    return nil
}

// UpdateAPIKey updates user's API key
func (r *UserRepository) UpdateAPIKey(userID, apiKey string) error {
    query := `UPDATE users SET api_key = ?, updated_at = ? WHERE id = ? AND is_active = 1`
    result, err := r.db.Exec(query, apiKey, time.Now(), userID)
    if err != nil {
        return fmt.Errorf("failed to update API key: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    if rows == 0 {
        return errors.New("user not found or inactive")
    }

    return nil
}

// DeductCredits deducts credits from user
func (r *UserRepository) DeductCredits(userID string, amount int) error {
    tx, err := r.db.Begin()
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Check current credits
    var credits int
    err = tx.QueryRow("SELECT credits FROM users WHERE id = ? AND is_active = 1", userID).Scan(&credits)
    if err != nil {
        if err == sql.ErrNoRows {
            return errors.New("user not found or inactive")
        }
        return fmt.Errorf("failed to get credits: %w", err)
    }

    if credits < amount {
        return errors.New("insufficient credits")
    }

    // Deduct credits
    _, err = tx.Exec("UPDATE users SET credits = credits - ?, updated_at = ? WHERE id = ?", amount, time.Now(), userID)
    if err != nil {
        return fmt.Errorf("failed to deduct credits: %w", err)
    }

    return tx.Commit()
}

// AddCredits adds credits to user
func (r *UserRepository) AddCredits(userID string, amount int) error {
    query := `UPDATE users SET credits = credits + ?, updated_at = ? WHERE id = ? AND is_active = 1`
    result, err := r.db.Exec(query, amount, time.Now(), userID)
    if err != nil {
        return fmt.Errorf("failed to add credits: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    if rows == 0 {
        return errors.New("user not found or inactive")
    }

    return nil
}

// =============================================
// LIST / COUNT / DELETE
// =============================================

// List returns all active users with pagination
func (r *UserRepository) List(limit, offset int) ([]*User, error) {
    if limit <= 0 {
        limit = 10
    }
    if offset < 0 {
        offset = 0
    }

    query := `SELECT id, email, name, api_key, plan, credits, created_at, last_login_at, is_active,
              google_id, avatar, provider, is_verified, password_hash, last_login, updated_at
              FROM users WHERE is_active = 1 ORDER BY created_at DESC LIMIT ? OFFSET ?`

    rows, err := r.db.Query(query, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to list users: %w", err)
    }
    defer rows.Close()

    var users []*User
    for rows.Next() {
        user := &User{}
        err := rows.Scan(
            &user.ID, &user.Email, &user.Name, &user.APIKey,
            &user.Plan, &user.Credits, &user.CreatedAt, &user.LastLoginAt, &user.IsActive,
            &user.GoogleID, &user.Avatar, &user.Provider, &user.IsVerified, &user.PasswordHash,
            &user.LastLogin, &user.UpdatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan user: %w", err)
        }
        users = append(users, user)
    }

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating rows: %w", err)
    }

    return users, nil
}

// Count returns the total number of active users
func (r *UserRepository) Count() (int, error) {
    var count int
    query := `SELECT COUNT(*) FROM users WHERE is_active = 1`
    err := r.db.QueryRow(query).Scan(&count)
    if err != nil {
        return 0, fmt.Errorf("failed to count users: %w", err)
    }
    return count, nil
}

// Delete soft deletes a user
func (r *UserRepository) Delete(id string) error {
    query := `UPDATE users SET is_active = 0, updated_at = ? WHERE id = ? AND is_active = 1`
    result, err := r.db.Exec(query, time.Now(), id)
    if err != nil {
        return fmt.Errorf("failed to delete user: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    if rows == 0 {
        return errors.New("user not found or already inactive")
    }

    return nil
}

// PermanentDelete permanently deletes a user
func (r *UserRepository) PermanentDelete(id string) error {
    query := `DELETE FROM users WHERE id = ?`
    result, err := r.db.Exec(query, id)
    if err != nil {
        return fmt.Errorf("failed to delete user: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    if rows == 0 {
        return errors.New("user not found")
    }

    return nil
}

// =============================================
// SEARCH / FILTER
// =============================================

// SearchUsers searches users by email or name
func (r *UserRepository) SearchUsers(query string, limit int) ([]*User, error) {
    if limit <= 0 {
        limit = 10
    }

    searchQuery := `SELECT id, email, name, api_key, plan, credits, created_at, last_login_at, is_active,
                    google_id, avatar, provider, is_verified, password_hash, last_login, updated_at
                    FROM users WHERE is_active = 1 AND (email LIKE ? OR name LIKE ?)
                    ORDER BY created_at DESC LIMIT ?`

    searchPattern := "%" + query + "%"
    rows, err := r.db.Query(searchQuery, searchPattern, searchPattern, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to search users: %w", err)
    }
    defer rows.Close()

    var users []*User
    for rows.Next() {
        user := &User{}
        err := rows.Scan(
            &user.ID, &user.Email, &user.Name, &user.APIKey,
            &user.Plan, &user.Credits, &user.CreatedAt, &user.LastLoginAt, &user.IsActive,
            &user.GoogleID, &user.Avatar, &user.Provider, &user.IsVerified, &user.PasswordHash,
            &user.LastLogin, &user.UpdatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan user: %w", err)
        }
        users = append(users, user)
    }

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating rows: %w", err)
    }

    return users, nil
}

// GetUsersByProvider returns users by provider type
func (r *UserRepository) GetUsersByProvider(provider string) ([]*User, error) {
    query := `SELECT id, email, name, api_key, plan, credits, created_at, last_login_at, is_active,
              google_id, avatar, provider, is_verified, password_hash, last_login, updated_at
              FROM users WHERE provider = ? AND is_active = 1 ORDER BY created_at DESC`

    rows, err := r.db.Query(query, provider)
    if err != nil {
        return nil, fmt.Errorf("failed to get users by provider: %w", err)
    }
    defer rows.Close()

    var users []*User
    for rows.Next() {
        user := &User{}
        err := rows.Scan(
            &user.ID, &user.Email, &user.Name, &user.APIKey,
            &user.Plan, &user.Credits, &user.CreatedAt, &user.LastLoginAt, &user.IsActive,
            &user.GoogleID, &user.Avatar, &user.Provider, &user.IsVerified, &user.PasswordHash,
            &user.LastLogin, &user.UpdatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan user: %w", err)
        }
        users = append(users, user)
    }

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating rows: %w", err)
    }

    return users, nil
}

// =============================================
// UTILITY
// =============================================

// Close closes the database connection
func (r *UserRepository) Close() error {
    if r.db != nil {
        return r.db.Close()
    }
    return nil
}

// Ping checks the database connection
func (r *UserRepository) Ping() error {
    if r.db == nil {
        return errors.New("database not initialized")
    }
    return r.db.Ping()
}