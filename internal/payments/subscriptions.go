// payments/subscriptions.go
package payments

import (
    "encoding/json"
    "fmt"
    "strconv"
    "time"
    
    "github.com/google/uuid"
    "ai-seo-backend/internal/models"
    "ai-seo-backend/internal/repository"
    "go.uber.org/zap"
    "gorm.io/gorm"
)

// Subscription represents a SEOSPS subscription
type Subscription struct {
    gorm.Model
    UserID                   uuid.UUID       `gorm:"type:uuid;not null;index"`
    PlanID                   uuid.UUID       `gorm:"type:uuid;not null;index"`
    RazorpaySubscriptionID   string          `gorm:"uniqueIndex;not null"`
    RazorpayCustomerID       string          `gorm:"index;not null"`
    Status                   string          `gorm:"index;not null"` // created, authenticated, active, pending, halted, cancelled, completed, trial
    CurrentStartDate         *time.Time
    CurrentEndDate           *time.Time
    StartedAt                *time.Time
    EndedAt                  *time.Time
    TrialStart               *time.Time
    TrialEnd                 *time.Time
    Quantity                 int             `gorm:"default:1"` // Number of websites/seats
    Metadata                 json.RawMessage `gorm:"type:jsonb"`
    Notes                    string
}

// SubscriptionCreateRequest for creating new SEOSPS subscriptions
type SubscriptionCreateRequest struct {
    UserID          uuid.UUID       `json:"user_id"`
    PlanID          uuid.UUID       `json:"plan_id"`
    CustomerDetails CustomerDetails `json:"customer_details"`
    Quantity        int             `json:"quantity"` // Number of websites
    StartDate       *time.Time      `json:"start_date,omitempty"`
    TrialEndAt      *time.Time      `json:"trial_end_at,omitempty"`
    Notes           map[string]string `json:"notes,omitempty"`
}

// CustomerDetails for Razorpay customer creation
type CustomerDetails struct {
    Name    string `json:"name"`
    Email   string `json:"email"`
    Contact string `json:"contact"`
    GSTN    string `json:"gstn,omitempty"`
}

