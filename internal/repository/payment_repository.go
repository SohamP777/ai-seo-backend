package repository

import (
    "database/sql"
    "errors"
    "fmt"
    "time"
)

type Payment struct {
    ID            int       `json:"id"`
    UserID        int       `json:"user_id"`
    Amount        float64   `json:"amount"`
    Currency      string    `json:"currency"`
    Status        string    `json:"status"` // pending, completed, failed, refunded
    PaymentMethod string    `json:"payment_method"`
    TransactionID string    `json:"transaction_id"`
    Credits       int       `json:"credits"`
    CreatedAt     time.Time `json:"created_at"`
    CompletedAt   time.Time `json:"completed_at,omitempty"`
}

type Subscription struct {
    ID              int       `json:"id"`
    UserID          int       `json:"user_id"`
    Plan            string    `json:"plan"` // free, basic, pro, enterprise
    Status          string    `json:"status"` // active, cancelled, expired
    CreditsPerMonth int       `json:"credits_per_month"`
    Price           float64   `json:"price"`
    Currency        string    `json:"currency"`
    StartedAt       time.Time `json:"started_at"`
    ExpiresAt       time.Time `json:"expires_at"`
    CancelledAt     time.Time `json:"cancelled_at,omitempty"`
    AutoRenew       bool      `json:"auto_renew"`
}

type PaymentRepository struct {
    db *sql.DB
}

// NewPaymentRepository creates a new payment repository
func NewPaymentRepository(dbPath string) (*PaymentRepository, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Create tables
    queries := []string{
        `CREATE TABLE IF NOT EXISTS payments (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            amount REAL NOT NULL,
            currency TEXT DEFAULT 'USD',
            status TEXT DEFAULT 'pending',
            payment_method TEXT,
            transaction_id TEXT UNIQUE,
            credits INTEGER DEFAULT 0,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            completed_at DATETIME,
            FOREIGN KEY (user_id) REFERENCES users(id)
        );`,
        `CREATE TABLE IF NOT EXISTS subscriptions (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL UNIQUE,
            plan TEXT NOT NULL,
            status TEXT DEFAULT 'active',
            credits_per_month INTEGER DEFAULT 0,
            price REAL DEFAULT 0,
            currency TEXT DEFAULT 'USD',
            started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            expires_at DATETIME,
            cancelled_at DATETIME,
            auto_renew BOOLEAN DEFAULT 1,
            FOREIGN KEY (user_id) REFERENCES users(id)
        );`,
        `CREATE INDEX IF NOT EXISTS idx_payments_user ON payments(user_id);`,
        `CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);`,
        `CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);`,
    }

    for _, query := range queries {
        _, err = db.Exec(query)
        if err != nil {
            return nil, fmt.Errorf("failed to create table: %w", err)
        }
    }

    return &PaymentRepository{db: db}, nil
}

