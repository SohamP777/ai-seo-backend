package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"ai-seo-backend/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
	
	// ✅ Import Razorpay SDK
	"github.com/razorpay/razorpay-go"
)

// ============ PAYMENT TYPES ============

type RazorpayProcessor struct {
	keyID     string
	keySecret string
	logger    *log.Logger
	client    *razorpay.Client // ✅ REAL Razorpay client
}

func NewRazorpayProcessor(keyID, keySecret string, logger *log.Logger) *RazorpayProcessor {
	// ✅ Initialize REAL Razorpay client
	client := razorpay.NewClient(keyID, keySecret)
	
	return &RazorpayProcessor{
		keyID:     keyID,
		keySecret: keySecret,
		logger:    logger,
		client:    client,
	}
}

// ✅ CreateOrder - Makes REAL API call to Razorpay
func (p *RazorpayProcessor) CreateOrder(amount int, currency string) (string, error) {
	p.logger.Printf("Creating REAL Razorpay order amount=%d currency=%s", amount, currency)
	
	// Create real order using Razorpay API
	data := map[string]interface{}{
		"amount":          amount,
		"currency":        currency,
		"receipt":         "receipt_" + fmt.Sprintf("%d", time.Now().UnixNano()),
		"payment_capture": 1,
	}
	
	// ✅ Make REAL API call to Razorpay
	order, err := p.client.Order.Create(data, nil)
	if err != nil {
		p.logger.Printf("❌ Razorpay order creation failed: %v", err)
		return "", fmt.Errorf("razorpay order creation failed: %w", err)
	}
	
	orderID, ok := order["id"].(string)
	if !ok || orderID == "" {
		return "", fmt.Errorf("razorpay returned empty order ID")
	}
	
	p.logger.Printf("✅ REAL Razorpay order created: %s", orderID)
	return orderID, nil
}

// ✅ VerifyPayment - Performs REAL signature verification (MANUAL)
func (p *RazorpayProcessor) VerifyPayment(orderID, paymentID, signature string) bool {
	p.logger.Printf("Verifying payment order_id=%s payment_id=%s", orderID, paymentID)
	
	// Create the string to verify: order_id|payment_id
	data := orderID + "|" + paymentID
	
	// Create HMAC SHA256
	h := hmac.New(sha256.New, []byte(p.keySecret))
	h.Write([]byte(data))
	expectedSignature := hex.EncodeToString(h.Sum(nil))
	
	// Compare signatures (constant time)
	result := hmac.Equal([]byte(signature), []byte(expectedSignature))
	
	if result {
		p.logger.Printf("✅ Signature verified successfully")
	} else {
		p.logger.Printf("❌ Signature verification failed")
		p.logger.Printf("   Expected: %s", expectedSignature)
		p.logger.Printf("   Received: %s", signature)
	}
	
	return result
}

// ============ PLAN MANAGER ============

type PlanManager struct {
	paymentRepo *PaymentRepository
	logger      *log.Logger
}

func NewPlanManager(paymentRepo *PaymentRepository, logger *log.Logger) *PlanManager {
	return &PlanManager{
		paymentRepo: paymentRepo,
		logger:      logger,
	}
}

func (m *PlanManager) GetPlan(planID string) (map[string]interface{}, error) {
	plans := map[string]map[string]interface{}{
		"starter": {
			"name":           "Starter",
			"price":          19,
			"yearly_price":   182,
			"currency":       "USD",
			"max_websites":   1,
			"scan_frequency": "weekly",
			"features":       []string{"SEO Analysis", "Basic Reports", "Monthly Updates", "Email Support"},
		},
		"professional": {
			"name":           "Professional",
			"price":          99,
			"yearly_price":   950,
			"currency":       "USD",
			"max_websites":   10,
			"scan_frequency": "daily",
			"features":       []string{"Advanced SEO Analysis", "Auto-Fix Issues", "Daily Monitoring", "Priority Support", "API Access"},
		},
		"enterprise": {
			"name":           "Enterprise",
			"price":          199,
			"yearly_price":   1910,
			"currency":       "USD",
			"max_websites":   25,
			"scan_frequency": "daily",
			"features":       []string{"Everything in Professional", "Custom Solutions", "24/7 Support", "SLA Agreement"},
		},
	}
	
	if plan, exists := plans[planID]; exists {
		return plan, nil
	}
	return nil, fmt.Errorf("plan not found")
}