// SubscriptionManager handles SEOSPS subscription operations
type SubscriptionManager struct {
    repo           *repository.PaymentRepository
    planManager    *PlanManager
    logger         *zap.Logger
    razorpayClient *RazorpayClient
    db             *gorm.DB
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(repo *repository.PaymentRepository, planManager *PlanManager, logger *zap.Logger, db *gorm.DB, razorpayConfig *RazorpayConfig) *SubscriptionManager {
    razorpayClient := NewRazorpayClient(razorpayConfig, db)
    
    return &SubscriptionManager{
        repo:           repo,
        planManager:    planManager,
        logger:         logger,
        razorpayClient: razorpayClient,
        db:             db,
    }
}

// CreateSubscription creates a new SEOSPS subscription
func (sm *SubscriptionManager) CreateSubscription(req *SubscriptionCreateRequest) (*Subscription, error) {
    // Begin database transaction
    tx := sm.db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            sm.logger.Error("panic recovered in CreateSubscription", zap.Any("recover", r))
        }
    }()

    // Get plan details
    plan, err := sm.planManager.GetPlanByID(req.PlanID)
    if err != nil {
        tx.Rollback()
        sm.logger.Error("failed to get plan", zap.Error(err), zap.String("plan_id", req.PlanID.String()))
        return nil, fmt.Errorf("failed to get plan: %v", err)
    }

    // Create Razorpay customer
    customerID, err := sm.createRazorpayCustomer(req.CustomerDetails)
    if err != nil {
        tx.Rollback()
        sm.logger.Error("failed to create customer", zap.Error(err), zap.String("email", req.CustomerDetails.Email))
        return nil, fmt.Errorf("failed to create customer: %v", err)
    }

    // Create Razorpay subscription
    razorpaySub, err := sm.createRazorpaySubscription(customerID, plan, req)
    if err != nil {
        tx.Rollback()
        sm.logger.Error("failed to create Razorpay subscription", 
            zap.Error(err), 
            zap.String("customer_id", customerID),
            zap.String("plan_id", strconv.FormatUint(uint64(plan.ID), 10))) // FIXED: Convert uint to string
        return nil, fmt.Errorf("failed to create Razorpay subscription: %v", err)
    }

    // Create local subscription record
    subscription := &Subscription{
        UserID:                 req.UserID,
        PlanID:                 req.PlanID,
        RazorpaySubscriptionID: razorpaySub["id"].(string),
        RazorpayCustomerID:     customerID,
        Status:                 "created",
        Quantity:               req.Quantity,
        Notes:                  "SEOSPS subscription created",
    }

    // Set trial dates if applicable
    if req.TrialEndAt != nil {
        now := time.Now()
        subscription.TrialStart = &now
        subscription.TrialEnd = req.TrialEndAt
        subscription.Status = "trial"
    }

    // Set start date
    if req.StartDate != nil {
        subscription.CurrentStartDate = req.StartDate
    } else {
        now := time.Now()
        subscription.CurrentStartDate = &now
    }

    // Save subscription to database
    if err := tx.Create(subscription).Error; err != nil {
        tx.Rollback()
        sm.logger.Error("failed to save subscription", 
            zap.Error(err), 
            zap.String("user_id", req.UserID.String()))
        return nil, fmt.Errorf("failed to save subscription: %v", err)
    }

    // Commit transaction
    if err := tx.Commit().Error; err != nil {
        sm.logger.Error("failed to commit transaction", zap.Error(err))
        return nil, fmt.Errorf("failed to commit transaction: %v", err)
    }

    sm.logger.Info("subscription created successfully", 
        zap.String("subscription_id", subscription.RazorpaySubscriptionID),
        zap.String("user_id", req.UserID.String()))

    return subscription, nil
}

// createRazorpayCustomer creates customer in Razorpay
func (sm *SubscriptionManager) createRazorpayCustomer(customer CustomerDetails) (string, error) {
    // Check if customer already exists
    existingCustomer, err := sm.findRazorpayCustomerByEmail(customer.Email)
    if err == nil && existingCustomer["id"] != nil {
        sm.logger.Info("existing customer found", zap.String("email", customer.Email))
        return existingCustomer["id"].(string), nil
    }

    // Create new customer
    customerData := map[string]interface{}{
        "name":    customer.Name,
        "email":   customer.Email,
        "contact": customer.Contact,
        "notes": map[string]string{
            "service": "SEOSPS",
            "type":    "seo_automation",
        },
    }

    if customer.GSTN != "" {
        customerData["gstin"] = customer.GSTN
    }

    resp, err := sm.razorpayClient.makeRequest("POST", "/v1/customers", customerData)
    if err != nil {
        return "", fmt.Errorf("Razorpay customer creation failed: %v", err)
    }

    var customerResp map[string]interface{}
    if err := json.Unmarshal(resp, &customerResp); err != nil {
        return "", fmt.Errorf("failed to parse customer response: %v", err)
    }

    sm.logger.Info("new customer created", zap.String("customer_id", customerResp["id"].(string)))
    return customerResp["id"].(string), nil
}

// findRazorpayCustomerByEmail finds existing customer by email
func (sm *SubscriptionManager) findRazorpayCustomerByEmail(email string) (map[string]interface{}, error) {
    endpoint := fmt.Sprintf("/v1/customers?email=%s", email)
    
    resp, err := sm.razorpayClient.makeRequest("GET", endpoint, nil)
    if err != nil {
        return nil, err
    }

    var response struct {
        Items []map[string]interface{} `json:"items"`
    }
    
    if err := json.Unmarshal(resp, &response); err != nil {
        return nil, fmt.Errorf("failed to parse customer search response: %v", err)
    }

    if len(response.Items) > 0 {
        return response.Items[0], nil
    }

    return nil, fmt.Errorf("customer not found")
}