// CreatePayment creates a new payment record
func (r *PaymentRepository) CreatePayment(payment *Payment) error {
    if payment.UserID == 0 || payment.Amount <= 0 {
        return errors.New("user_id and valid amount are required")
    }

    query := `
    INSERT INTO payments (user_id, amount, currency, status, payment_method, transaction_id, credits, created_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

    result, err := r.db.Exec(query,
        payment.UserID, payment.Amount, payment.Currency, payment.Status,
        payment.PaymentMethod, payment.TransactionID, payment.Credits, time.Now(),
    )
    if err != nil {
        return fmt.Errorf("failed to create payment: %w", err)
    }

    id, err := result.LastInsertId()
    if err != nil {
        return fmt.Errorf("failed to get last insert ID: %w", err)
    }

    payment.ID = int(id)
    payment.CreatedAt = time.Now()
    return nil
}

// UpdatePaymentStatus updates payment status
func (r *PaymentRepository) UpdatePaymentStatus(transactionID string, status string) error {
    query := `UPDATE payments SET status = ?, completed_at = ? WHERE transaction_id = ?`
    
    var completedAt interface{} = nil
    if status == "completed" {
        completedAt = time.Now()
    }

    result, err := r.db.Exec(query, status, completedAt, transactionID)
    if err != nil {
        return fmt.Errorf("failed to update payment status: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    if rows == 0 {
        return errors.New("payment not found")
    }

    return nil
}

// GetPayment retrieves a payment by ID
func (r *PaymentRepository) GetPayment(id int) (*Payment, error) {
    payment := &Payment{}
    query := `SELECT id, user_id, amount, currency, status, payment_method, transaction_id, credits, created_at, completed_at 
              FROM payments WHERE id = ?`

    var completedAt sql.NullTime
    err := r.db.QueryRow(query, id).Scan(
        &payment.ID, &payment.UserID, &payment.Amount, &payment.Currency,
        &payment.Status, &payment.PaymentMethod, &payment.TransactionID,
        &payment.Credits, &payment.CreatedAt, &completedAt,
    )

    if err == sql.ErrNoRows {
        return nil, errors.New("payment not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get payment: %w", err)
    }

    if completedAt.Valid {
        payment.CompletedAt = completedAt.Time
    }

    return payment, nil
}

// GetUserPayments retrieves all payments for a user
func (r *PaymentRepository) GetUserPayments(userID int, limit, offset int) ([]*Payment, error) {
    query := `SELECT id, user_id, amount, currency, status, payment_method, transaction_id, credits, created_at, completed_at 
              FROM payments WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`

    rows, err := r.db.Query(query, userID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get payments: %w", err)
    }
    defer rows.Close()

    var payments []*Payment
    for rows.Next() {
        p := &Payment{}
        var completedAt sql.NullTime
        err := rows.Scan(
            &p.ID, &p.UserID, &p.Amount, &p.Currency, &p.Status,
            &p.PaymentMethod, &p.TransactionID, &p.Credits, &p.CreatedAt, &completedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan payment: %w", err)
        }
        if completedAt.Valid {
            p.CompletedAt = completedAt.Time
        }
        payments = append(payments, p)
    }

    return payments, nil
}

// CreateSubscription creates a new subscription
func (r *PaymentRepository) CreateSubscription(sub *Subscription) error {
    if sub.UserID == 0 || sub.Plan == "" {
        return errors.New("user_id and plan are required")
    }

    query := `
    INSERT INTO subscriptions (user_id, plan, status, credits_per_month, price, currency, started_at, expires_at, auto_renew)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

    result, err := r.db.Exec(query,
        sub.UserID, sub.Plan, sub.Status, sub.CreditsPerMonth,
        sub.Price, sub.Currency, sub.StartedAt, sub.ExpiresAt, sub.AutoRenew,
    )
    if err != nil {
        return fmt.Errorf("failed to create subscription: %w", err)
    }

    id, err := result.LastInsertId()
    if err != nil {
        return fmt.Errorf("failed to get last insert ID: %w", err)
    }

    sub.ID = int(id)
    return nil
}

// GetUserSubscription gets a user's subscription
func (r *PaymentRepository) GetUserSubscription(userID int) (*Subscription, error) {
    sub := &Subscription{}
    query := `SELECT id, user_id, plan, status, credits_per_month, price, currency, started_at, expires_at, cancelled_at, auto_renew 
              FROM subscriptions WHERE user_id = ?`

    var cancelledAt sql.NullTime
    err := r.db.QueryRow(query, userID).Scan(
        &sub.ID, &sub.UserID, &sub.Plan, &sub.Status, &sub.CreditsPerMonth,
        &sub.Price, &sub.Currency, &sub.StartedAt, &sub.ExpiresAt, &cancelledAt, &sub.AutoRenew,
    )

    if err == sql.ErrNoRows {
        return nil, errors.New("subscription not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get subscription: %w", err)
    }

    if cancelledAt.Valid {
        sub.CancelledAt = cancelledAt.Time
    }

    return sub, nil
}