func (m *PlanManager) ListPlans() []map[string]interface{} {
	return []map[string]interface{}{
		{"id": "starter", "name": "Starter", "price": 29, "yearly_price": 278, "currency": "USD", "popular": false},
		{"id": "professional", "name": "Professional", "price": 99, "yearly_price": 950, "currency": "USD", "popular": true},
		{"id": "enterprise", "name": "Enterprise", "price": 199, "yearly_price": 1910, "currency": "USD", "popular": false},
	}
}

// ============ SUBSCRIPTION MANAGER ============

type SubscriptionManager struct {
	paymentRepo *PaymentRepository
	planManager *PlanManager
	logger      *log.Logger
}

func NewSubscriptionManager(paymentRepo *PaymentRepository, planManager *PlanManager, logger *log.Logger) *SubscriptionManager {
	return &SubscriptionManager{
		paymentRepo: paymentRepo,
		planManager: planManager,
		logger:      logger,
	}
}

func (m *SubscriptionManager) CreateSubscription(userID, planID string) (string, error) {
	subscriptionID := "sub_" + uuid.New().String()
	m.logger.Printf("Created subscription user_id=%s plan_id=%s subscription_id=%s", userID, planID, subscriptionID)
	return subscriptionID, nil
}

func (m *SubscriptionManager) CancelSubscription(subscriptionID string) error {
	m.logger.Printf("Cancelled subscription subscription_id=%s", subscriptionID)
	return nil
}

// ============ PAYMENT REPOSITORY ============

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(payment *Payment) error {
	return r.db.Create(payment).Error
}

func (r *PaymentRepository) FindByUserID(userID string) ([]Payment, error) {
	var payments []Payment
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&payments).Error
	return payments, err
}

func (r *PaymentRepository) FindByOrderID(orderID string) (*Payment, error) {
	var payment Payment
	err := r.db.Where("order_id = ?", orderID).First(&payment).Error
	return &payment, err
}

// ============ PAYMENT MODEL ============