// createRazorpaySubscription creates subscription in Razorpay
func (sm *SubscriptionManager) createRazorpaySubscription(customerID string, plan *PaymentPlan, req *SubscriptionCreateRequest) (map[string]interface{}, error) {
    // Build subscription request for SEOSPS
    subscriptionReq := map[string]interface{}{
        "plan_id":     plan.RazorpayPlanID,
        "customer_id": customerID,
        "total_count": 12, // 1 year subscription
        "quantity":    req.Quantity,
        "notes": map[string]string{
            "user_id":     req.UserID.String(),
            "plan_name":   plan.Name,
            "service":     "SEOSPS",
            "websites":    fmt.Sprintf("%d websites", req.Quantity),
            "plan_id":     strconv.FormatUint(uint64(plan.ID), 10), // FIXED: Convert uint to string
        },
        "notify_info": map[string]interface{}{
            "notify_email": []string{req.CustomerDetails.Email},
        },
    }

    // Add trial end if applicable
    if req.TrialEndAt != nil {
        subscriptionReq["start_at"] = req.TrialEndAt.Unix()
    }

    // Create subscription in Razorpay
    resp, err := sm.razorpayClient.makeRequest("POST", "/v1/subscriptions", subscriptionReq)
    if err != nil {
        return nil, fmt.Errorf("Razorpay subscription creation failed: %v", err)
    }

    var subscriptionResp map[string]interface{}
    if err := json.Unmarshal(resp, &subscriptionResp); err != nil {
        return nil, fmt.Errorf("failed to parse subscription response: %v", err)
    }

    return subscriptionResp, nil
}

// GetSubscriptionByID retrieves subscription by ID
func (sm *SubscriptionManager) GetSubscriptionByID(subscriptionID uuid.UUID) (*Subscription, error) {
    var subscription Subscription
    result := sm.db.Where("id = ?", subscriptionID).First(&subscription)
    if result.Error != nil {
        sm.logger.Error("subscription not found", 
            zap.Error(result.Error), 
            zap.String("subscription_id", subscriptionID.String()))
        return nil, fmt.Errorf("subscription not found: %v", result.Error)
    }
    return &subscription, nil
}

// GetSubscriptionByRazorpayID retrieves subscription by Razorpay ID
func (sm *SubscriptionManager) GetSubscriptionByRazorpayID(razorpayID string) (*Subscription, error) {
    var subscription Subscription
    result := sm.db.Where("razorpay_subscription_id = ?", razorpayID).First(&subscription)
    if result.Error != nil {
        sm.logger.Error("subscription not found by razorpay ID", 
            zap.Error(result.Error), 
            zap.String("razorpay_id", razorpayID))
        return nil, fmt.Errorf("subscription not found: %v", result.Error)
    }
    return &subscription, nil
}

// GetUserSubscription retrieves active subscription for user
func (sm *SubscriptionManager) GetUserSubscription(userID uuid.UUID) (*Subscription, error) {
    var subscription Subscription
    result := sm.db.Where("user_id = ? AND status IN ?", 
        userID, []string{"active", "trial", "pending"}).
        Order("created_at DESC").
        First(&subscription)
    
    if result.Error != nil {
        sm.logger.Error("no active subscription found", 
            zap.Error(result.Error), 
            zap.String("user_id", userID.String()))
        return nil, fmt.Errorf("no active subscription found: %v", result.Error)
    }
    return &subscription, nil
}

