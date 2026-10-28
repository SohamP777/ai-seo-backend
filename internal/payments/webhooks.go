// payments/webhooks.go
package payments

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
    
    "gorm.io/gorm"
)

// WebhookEvent represents a Razorpay webhook event
type WebhookEvent struct {
    gorm.Model
    RazorpayEventID string          `gorm:"uniqueIndex;not null"`
    EventType       string          `gorm:"index;not null"` // payment.captured, subscription.charged, etc.
    EntityType      string          `gorm:"not null"`       // payment, subscription, invoice
    EntityID        string          `gorm:"index;not null"`
    Payload         json.RawMessage `gorm:"type:jsonb;not null"`
    Status          string          `gorm:"index;default:'pending'"` // pending, processing, processed, failed
    Attempts        int             `gorm:"default:0"`
    LastAttemptAt   *time.Time
    ErrorMessage    string
    ProcessedAt     *time.Time
}

// WebhookHandler handles all Razorpay webhook events for SEOSPS
type WebhookHandler struct {
    config         *RazorpayConfig
    db             *gorm.DB
    razorpayClient *RazorpayClient
    eventHandlers  map[string]func(*WebhookEvent) error
}

// WebhookPayload represents the standard Razorpay webhook payload
type WebhookPayload struct {
    Entity   string                 `json:"entity"`
    Account  string                 `json:"account"`
    Event    string                 `json:"event"`
    Contains []string               `json:"contains"`
    Payload  map[string]interface{} `json:"payload"`
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(config *RazorpayConfig, db *gorm.DB, razorpayClient *RazorpayClient) *WebhookHandler {
    handler := &WebhookHandler{
        config:         config,
        db:             db,
        razorpayClient: razorpayClient,
        eventHandlers:  make(map[string]func(*WebhookEvent) error),
    }
    
    // Register all event handlers for SEOSPS
    handler.registerEventHandlers()
    
    return handler
}

// registerEventHandlers registers all webhook event handlers
func (wh *WebhookHandler) registerEventHandlers() {
    // Payment events
    wh.eventHandlers["payment.captured"] = wh.handlePaymentCaptured
    wh.eventHandlers["payment.failed"] = wh.handlePaymentFailed
    wh.eventHandlers["payment.refunded"] = wh.handlePaymentRefunded
    wh.eventHandlers["payment.refund.failed"] = wh.handleRefundFailed
    
    // Subscription events for SEOSPS plans
    wh.eventHandlers["subscription.authenticated"] = wh.handleSubscriptionAuthenticated
    wh.eventHandlers["subscription.activated"] = wh.handleSubscriptionActivated
    wh.eventHandlers["subscription.charged"] = wh.handleSubscriptionCharged
    wh.eventHandlers["subscription.completed"] = wh.handleSubscriptionCompleted
    wh.eventHandlers["subscription.pending"] = wh.handleSubscriptionPending
    wh.eventHandlers["subscription.halted"] = wh.handleSubscriptionHalted
    wh.eventHandlers["subscription.cancelled"] = wh.handleSubscriptionCancelled
    wh.eventHandlers["subscription.paused"] = wh.handleSubscriptionPaused
    wh.eventHandlers["subscription.resumed"] = wh.handleSubscriptionResumed
    
    // Invoice events
    wh.eventHandlers["invoice.paid"] = wh.handleInvoicePaid
    wh.eventHandlers["invoice.partially_paid"] = wh.handleInvoicePartiallyPaid
    wh.eventHandlers["invoice.expired"] = wh.handleInvoiceExpired
    
    // Order events
    wh.eventHandlers["order.paid"] = wh.handleOrderPaid
}

// VerifySignature verifies Razorpay webhook signature
func (wh *WebhookHandler) VerifySignature(signature string, body []byte) bool {
    // Create HMAC SHA256 hash
    h := hmac.New(sha256.New, []byte(wh.config.WebhookSecret))
    h.Write(body)
    expectedSignature := hex.EncodeToString(h.Sum(nil))
    
    // Compare signatures
    return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// HandleWebhook processes incoming webhook requests
func (wh *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
    // Read request body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read request body", http.StatusBadRequest)
        return
    }
    
    // Verify webhook signature
    signature := r.Header.Get("X-Razorpay-Signature")
    if !wh.VerifySignature(signature, body) {
        http.Error(w, "Invalid webhook signature", http.StatusUnauthorized)
        return
    }
    
    // Parse webhook payload
    var payload WebhookPayload
    if err := json.Unmarshal(body, &payload); err != nil {
        http.Error(w, "Invalid webhook payload", http.StatusBadRequest)
        return
    }
    
    // Save webhook event to database with retry mechanism
    webhookEvent, err := wh.saveWebhookEvent(payload, body)
    if err != nil {
        http.Error(w, "Failed to save webhook event", http.StatusInternalServerError)
        return
    }
    
    // Process webhook event asynchronously
    go wh.processWebhookEvent(webhookEvent)
    
    // Acknowledge receipt immediately
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Webhook received successfully"))
}

// saveWebhookEvent saves webhook event to database with idempotency check
func (wh *WebhookHandler) saveWebhookEvent(payload WebhookPayload, rawBody []byte) (*WebhookEvent, error) {
    // Extract entity ID from payload
    entityID := ""
    if entity, ok := payload.Payload[payload.Entity].(map[string]interface{}); ok {
        if id, ok := entity["id"].(string); ok {
            entityID = id
        }
    }
    
    // Check if event already processed (idempotency)
    var existingEvent WebhookEvent
    result := wh.db.Where("razorpay_event_id = ?", payload.Event+"_"+entityID).First(&existingEvent)
    if result.Error == nil {
        // Event already exists, return it
        return &existingEvent, nil
    }
    
    // Create new webhook event
    event := &WebhookEvent{
        RazorpayEventID: payload.Event + "_" + entityID,
        EventType:       payload.Event,
        EntityType:      payload.Entity,
        EntityID:        entityID,
        Payload:         rawBody,
        Status:          "pending",
        Attempts:        0,
    }
    
    // Save to database
    result = wh.db.Create(event)
    if result.Error != nil {
        return nil, fmt.Errorf("failed to save webhook event: %v", result.Error)
    }
    
    return event, nil
}

// processWebhookEvent processes webhook event with retry logic
func (wh *WebhookHandler) processWebhookEvent(event *WebhookEvent) {
    maxRetries := 3
    retryDelay := 2 * time.Second
    
    for attempt := 1; attempt <= maxRetries; attempt++ {
        err := wh.processEvent(event)
        
        if err == nil {
            // Success - update event status
            now := time.Now()
            event.Status = "processed"
            event.ProcessedAt = &now
            event.Attempts = attempt
            wh.db.Save(event)
            
            // Log successful processing
            fmt.Printf("Webhook event %s processed successfully (attempt %d)\n", 
                event.RazorpayEventID, attempt)
            return
        }
        
        // Failure - update attempt info
        now := time.Now()
        event.Status = "failed"
        event.LastAttemptAt = &now
        event.ErrorMessage = err.Error()
        event.Attempts = attempt
        wh.db.Save(event)
        
        // Log failure
        fmt.Printf("Failed to process webhook event %s (attempt %d): %v\n", 
            event.RazorpayEventID, attempt, err)
        
        // Exponential backoff for retries
        if attempt < maxRetries {
            time.Sleep(retryDelay * time.Duration(attempt))
        }
    }
    
    // All retries failed - mark as permanently failed
    event.Status = "permanently_failed"
    wh.db.Save(event)
    
    // TODO: Send alert to admin about failed webhook
    fmt.Printf("Webhook event %s permanently failed after %d attempts\n", 
        event.RazorpayEventID, maxRetries)
}

// processEvent routes event to appropriate handler
func (wh *WebhookHandler) processEvent(event *WebhookEvent) error {
    // Parse payload
    var payload WebhookPayload
    if err := json.Unmarshal(event.Payload, &payload); err != nil {
        return fmt.Errorf("failed to parse event payload: %v", err)
    }
    
    // Find handler for event type
    handler, exists := wh.eventHandlers[event.EventType]
    if !exists {
        return fmt.Errorf("no handler registered for event type: %s", event.EventType)
    }
    
    // Execute handler
    return handler(event)
}

// Event Handlers for SEOSPS

func (wh *WebhookHandler) handlePaymentCaptured(event *WebhookEvent) error {
    var payload WebhookPayload
    if err := json.Unmarshal(event.Payload, &payload); err != nil {
        return fmt.Errorf("failed to unmarshal payload: %v", err)
    }
    
    paymentData, ok := payload.Payload["payment"].(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid payment data in payload")
    }
    
    paymentID, ok := paymentData["id"].(string)
    if !ok {
        return fmt.Errorf("payment ID not found or invalid")
    }
    
    // Use amount if needed, otherwise ignore with underscore
    amount, _ := paymentData["amount"].(float64)
    _ = amount // Explicitly mark as unused if not needed
    
    // Update payment status in database - only passing paymentID and status
    if err := wh.razorpayClient.UpdatePaymentStatus(paymentID, "captured"); err != nil {
        return fmt.Errorf("failed to update payment status: %v", err)
    }
    
    // Store additional data in a separate table or log if needed
    fee := getInt64FromMap(paymentData, "fee")
    tax := getInt64FromMap(paymentData, "tax")
    
    // TODO: Store fee and tax information in a payment_details table
    fmt.Printf("Payment %s captured successfully. Fee: %d, Tax: %d\n", paymentID, fee, tax)
    
    // TODO: Activate user access to SEOSPS based on plan
    // TODO: Send payment confirmation email
    // TODO: Update analytics
    
    return nil
}

func (wh *WebhookHandler) handlePaymentFailed(event *WebhookEvent) error {
    var payload WebhookPayload
    if err := json.Unmarshal(event.Payload, &payload); err != nil {
        return fmt.Errorf("failed to unmarshal payload: %v", err)
    }
    
    paymentData, ok := payload.Payload["payment"].(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid payment data in payload")
    }
    
    paymentID, ok := paymentData["id"].(string)
    if !ok {
        return fmt.Errorf("payment ID not found or invalid")
    }
    
    errorCode, _ := paymentData["error_code"].(string)
    errorDesc, _ := paymentData["error_description"].(string)
    
    // Update payment status - only passing paymentID and status
    if err := wh.razorpayClient.UpdatePaymentStatus(paymentID, "failed"); err != nil {
        return fmt.Errorf("failed to update payment status: %v", err)
    }
    
    // Store error information separately or log it
    fmt.Printf("Payment %s failed: %s - %s\n", paymentID, errorCode, errorDesc)
    
    // TODO: Send payment failure notification to user
    // TODO: Trigger retry logic if applicable
    
    return nil
}

func (wh *WebhookHandler) handleSubscriptionActivated(event *WebhookEvent) error {
    var payload WebhookPayload
    if err := json.Unmarshal(event.Payload, &payload); err != nil {
        return fmt.Errorf("failed to unmarshal payload: %v", err)
    }
    
    subscriptionData, ok := payload.Payload["subscription"].(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid subscription data in payload")
    }
    
    subscriptionID, ok := subscriptionData["id"].(string)
    if !ok {
        return fmt.Errorf("subscription ID not found or invalid")
    }
    
    // Update subscription status in database
    var subscription Subscription
    result := wh.db.Where("razorpay_subscription_id = ?", subscriptionID).First(&subscription)
    if result.Error != nil {
        return fmt.Errorf("subscription not found: %v", result.Error)
    }
    
    subscription.Status = "active"
    if startDate, ok := subscriptionData["start_at"].(float64); ok {
        t := time.Unix(int64(startDate), 0)
        subscription.CurrentStartDate = &t
    }
    if endDate, ok := subscriptionData["end_at"].(float64); ok {
        t := time.Unix(int64(endDate), 0)
        subscription.CurrentEndDate = &t
    }
    
    result = wh.db.Save(&subscription)
    if result.Error != nil {
        return fmt.Errorf("failed to update subscription: %v", result.Error)
    }
    
    // TODO: Activate user's SEOSPS account features
    // TODO: Send welcome email with plan details
    
    fmt.Printf("SEOSPS subscription %s activated for user %s\n", 
        subscriptionID, subscription.UserID)
    return nil
}

func (wh *WebhookHandler) handleSubscriptionCharged(event *WebhookEvent) error {
    var payload WebhookPayload
    if err := json.Unmarshal(event.Payload, &payload); err != nil {
        return fmt.Errorf("failed to unmarshal payload: %v", err)
    }
    
    // This event contains both subscription and payment details
    paymentData, ok := payload.Payload["payment"].(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid payment data in payload")
    }
    
    subscriptionData, ok := payload.Payload["subscription"].(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid subscription data in payload")
    }
    
    paymentID, ok := paymentData["id"].(string)
    if !ok {
        return fmt.Errorf("payment ID not found or invalid")
    }
    
    subscriptionID, ok := subscriptionData["id"].(string)
    if !ok {
        return fmt.Errorf("subscription ID not found or invalid")
    }
    
    // Update payment status - only passing paymentID and status
    if err := wh.razorpayClient.UpdatePaymentStatus(paymentID, "captured"); err != nil {
        return fmt.Errorf("failed to update payment status: %v", err)
    }
    
    // Store additional payment details
    fee := getInt64FromMap(paymentData, "fee")
    tax := getInt64FromMap(paymentData, "tax")
    
    fmt.Printf("Subscription %s charged successfully. Payment: %s, Fee: %d, Tax: %d\n", 
        subscriptionID, paymentID, fee, tax)
    
    // TODO: Generate invoice for this charge
    // TODO: Update subscription renewal date
    // TODO: Send payment receipt email
    
    return nil
}

func (wh *WebhookHandler) handleSubscriptionCancelled(event *WebhookEvent) error {
    var payload WebhookPayload
    if err := json.Unmarshal(event.Payload, &payload); err != nil {
        return fmt.Errorf("failed to unmarshal payload: %v", err)
    }
    
    subscriptionData, ok := payload.Payload["subscription"].(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid subscription data in payload")
    }
    
    subscriptionID, ok := subscriptionData["id"].(string)
    if !ok {
        return fmt.Errorf("subscription ID not found or invalid")
    }
    
    // Update subscription status
    var subscription Subscription
    result := wh.db.Where("razorpay_subscription_id = ?", subscriptionID).First(&subscription)
    if result.Error != nil {
        return fmt.Errorf("subscription not found: %v", result.Error)
    }
    
    subscription.Status = "cancelled"
    if endedAt, ok := subscriptionData["ended_at"].(float64); ok {
        t := time.Unix(int64(endedAt), 0)
        subscription.EndedAt = &t
    }
    
    result = wh.db.Save(&subscription)
    if result.Error != nil {
        return fmt.Errorf("failed to update subscription: %v", result.Error)
    }
    
    // TODO: Downgrade user to free plan
    // TODO: Send cancellation confirmation email
    // TODO: Update churn analytics
    
    fmt.Printf("SEOSPS subscription %s cancelled\n", subscriptionID)
    return nil
}

// Helper function to safely get int64 from map
func getInt64FromMap(data map[string]interface{}, key string) int64 {
    if val, ok := data[key].(float64); ok {
        return int64(val)
    }
    return 0
}

// Additional handler stubs for other events
func (wh *WebhookHandler) handlePaymentRefunded(event *WebhookEvent) error {
    // TODO: Implement refund handling
    return nil
}

func (wh *WebhookHandler) handleRefundFailed(event *WebhookEvent) error {
    // TODO: Implement refund failure handling
    return nil
}

func (wh *WebhookHandler) handleSubscriptionAuthenticated(event *WebhookEvent) error {
    // TODO: Implement subscription authentication handling
    return nil
}

func (wh *WebhookHandler) handleSubscriptionCompleted(event *WebhookEvent) error {
    // TODO: Implement subscription completion handling
    return nil
}

func (wh *WebhookHandler) handleSubscriptionPending(event *WebhookEvent) error {
    // TODO: Implement subscription pending handling
    return nil
}

func (wh *WebhookHandler) handleSubscriptionHalted(event *WebhookEvent) error {
    // TODO: Implement subscription halted handling
    return nil
}

func (wh *WebhookHandler) handleSubscriptionPaused(event *WebhookEvent) error {
    // TODO: Implement subscription paused handling
    return nil
}

func (wh *WebhookHandler) handleSubscriptionResumed(event *WebhookEvent) error {
    // TODO: Implement subscription resumed handling
    return nil
}

func (wh *WebhookHandler) handleInvoicePaid(event *WebhookEvent) error {
    // TODO: Implement invoice paid handling
    return nil
}

func (wh *WebhookHandler) handleInvoicePartiallyPaid(event *WebhookEvent) error {
    // TODO: Implement partially paid invoice handling
    return nil
}

func (wh *WebhookHandler) handleInvoiceExpired(event *WebhookEvent) error {
    // TODO: Implement expired invoice handling
    return nil
}

func (wh *WebhookHandler) handleOrderPaid(event *WebhookEvent) error {
    // TODO: Implement order paid handling
    return nil
}

// GetWebhookEvents retrieves webhook events with filtering
func (wh *WebhookHandler) GetWebhookEvents(filter WebhookFilter) ([]WebhookEvent, error) {
    query := wh.db.Model(&WebhookEvent{})
    
    if filter.EventType != "" {
        query = query.Where("event_type = ?", filter.EventType)
    }
    if filter.Status != "" {
        query = query.Where("status = ?", filter.Status)
    }
    if filter.EntityID != "" {
        query = query.Where("entity_id = ?", filter.EntityID)
    }
    if !filter.StartDate.IsZero() {
        query = query.Where("created_at >= ?", filter.StartDate)
    }
    if !filter.EndDate.IsZero() {
        query = query.Where("created_at <= ?", filter.EndDate)
    }
    
    var events []WebhookEvent
    result := query.Order("created_at DESC").Limit(filter.Limit).Offset(filter.Offset).Find(&events)
    if result.Error != nil {
        return nil, fmt.Errorf("failed to fetch webhook events: %v", result.Error)
    }
    
    return events, nil
}

// RetryFailedEvents retries processing of failed webhook events
func (wh *WebhookHandler) RetryFailedEvents() error {
    var failedEvents []WebhookEvent
    result := wh.db.Where("status IN ?", []string{"failed", "pending"}).
        Where("attempts < 3").
        Where("created_at >= ?", time.Now().Add(-24*time.Hour)).
        Find(&failedEvents)
    
    if result.Error != nil {
        return fmt.Errorf("failed to fetch failed events: %v", result.Error)
    }
    
    for _, event := range failedEvents {
        go wh.processWebhookEvent(&event)
    }
    
    fmt.Printf("Retrying %d failed webhook events\n", len(failedEvents))
    return nil
}

// WebhookFilter for querying webhook events
type WebhookFilter struct {
    EventType string
    Status    string
    EntityID  string
    StartDate time.Time
    EndDate   time.Time
    Limit     int
    Offset    int
}