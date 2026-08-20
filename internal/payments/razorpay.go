// payments/razorpay.go
package payments

import (
    "bytes"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
    
    "github.com/google/uuid"
    "gorm.io/gorm"
)

// Razorpay API configurations
type RazorpayConfig struct {
    KeyID        string `yaml:"key_id"`
    KeySecret    string `yaml:"key_secret"`
    WebhookSecret string `yaml:"webhook_secret"`
    BaseURL      string `yaml:"base_url"`
}

// Payment represents a payment transaction
type Payment struct {
    gorm.Model
    UserID                 uuid.UUID `gorm:"type:uuid;not null"`
    SubscriptionID         *uuid.UUID `gorm:"type:uuid"`
    RazorpayPaymentID      string `gorm:"uniqueIndex;not null"`
    RazorpayOrderID        string `gorm:"index"`
    RazorpayInvoiceID      string `gorm:"index"`
    Amount                 int64  `gorm:"not null"` // Amount in paise (â‚¹100 = 10000)
    Currency               string `gorm:"default:'USD'"`
    Status                 string `gorm:"index"` // created, authorized, captured, refunded, failed
    Method                 string // card, netbanking, upi, wallet
    CardID                 string
    Bank                   string
    Wallet                 string
    VPA                    string // Virtual Payment Address for UPI
    Email                  string
    Contact                string
    Fee                    int64  // Razorpay fee deducted
    Tax                    int64  // GST/Tax amount
    Description            string
    Notes                  json.RawMessage `gorm:"type:jsonb"`
    ErrorCode              string
    ErrorDescription       string
    RefundStatus           string
    AmountRefunded         int64
    Captured               bool `gorm:"default:false"`
    International          bool `gorm:"default:false"`
}

// RazorpayOrder represents order creation request
type RazorpayOrder struct {
    Amount          int64             `json:"amount"`
    Currency        string           `json:"currency"`
    Receipt         string           `json:"receipt"`
    Notes           map[string]string `json:"notes,omitempty"`
    PartialPayment  bool             `json:"partial_payment,omitempty"`
    PaymentCapture  bool             `json:"payment_capture"`
    CustomerID      string           `json:"customer_id,omitempty"`
}

// RazorpayOrderResponse represents order creation response
type RazorpayOrderResponse struct {
    ID        string `json:"id"`
    Entity    string `json:"entity"`
    Amount    int64  `json:"amount"`
    AmountPaid int64 `json:"amount_paid"`
    AmountDue int64 `json:"amount_due"`
    Currency  string `json:"currency"`
    Receipt   string `json:"receipt"`
    Status    string `json:"status"`
    Attempts  int    `json:"attempts"`
    Notes     map[string]string `json:"notes"`
    CreatedAt int64  `json:"created_at"`
}

// RazorpayClient handles all Razorpay API interactions
type RazorpayClient struct {
    config     *RazorpayConfig
    httpClient *http.Client
    db         *gorm.DB
}

// NewRazorpayClient creates a new Razorpay client
func NewRazorpayClient(config *RazorpayConfig, db *gorm.DB) *RazorpayClient {
    return &RazorpayClient{
        config: config,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        db: db,
    }
}

// ================== CRITICAL MISSING PARTS ==================

// VerifyPaymentSignature verifies Razorpay payment signature (MOST IMPORTANT)
func (rc *RazorpayClient) VerifyPaymentSignature(orderID, paymentID, signature string) bool {
    payload := orderID + "|" + paymentID
    expectedSignature := ComputeHMACSHA256([]byte(payload), rc.config.KeySecret)
    return expectedSignature == signature
}

// VerifyWebhookSignature verifies Razorpay webhook signature
func (rc *RazorpayClient) VerifyWebhookSignature(body []byte, signature string) bool {
    expectedSignature := ComputeHMACSHA256(body, rc.config.WebhookSecret)
    return expectedSignature == signature
}

// ComputeHMACSHA256 computes HMAC SHA256
func ComputeHMACSHA256(data []byte, secret string) string {
    h := hmac.New(sha256.New, []byte(secret))
    h.Write(data)
    return hex.EncodeToString(h.Sum(nil))
}

// ================== SUBSCRIPTION METHODS ==================