// UpdateSubscriptionStatus updates subscription status
func (sm *SubscriptionManager) UpdateSubscriptionStatus(subscriptionID uuid.UUID, status string, additionalData map[string]interface{}) error {
    subscription, err := sm.GetSubscriptionByID(subscriptionID)
    if err != nil {
        return err
    }

    // Validate status transition
    if !sm.isValidStatusTransition(subscription.Status, status) {
        sm.logger.Warn("invalid status transition", 
            zap.String("old_status", subscription.Status),
            zap.String("new_status", status))
        return fmt.Errorf("invalid status transition from %s to %s", subscription.Status, status)
    }

    // Update subscription
    subscription.Status = status
    
    // Update dates based on status
    switch status {
    case "active":
        now := time.Now()
        subscription.StartedAt = &now
        if subscription.CurrentStartDate == nil {
            subscription.CurrentStartDate = &now
        }
        // Set end date based on plan interval
        plan, err := sm.planManager.GetPlanByID(subscription.PlanID)
        if err == nil {
            // Calculate interval duration based on plan's interval
            intervalDuration := 30 * 24 * time.Hour // default monthly
            if plan.Interval == "yearly" {
                intervalDuration = 365 * 24 * time.Hour
            } else if plan.Interval == "quarterly" {
                intervalDuration = 90 * 24 * time.Hour
            }
            endDate := subscription.CurrentStartDate.Add(intervalDuration)
            subscription.CurrentEndDate = &endDate
        }
        
    case "cancelled":
        now := time.Now()
        subscription.EndedAt = &now
        
    case "trial":
        if subscription.TrialStart == nil {
            now := time.Now()
            subscription.TrialStart = &now
        }
    }

    // Update additional metadata if provided
    if additionalData != nil {
        if metadataBytes, err := json.Marshal(additionalData); err == nil {
            subscription.Metadata = metadataBytes
        }
    }

    result := sm.db.Save(subscription)
    if result.Error != nil {
        sm.logger.Error("failed to update subscription", 
            zap.Error(result.Error),
            zap.String("subscription_id", subscriptionID.String()))
        return fmt.Errorf("failed to update subscription: %v", result.Error)
    }

    sm.logger.Info("subscription status updated", 
        zap.String("subscription_id", subscriptionID.String()),
        zap.String("old_status", subscription.Status),
        zap.String("new_status", status))

    return nil
}

// isValidStatusTransition validates subscription status changes
func (sm *SubscriptionManager) isValidStatusTransition(oldStatus, newStatus string) bool {
    validTransitions := map[string][]string{
        "created":        {"authenticated", "active", "trial", "cancelled"},
        "authenticated":  {"active", "pending", "cancelled"},
        "active":         {"halted", "cancelled", "completed", "pending"},
        "trial":          {"active", "cancelled"},
        "pending":        {"active", "cancelled", "halted"},
        "halted":         {"active", "cancelled"},
        "cancelled":      {}, // Terminal state
        "completed":      {}, // Terminal state
        "pending_cancellation": {"cancelled"},
    }
    
    allowed, exists := validTransitions[oldStatus]
    if !exists {
        return false
    }
    
    for _, s := range allowed {
        if s == newStatus {
            return true
        }
    }
    
    return false
}

// CancelSubscription cancels a SEOSPS subscription
func (sm *SubscriptionManager) CancelSubscription(subscriptionID uuid.UUID, cancelAtPeriodEnd bool) error {
    subscription, err := sm.GetSubscriptionByID(subscriptionID)
    if err != nil {
        return err
    }

    // Cancel in Razorpay
    endpoint := fmt.Sprintf("/v1/subscriptions/%s/cancel", subscription.RazorpaySubscriptionID)
    
    reqBody := map[string]interface{}{
        "cancel_at_period_end": cancelAtPeriodEnd,
    }
    
    _, err = sm.razorpayClient.makeRequest("POST", endpoint, reqBody)
    if err != nil {
        sm.logger.Error("failed to cancel Razorpay subscription", 
            zap.Error(err),
            zap.String("razorpay_id", subscription.RazorpaySubscriptionID))
        return fmt.Errorf("failed to cancel Razorpay subscription: %v", err)
    }

    // Update local status
    status := "cancelled"
    if cancelAtPeriodEnd {
        status = "pending_cancellation"
    }
    
    return sm.UpdateSubscriptionStatus(subscriptionID, status, map[string]interface{}{
        "cancelled_at":     time.Now(),
        "cancel_at_period_end": cancelAtPeriodEnd,
    })
}