// UpdateSubscription updates subscription
func (r *PaymentRepository) UpdateSubscription(sub *Subscription) error {
    query := `
    UPDATE subscriptions 
    SET plan = ?, status = ?, credits_per_month = ?, price = ?, currency = ?, 
        expires_at = ?, cancelled_at = ?, auto_renew = ?
    WHERE user_id = ?`

    result, err := r.db.Exec(query,
        sub.Plan, sub.Status, sub.CreditsPerMonth, sub.Price, sub.Currency,
        sub.ExpiresAt, sub.CancelledAt, sub.AutoRenew, sub.UserID,
    )
    if err != nil {
        return fmt.Errorf("failed to update subscription: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    if rows == 0 {
        return errors.New("subscription not found")
    }

    return nil
}

// CancelSubscription cancels a subscription
func (r *PaymentRepository) CancelSubscription(userID int) error {
    query := `UPDATE subscriptions SET status = 'cancelled', cancelled_at = ?, auto_renew = 0 WHERE user_id = ?`
    
    result, err := r.db.Exec(query, time.Now(), userID)
    if err != nil {
        return fmt.Errorf("failed to cancel subscription: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    if rows == 0 {
        return errors.New("subscription not found")
    }

    return nil
}

// GetActiveSubscriptions gets all active subscriptions expiring soon
func (r *PaymentRepository) GetActiveSubscriptions(daysUntilExpiry int) ([]*Subscription, error) {
    expiryDate := time.Now().AddDate(0, 0, daysUntilExpiry)
    
    query := `SELECT id, user_id, plan, status, credits_per_month, price, currency, started_at, expires_at, cancelled_at, auto_renew 
              FROM subscriptions 
              WHERE status = 'active' AND expires_at <= ? AND auto_renew = 1`

    rows, err := r.db.Query(query, expiryDate)
    if err != nil {
        return nil, fmt.Errorf("failed to get active subscriptions: %w", err)
    }
    defer rows.Close()

    var subscriptions []*Subscription
    for rows.Next() {
        sub := &Subscription{}
        var cancelledAt sql.NullTime
        err := rows.Scan(
            &sub.ID, &sub.UserID, &sub.Plan, &sub.Status, &sub.CreditsPerMonth,
            &sub.Price, &sub.Currency, &sub.StartedAt, &sub.ExpiresAt, &cancelledAt, &sub.AutoRenew,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan subscription: %w", err)
        }
        if cancelledAt.Valid {
            sub.CancelledAt = cancelledAt.Time
        }
        subscriptions = append(subscriptions, sub)
    }

    return subscriptions, nil
}

// GetRevenueStats gets revenue statistics
func (r *PaymentRepository) GetRevenueStats(startDate, endDate time.Time) (map[string]interface{}, error) {
    stats := make(map[string]interface{})

    // Total revenue
    var totalRevenue float64
    err := r.db.QueryRow(`
        SELECT COALESCE(SUM(amount), 0) 
        FROM payments 
        WHERE status = 'completed' AND created_at BETWEEN ? AND ?`,
        startDate, endDate).Scan(&totalRevenue)
    if err != nil {
        return nil, fmt.Errorf("failed to get total revenue: %w", err)
    }
    stats["total_revenue"] = totalRevenue

    // Number of transactions
    var transactionCount int
    err = r.db.QueryRow(`
        SELECT COUNT(*) 
        FROM payments 
        WHERE status = 'completed' AND created_at BETWEEN ? AND ?`,
        startDate, endDate).Scan(&transactionCount)
    if err != nil {
        return nil, fmt.Errorf("failed to get transaction count: %w", err)
    }
    stats["transaction_count"] = transactionCount

    // Average transaction value
    if transactionCount > 0 {
        stats["avg_transaction"] = totalRevenue / float64(transactionCount)
    } else {
        stats["avg_transaction"] = 0.0
    }

    // Revenue by payment method
    rows, err := r.db.Query(`
        SELECT payment_method, COALESCE(SUM(amount), 0) as total
        FROM payments 
        WHERE status = 'completed' AND created_at BETWEEN ? AND ?
        GROUP BY payment_method`,
        startDate, endDate)
    if err != nil {
        return nil, fmt.Errorf("failed to get revenue by method: %w", err)
    }
    defer rows.Close()

    methodStats := make(map[string]float64)
    for rows.Next() {
        var method string
        var total float64
        err := rows.Scan(&method, &total)
        if err != nil {
            continue
        }
        methodStats[method] = total
    }
    stats["by_method"] = methodStats

    return stats, nil
}

// Close closes the database connection
func (r *PaymentRepository) Close() error {
    return r.db.Close()
}