type Payment struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID      string    `gorm:"index" json:"user_id"`
	Amount      int       `json:"amount"`
	Currency    string    `gorm:"default:USD" json:"currency"`
	Status      string    `json:"status"`
	PaymentID   string    `gorm:"uniqueIndex" json:"payment_id"`
	OrderID     string    `gorm:"uniqueIndex" json:"order_id"`
	PlanID      string    `json:"plan_id"`
	Interval    string    `json:"interval"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ============ PAYMENT HANDLER ============

type PaymentHandler struct {
	logger           *log.Logger
	paymentProcessor *RazorpayProcessor
	subscriptionMgr  *SubscriptionManager
	planManager      *PlanManager
	paymentRepo      *PaymentRepository
	db               *gorm.DB
}

func NewPaymentHandler(logger *log.Logger, processor *RazorpayProcessor, subMgr *SubscriptionManager, planMgr *PlanManager, paymentRepo *PaymentRepository) *PaymentHandler {
	return &PaymentHandler{
		logger:           logger,
		paymentProcessor: processor,
		subscriptionMgr:  subMgr,
		planManager:      planMgr,
		paymentRepo:      paymentRepo,
	}
}

// SetDB sets the database connection
func (h *PaymentHandler) SetDB(db *gorm.DB) {
	h.db = db
}

// GetConfig returns Razorpay configuration for frontend
func (h *PaymentHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("GetConfig called")
	
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"key_id":  h.paymentProcessor.keyID,
		"key":     h.paymentProcessor.keyID,
		"success": true,
	})
}

// ListPlans returns all available plans
func (h *PaymentHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("ListPlans called")
	
	plans := h.planManager.ListPlans()
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"plans":   plans,
		"success": true,
	})
}

// CreateOrder creates a Razorpay order
func (h *PaymentHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Plan     string `json:"plan"`
		Interval string `json:"interval"`
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Printf("CreateOrder: Invalid request: %v", err)
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}
	
	h.logger.Printf("CreateOrder request: plan=%s, interval=%s, amount=%d, currency=%s", 
		req.Plan, req.Interval, req.Amount, req.Currency)
	
	// Normalize plan name
	planID := req.Plan
	switch req.Plan {
	case "starter", "essential", "basic":
		planID = "starter"
	case "professional", "pro":
		planID = "professional"
	case "enterprise", "agency":
		planID = "enterprise"
	default:
		planID = "starter"
	}
	
	// Get plan details
	plan, err := h.planManager.GetPlan(planID)
	if err != nil {
		h.logger.Printf("CreateOrder: Plan not found: %s", req.Plan)
		utils.ErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Plan '%s' not found", req.Plan))
		return
	}
	
	// Get price based on interval
	var price int
	
	if req.Interval == "yearly" {
		if yearlyPrice, ok := plan["yearly_price"].(int); ok {
			price = yearlyPrice
		} else {
			price = plan["price"].(int) * 12 * 80 / 100
		}
	} else {
		price = plan["price"].(int)
	}
	
	// ✅ Convert to smallest currency unit (cents for USD, paise for INR)
	amount := int64(price * 100)
	if req.Amount > 0 {
		amount = req.Amount
	}
	
	currency := "USD"
	if req.Currency != "" {
		currency = req.Currency
	}
	
	// Get user ID from context
	userID := r.Context().Value("user_id")
	if userID == nil {
		h.logger.Printf("CreateOrder: User not authenticated")
		utils.ErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}
	
	userIDStr := fmt.Sprintf("%v", userID)
	h.logger.Printf("Creating order for user: %s, amount: %d", userIDStr, amount)
	
	// ✅ Create REAL Razorpay order
	orderID, err := h.paymentProcessor.CreateOrder(int(amount), currency)
	if err != nil {
		h.logger.Printf("CreateOrder: Failed to create order: %v", err)
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to create payment order")
		return
	}
	
	// Save payment record
	payment := &Payment{
		ID:        uuid.New().String(),
		UserID:    userIDStr,
		Amount:    int(amount),
		Currency:  currency,
		Status:    "pending",
		OrderID:   orderID,
		PlanID:    planID,
		Interval:  req.Interval,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	if h.paymentRepo != nil {
		h.paymentRepo.Create(payment)
	}
	
	// ✅ Return REAL order details to frontend
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"order_id":   orderID,
		"amount":     amount,
		"currency":   currency,
		"plan":       req.Plan,
		"plan_name":  plan["name"],
		"key_id":     h.paymentProcessor.keyID,
		"interval":   req.Interval,
		"success":    true,
	})
}

// VerifyPayment verifies Razorpay payment and updates user subscription
func (h *PaymentHandler) VerifyPayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID   string `json:"order_id"`
		PaymentID string `json:"payment_id"`
		Signature string `json:"signature"`
		Plan      string `json:"plan"`
		Interval  string `json:"interval"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Printf("VerifyPayment: Invalid request: %v", err)
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}
	
	h.logger.Printf("VerifyPayment: order_id=%s, payment_id=%s, plan=%s, interval=%s", 
		req.OrderID, req.PaymentID, req.Plan, req.Interval)
	
	// Get user ID from context
	userID := r.Context().Value("user_id")
	if userID == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}
	
	userIDStr := fmt.Sprintf("%v", userID)
	
	// ✅ Verify payment signature
	verified := h.paymentProcessor.VerifyPayment(req.OrderID, req.PaymentID, req.Signature)
	
	if !verified {
		utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"verified": false,
			"success":  false,
			"message":  "Payment verification failed - invalid signature",
		})
		return
	}
	
	// Get plan details for subscription update
	planID := req.Plan
	switch req.Plan {
	case "starter", "essential":
		planID = "starter"
	case "professional", "pro":
		planID = "professional"
	case "enterprise", "agency":
		planID = "enterprise"
	default:
		planID = "starter"
	}
	
	plan, err := h.planManager.GetPlan(planID)
	if err != nil {
		h.logger.Printf("VerifyPayment: Plan not found: %s", req.Plan)
		utils.ErrorResponse(w, http.StatusNotFound, "Plan not found")
		return
	}
	
	// Calculate subscription end date
	subscriptionMonths := 1
	if req.Interval == "yearly" {
		subscriptionMonths = 12
	}
	
	subscriptionEndDate := time.Now().AddDate(0, subscriptionMonths, 0)
	
	// Get max websites from plan
	maxWebsites := 1
	if val, ok := plan["max_websites"].(int); ok {
		maxWebsites = val
	}
	
	// Get scan frequency from plan
	scanFrequency := "weekly"
	if val, ok := plan["scan_frequency"].(string); ok {
		scanFrequency = val
	}
	
	// Get amount
	var amount float64
	if req.Interval == "yearly" {
		if val, ok := plan["yearly_price"].(int); ok {
			amount = float64(val)
		}
	} else {
		if val, ok := plan["price"].(int); ok {
			amount = float64(val)
		}
	}
	
	// ✅ Update user subscription in database
	if h.db != nil {
		updates := map[string]interface{}{
			"plan":                   planID,
			"subscription_end_date":  subscriptionEndDate,
			"max_websites":           maxWebsites,
			"scan_frequency":         scanFrequency,
			"status":                 "active",
			"last_payment_amount":    amount,
			"updated_at":             time.Now(),
		}
		
		result := h.db.Table("users").Where("id = ?", userIDStr).Updates(updates)
		if result.Error != nil {
			h.logger.Printf("VerifyPayment: Failed to update user subscription: %v", result.Error)
		} else {
			h.logger.Printf("✅ VerifyPayment: Updated user subscription for %s", userIDStr)
		}
	}
	
	// ✅ Update payment status in database
	if h.paymentRepo != nil {
		payment, err := h.paymentRepo.FindByOrderID(req.OrderID)
		if err == nil && payment != nil {
			payment.Status = "completed"
			payment.PaymentID = req.PaymentID
			payment.UpdatedAt = time.Now()
			h.paymentRepo.db.Save(payment)
		}
	}
	
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"verified":             true,
		"success":              true,
		"message":              "✅ Payment verified successfully! Your subscription is now active.",
		"plan":                 plan["name"],
		"max_websites":         maxWebsites,
		"scan_frequency":       scanFrequency,
		"subscription_end":     subscriptionEndDate.Format("2006-01-02"),
		"subscription_status":  "active",
	})
}