// UpdateSubscriptionQuantity updates number of websites/seats
func (sm *SubscriptionManager) UpdateSubscriptionQuantity(subscriptionID uuid.UUID, newQuantity int) error {
    subscription, err := sm.GetSubscriptionByID(subscriptionID)
    if err != nil {
        return err
    }

    // Update in Razorpay
    endpoint := fmt.Sprintf("/v1/subscriptions/%s", subscription.RazorpaySubscriptionID)
    
    reqBody := map[string]interface{}{
        "quantity": newQuantity,
        "notes": map[string]string{
            "updated_at":   time.Now().Format(time.RFC3339),
            "reason":       "Website count updated",
            "old_quantity": fmt.Sprintf("%d", subscription.Quantity),
            "new_quantity": fmt.Sprintf("%d", newQuantity),
        },
    }
    
    _, err = sm.razorpayClient.makeRequest("PATCH", endpoint, reqBody)
    if err != nil {
        sm.logger.Error("failed to update subscription quantity in Razorpay", 
            zap.Error(err),
            zap.String("subscription_id", subscriptionID.String()))
        return fmt.Errorf("failed to update subscription quantity: %v", err)
    }

    // Update locally
    subscription.Quantity = newQuantity
    result := sm.db.Save(subscription)
    if result.Error != nil {
        sm.logger.Error("failed to update local subscription quantity", 
            zap.Error(result.Error),
            zap.String("subscription_id", subscriptionID.String()))
        return fmt.Errorf("failed to update local subscription: %v", result.Error)
    }

    sm.logger.Info("subscription quantity updated", 
        zap.String("subscription_id", subscriptionID.String()),
        zap.Int("old_quantity", subscription.Quantity),
        zap.Int("new_quantity", newQuantity))

    return nil
}

// GetSubscriptionInvoices retrieves invoices for a subscription
func (sm *SubscriptionManager) GetSubscriptionInvoices(subscriptionID uuid.UUID) ([]models.Invoice, error) {
    _, err := sm.GetSubscriptionByID(subscriptionID)
    if err != nil {
        return nil, err
    }

    var invoices []models.Invoice
    result := sm.db.Where("subscription_id = ?", subscriptionID).
        Order("issue_date DESC").
        Find(&invoices)
    
    if result.Error != nil {
        sm.logger.Error("failed to fetch invoices", 
            zap.Error(result.Error),
            zap.String("subscription_id", subscriptionID.String()))
        return nil, fmt.Errorf("failed to fetch invoices: %v", result.Error)
    }

    return invoices, nil
}

// GetActiveSubscriptionsCount returns count of active subscriptions
func (sm *SubscriptionManager) GetActiveSubscriptionsCount() (int64, error) {
    var count int64
    result := sm.db.Model(&Subscription{}).
        Where("status IN ?", []string{"active", "trial"}).
        Count(&count)
    
    if result.Error != nil {
        sm.logger.Error("failed to count active subscriptions", zap.Error(result.Error))
        return 0, fmt.Errorf("failed to count active subscriptions: %v", result.Error)
    }
    
    return count, nil
}