// CreateSubscriptionOrder creates order for subscription plans
func (rc *RazorpayClient) CreateSubscriptionOrder(planID string, amount int64, userID uuid.UUID) (*RazorpayOrderResponse, error) {
    // Plan mapping - UPDATE WITH YOUR ACTUAL PRICES
   plans := map[string]struct {
    Name   string
    Amount int64
}{
    "Starter":   {Name: "Starter Plan", Amount: 408300},   // $49 = â‚¹4083
    "pro":       {Name: "Pro Plan", Amount: 1241700},      // $149 = â‚¹12,417
    "agency":    {Name: "Agency Plan", Amount: 2492500},   // $299 = â‚¹24,925
}
    
    plan, exists := plans[planID]
    if !exists {
        return nil, fmt.Errorf("invalid plan: %s", planID)
    }
    
    // Use provided amount or plan amount
    if amount == 0 {
        amount = plan.Amount
    }
    
    orderReq := &RazorpayOrder{
        Amount:         amount,
        Currency:       "USD",
        Receipt:        fmt.Sprintf("receipt_%s_%d", userID.String()[:8], time.Now().Unix()),
        PaymentCapture: true, // Auto-capture payments
        Notes: map[string]string{
            "user_id":    userID.String(),
            "plan":       planID,
            "plan_name":  plan.Name,
            "source":     "seosps",
        },
    }
    
    return rc.CreateOrder(orderReq)
}

// ================== FRONTEND HELPER METHODS ==================

// GetFrontendConfig returns config needed for frontend Razorpay checkout
func (rc *RazorpayClient) GetFrontendConfig() map[string]interface{} {
    return map[string]interface{}{
        "key_id":      rc.config.KeyID,
        "currency":    "USD",
        "name":        "SEOSPS",
        "description": "SEOSPS Subscription",
        "image":       "https://seosps.com/logo.png", // UPDATE THIS
        "prefill": map[string]interface{}{
            "name":    "Customer Name",
            "email":   "customer@example.com",
            "contact": "9999999999",
        },
        "theme": map[string]interface{}{
            "color": "#4F46E5",
        },
    }
}

// ================== PAYMENT STATUS METHODS ==================

// ProcessSuccessfulPayment handles successful payment
func (rc *RazorpayClient) ProcessSuccessfulPayment(paymentID, orderID string, amount int64, userID uuid.UUID) error {
    // 1. Get payment details from Razorpay
    paymentDetails, err := rc.FetchPayment(paymentID)
    if err != nil {
        return fmt.Errorf("failed to fetch payment details: %v", err)
    }
    
    // 2. Save payment to database
    payment := &Payment{
        UserID:            userID,
        RazorpayPaymentID: paymentID,
        RazorpayOrderID:   orderID,
        Amount:            amount,
        Currency:          "USD",
        Status:            "captured",
        Captured:          true,
        Email:             getString(paymentDetails, "email"),
        Contact:           getString(paymentDetails, "contact"),
        Method:            getString(paymentDetails, "method"),
        Description:       "Subscription Payment",
    }
    
    // Extract method-specific details
    if methodDetails, ok := paymentDetails[payment.Method].(map[string]interface{}); ok {
        payment.CardID = getString(methodDetails, "card_id")
        payment.Bank = getString(methodDetails, "bank")
        payment.Wallet = getString(methodDetails, "wallet")
        payment.VPA = getString(methodDetails, "vpa")
    }
    
    // Save to database
    if err := rc.SavePaymentToDB(payment); err != nil {
        return fmt.Errorf("failed to save payment: %v", err)
    }
    
    // 3. Update user subscription (implement this in your user service)
    // TODO: Call your subscription service to update user's plan
    
    // 4. Send confirmation email
    // TODO: Implement email sending
    
    return nil
}

// ================== UTILITY METHODS ==================

func getString(data map[string]interface{}, key string) string {
    if val, ok := data[key].(string); ok {
        return val
    }
    return ""
}

func getInt64(data map[string]interface{}, key string) int64 {
    if val, ok := data[key].(float64); ok {
        return int64(val)
    }
    return 0
}

// ================== EXISTING METHODS (keep these) ==================

// makeRequest makes authenticated requests to Razorpay API
func (rc *RazorpayClient) makeRequest(method, endpoint string, body interface{}) ([]byte, error) {
    var reqBody io.Reader
    if body != nil {
        jsonBody, err := json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal request body: %v", err)
        }
        reqBody = bytes.NewBuffer(jsonBody)
    }
    
    // Build URL
    url := fmt.Sprintf("%s%s", rc.config.BaseURL, endpoint)
    
    // Create request
    req, err := http.NewRequest(method, url, reqBody)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %v", err)
    }
    
    // Add authentication
    auth := base64.StdEncoding.EncodeToString([]byte(rc.config.KeyID + ":" + rc.config.KeySecret))
    req.Header.Set("Authorization", "Basic "+auth)
    req.Header.Set("Content-Type", "application/json")
    
    // Make request
    resp, err := rc.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("API request failed: %v", err)
    }
    defer resp.Body.Close()
    
    // Read response
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %v", err)
    }
    
    // Check for errors
    if resp.StatusCode >= 400 {
        var apiErr struct {
            Error struct {
                Code        string `json:"code"`
                Description string `json:"description"`
            } `json:"error"`
        }
        if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.Error.Code != "" {
            return nil, fmt.Errorf("razorpay error [%s]: %s", apiErr.Error.Code, apiErr.Error.Description)
        }
        return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
    }
    
    return respBody, nil
}