// CheckSubscriptionStatus checks user's subscription status
func (h *PaymentHandler) CheckSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}
	
	userIDStr := fmt.Sprintf("%v", userID)
	h.logger.Printf("CheckSubscriptionStatus for user: %s", userIDStr)
	
	// Default values
	status := "inactive"
	plan := "starter"
	maxWebsites := 1
	scanFrequency := "weekly"
	subscriptionEndDate := time.Now()
	isActive := false
	
	// Query database for subscription status
	if h.db != nil {
		var user struct {
			Status              string
			Plan                string
			MaxWebsites         int
			ScanFrequency       string
			SubscriptionEndDate time.Time
		}
		
		err := h.db.Table("users").
			Where("id = ?", userIDStr).
			Select("status, plan, max_websites, scan_frequency, subscription_end_date").
			First(&user).Error
		
		if err == nil {
			status = user.Status
			plan = user.Plan
			maxWebsites = user.MaxWebsites
			scanFrequency = user.ScanFrequency
			subscriptionEndDate = user.SubscriptionEndDate
			isActive = status == "active" && time.Now().Before(subscriptionEndDate)
		} else {
			h.logger.Printf("CheckSubscriptionStatus: User not found in database: %v", err)
		}
	}
	
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"has_active_subscription": isActive,
		"status":                  status,
		"plan":                    plan,
		"max_websites":            maxWebsites,
		"scan_frequency":          scanFrequency,
		"subscription_end_date":   subscriptionEndDate.Format("2006-01-02"),
		"is_expired":              !isActive,
	})
}

// HandleWebhook handles Razorpay webhooks
func (h *PaymentHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	var webhookData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&webhookData); err != nil {
		h.logger.Printf("HandleWebhook: Invalid webhook data: %v", err)
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid webhook data")
		return
	}
	
	h.logger.Printf("Received webhook: event=%v", webhookData["event"])
	
	// Process webhook event based on type
	event, ok := webhookData["event"].(string)
	if ok {
		switch event {
		case "payment.captured":
			h.logger.Printf("Payment captured: %v", webhookData["payload"])
		case "payment.failed":
			h.logger.Printf("Payment failed: %v", webhookData["payload"])
		}
	}
	
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"received": true,
		"success":  true,
	})
}

// GetPaymentHistory returns user's payment history
func (h *PaymentHandler) GetPaymentHistory(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	if userID == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}
	
	userIDStr := fmt.Sprintf("%v", userID)
	h.logger.Printf("GetPaymentHistory for user: %s", userIDStr)
	
	var payments []Payment
	
	if h.paymentRepo != nil {
		var err error
		payments, err = h.paymentRepo.FindByUserID(userIDStr)
		if err != nil {
			h.logger.Printf("GetPaymentHistory: Error fetching payments: %v", err)
			utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to fetch payments")
			return
		}
	}
	
	if payments == nil {
		payments = []Payment{}
	}
	
	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"payments": payments,
		"success":  true,
	})
}