// GetSubscriptionAnalytics returns subscription analytics for SEOSPS
func (sm *SubscriptionManager) GetSubscriptionAnalytics(startDate, endDate time.Time) (*SubscriptionAnalytics, error) {
    var analytics SubscriptionAnalytics
    
    // Get total subscriptions
    sm.db.Model(&Subscription{}).
        Where("created_at BETWEEN ? AND ?", startDate, endDate).
        Count(&analytics.TotalSubscriptions)
    
    // Get active subscriptions
    sm.db.Model(&Subscription{}).
        Where("status IN ? AND created_at BETWEEN ? AND ?", 
            []string{"active", "trial"}, startDate, endDate).
        Count(&analytics.ActiveSubscriptions)
    
    // Get trial subscriptions
    sm.db.Model(&Subscription{}).
        Where("status = 'trial' AND created_at BETWEEN ? AND ?", startDate, endDate).
        Count(&analytics.TrialSubscriptions)
    
    // Get cancelled subscriptions
    sm.db.Model(&Subscription{}).
        Where("status = 'cancelled' AND created_at BETWEEN ? AND ?", startDate, endDate).
        Count(&analytics.CancelledSubscriptions)
    
    // Calculate churn rate
    if analytics.TotalSubscriptions > 0 {
        analytics.ChurnRate = float64(analytics.CancelledSubscriptions) / float64(analytics.TotalSubscriptions) * 100
    }
    
    // Get subscriptions by plan
    var planCounts []struct {
        PlanID uuid.UUID
        Count  int64
    }
    
    sm.db.Model(&Subscription{}).
        Select("plan_id, COUNT(*) as count").
        Where("created_at BETWEEN ? AND ?", startDate, endDate).
        Group("plan_id").
        Scan(&planCounts)
    
    analytics.SubscriptionsByPlan = make(map[uuid.UUID]int64)
    for _, pc := range planCounts {
        analytics.SubscriptionsByPlan[pc.PlanID] = pc.Count
    }
    
    // Calculate MRR (Monthly Recurring Revenue)
    var mrr struct {
        Total float64
    }
    
    sm.db.Model(&Subscription{}).
        Joins("JOIN payment_plans ON subscriptions.plan_id = payment_plans.id").
        Where("subscriptions.status IN ? AND subscriptions.created_at BETWEEN ? AND ?", 
            []string{"active", "trial"}, startDate, endDate).
        Select("SUM(payment_plans.amount * subscriptions.quantity) as total").
        Scan(&mrr)
    
    analytics.MRR = mrr.Total / 100 // Convert from paise to currency
    
    sm.logger.Info("subscription analytics generated",
        zap.Time("start_date", startDate),
        zap.Time("end_date", endDate),
        zap.Float64("mrr", analytics.MRR))

    return &analytics, nil
}

// CheckSubscriptionAccess checks if user has access to SEOSPS features
func (sm *SubscriptionManager) CheckSubscriptionAccess(userID uuid.UUID, feature string) (bool, error) {
    subscription, err := sm.GetUserSubscription(userID)
    if err != nil {
        sm.logger.Warn("no subscription found for user", 
            zap.String("user_id", userID.String()))
        return false, nil // No subscription found
    }
    
    // Check if subscription is active
    if !sm.isSubscriptionActive(subscription) {
        sm.logger.Warn("subscription is not active", 
            zap.String("user_id", userID.String()),
            zap.String("status", subscription.Status))
        return false, fmt.Errorf("subscription is not active")
    }
    
    // Get plan details
    plan, err := sm.planManager.GetPlanByID(subscription.PlanID)
    if err != nil {
        sm.logger.Error("failed to get plan details", 
            zap.Error(err),
            zap.String("plan_id", subscription.PlanID.String()))
        return false, fmt.Errorf("failed to get plan details: %v", err)
    }
    
    // Check feature access based on plan
    switch feature {
    case "website_scan":
        return subscription.Quantity > 0, nil
    case "ai_automation":
        return plan.HasFeature("ai_automation"), nil
    case "priority_support":
        return plan.HasFeature("priority_support"), nil
    case "advanced_reports":
        return plan.HasFeature("advanced_reports"), nil
    default:
        return true, nil // Default access for basic features
    }
}