// CreateOrder creates a new order in Razorpay
// Add these methods to razorpay.go

// CreateOrderForPlan creates an order for a specific plan
func (rc *RazorpayClient) CreateOrderForPlan(planID string, amount int64, userID uuid.UUID, interval string) (map[string]interface{}, error) {
    // Map plan names to amounts (in paise/cents)
    plans := map[string]struct {
        Name   string
        Amount int64
    }{
        "Starter":      {Name: "Starter Plan", Amount: 4900},      // $49.00
        "professional": {Name: "Professional Plan", Amount: 14900}, // $149.00
        "enterprise":   {Name: "Enterprise Plan", Amount: 29900},   // $299.00
        "Starter":      {Name: "Starter Plan", Amount: 4900},
        "pro":          {Name: "Professional Plan", Amount: 14900},
        "agency":       {Name: "Enterprise Plan", Amount: 29900},
    }
    
    plan, exists := plans[planID]
    if !exists {
        return nil, fmt.Errorf("plan not found: %s", planID)
    }
    
    // Use provided amount or plan amount
    orderAmount := amount
    if orderAmount == 0 {
        orderAmount = plan.Amount
    }
    
    // Create order in Razorpay
    orderReq := &RazorpayOrder{
        Amount:         orderAmount,
        Currency:       "USD",
        Receipt:        fmt.Sprintf("receipt_%s_%d", userID.String()[:8], time.Now().Unix()),
        PaymentCapture: true,
        Notes: map[string]string{
            "user_id":   userID.String(),
            "plan":      planID,
            "plan_name": plan.Name,
            "interval":  interval,
            "source":    "seosps",
        },
    }
    
    orderResp, err := rc.CreateOrder(orderReq)
    if err != nil {
        return nil, err
    }
    
    // Return as map for compatibility with PaymentHandler
    return map[string]interface{}{
        "id":       orderResp.ID,
        "amount":   orderResp.Amount,
        "currency": orderResp.Currency,
        "status":   orderResp.Status,
    }, nil
}

// GetConfig returns Razorpay configuration for frontend
func (rc *RazorpayClient) GetConfig() map[string]interface{} {
    return map[string]interface{}{
        "key_id":   rc.config.KeyID,
        "currency": "USD",
        "name":     "SEOSPS",
    }
}

// VerifyPaymentSignature verifies payment signature
func (rc *RazorpayClient) VerifyPaymentSignature(orderID, paymentID, signature string) bool {
    payload := orderID + "|" + paymentID
    h := hmac.New(sha256.New, []byte(rc.config.KeySecret))
    h.Write([]byte(payload))
    expectedSignature := hex.EncodeToString(h.Sum(nil))
    return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// FetchPayment fetches payment details from Razorpay
func (rc *RazorpayClient) FetchPayment(paymentID string) (map[string]interface{}, error) {
    endpoint := fmt.Sprintf("/v1/payments/%s", paymentID)
    
    respBody, err := rc.makeRequest("GET", endpoint, nil)
    if err != nil {
        return nil, err
    }
    
    var paymentDetails map[string]interface{}
    if err := json.Unmarshal(respBody, &paymentDetails); err != nil {
        return nil, fmt.Errorf("failed to parse payment details: %v", err)
    }
    
    return paymentDetails, nil
}

// SavePaymentToDB saves payment details to database
func (rc *RazorpayClient) SavePaymentToDB(payment *Payment) error {
    result := rc.db.Create(payment)
    if result.Error != nil {
        return fmt.Errorf("failed to save payment to database: %v", result.Error)
    }
    return nil
}

// GetPaymentByID retrieves payment by Razorpay ID
func (rc *RazorpayClient) GetPaymentByID(razorpayPaymentID string) (*Payment, error) {
    var payment Payment
    result := rc.db.Where("razorpay_payment_id = ?", razorpayPaymentID).First(&payment)
    if result.Error != nil {
        return nil, fmt.Errorf("payment not found: %v", result.Error)
    }
    return &payment, nil
}

// GetUserPayments retrieves all payments for a user
func (rc *RazorpayClient) GetUserPayments(userID uuid.UUID) ([]Payment, error) {
    var payments []Payment
    result := rc.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&payments)
    if result.Error != nil {
        return nil, fmt.Errorf("failed to fetch user payments: %v", result.Error)
    }
    return payments, nil
}

func (rc *RazorpayClient) UpdatePaymentStatus(paymentID string, status string) error {
    // Implement payment status update logic
    // This might call Razorpay API or update database
    return nil // Placeholder
}