// isSubscriptionActive checks if subscription is currently active
func (sm *SubscriptionManager) isSubscriptionActive(subscription *Subscription) bool {
    now := time.Now()
    
    // Check status
    if subscription.Status != "active" && subscription.Status != "trial" && subscription.Status != "pending_cancellation" {
        return false
    }
    
    // Check trial period
    if subscription.Status == "trial" {
        if subscription.TrialEnd != nil && now.After(*subscription.TrialEnd) {
            return false
        }
    }
    
    // Check subscription period
    if subscription.CurrentEndDate != nil && now.After(*subscription.CurrentEndDate) {
        return false
    }
    
    return true
}

// RenewSubscription manually triggers subscription renewal
func (sm *SubscriptionManager) RenewSubscription(subscriptionID uuid.UUID) error {
    subscription, err := sm.GetSubscriptionByID(subscriptionID)
    if err != nil {
        return err
    }
    
    // Update end date
    plan, err := sm.planManager.GetPlanByID(subscription.PlanID)
    if err != nil {
        sm.logger.Error("failed to get plan for renewal", 
            zap.Error(err),
            zap.String("plan_id", subscription.PlanID.String()))
        return fmt.Errorf("failed to get plan: %v", err)
    }
    
    // Calculate interval duration based on plan's interval
    intervalDuration := 30 * 24 * time.Hour // default monthly
    if plan.Interval == "yearly" {
        intervalDuration = 365 * 24 * time.Hour
    } else if plan.Interval == "quarterly" {
        intervalDuration = 90 * 24 * time.Hour
    }
    
    newEndDate := time.Now().Add(intervalDuration)
    subscription.CurrentEndDate = &newEndDate
    
    result := sm.db.Save(subscription)
    if result.Error != nil {
        sm.logger.Error("failed to renew subscription", 
            zap.Error(result.Error),
            zap.String("subscription_id", subscriptionID.String()))
        return fmt.Errorf("failed to renew subscription: %v", result.Error)
    }
    
    sm.logger.Info("subscription renewed successfully", 
        zap.String("subscription_id", subscriptionID.String()),
        zap.Time("new_end_date", newEndDate))
    
    return nil
}

// SubscriptionAnalytics holds subscription analytics data
type SubscriptionAnalytics struct {
    TotalSubscriptions       int64
    ActiveSubscriptions      int64
    TrialSubscriptions       int64
    CancelledSubscriptions   int64
    ChurnRate                float64
    SubscriptionsByPlan      map[uuid.UUID]int64
    MRR                      float64
}

// ListSubscriptions retrieves subscriptions with filtering
func (sm *SubscriptionManager) ListSubscriptions(filter SubscriptionFilter) ([]Subscription, error) {
    query := sm.db.Model(&Subscription{})
    
    if filter.UserID != uuid.Nil {
        query = query.Where("user_id = ?", filter.UserID)
    }
    if filter.PlanID != uuid.Nil {
        query = query.Where("plan_id = ?", filter.PlanID)
    }
    if len(filter.Status) > 0 {
        query = query.Where("status IN ?", filter.Status)
    }
    if !filter.StartDate.IsZero() {
        query = query.Where("created_at >= ?", filter.StartDate)
    }
    if !filter.EndDate.IsZero() {
        query = query.Where("created_at <= ?", filter.EndDate)
    }
    
    var subscriptions []Subscription
    result := query.Order("created_at DESC").
        Limit(filter.Limit).
        Offset(filter.Offset).
        Find(&subscriptions)
    
    if result.Error != nil {
        sm.logger.Error("failed to fetch subscriptions", 
            zap.Error(result.Error),
            zap.Any("filter", filter))
        return nil, fmt.Errorf("failed to fetch subscriptions: %v", result.Error)
    }
    
    return subscriptions, nil
}

// SubscriptionFilter for querying subscriptions
type SubscriptionFilter struct {
    UserID    uuid.UUID
    PlanID    uuid.UUID
    Status    []string
    StartDate time.Time
    EndDate   time.Time
    Limit     int
    Offset    int
}