// Package models contains data structures for payment processing
package models

import (
	"time"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusPaid      PaymentStatus = "paid"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

// SubscriptionStatus represents subscription state
type SubscriptionStatus string

const (
	SubscriptionActive    SubscriptionStatus = "active"
	SubscriptionPastDue   SubscriptionStatus = "past_due"
	SubscriptionCanceled  SubscriptionStatus = "canceled"
	SubscriptionIncomplete SubscriptionStatus = "incomplete"
	SubscriptionTrialing  SubscriptionStatus = "trialing"
	SubscriptionExpired   SubscriptionStatus = "expired"
)

// BillingPeriod represents billing cycle
type BillingPeriod string

const (
	BillingMonthly BillingPeriod = "monthly"
	BillingYearly  BillingPeriod = "yearly"
	BillingLifetime BillingPeriod = "lifetime"
)

// PaymentProvider represents payment processor
type PaymentProvider string

const (
	ProviderStripe     PaymentProvider = "stripe"
	ProviderPayPal     PaymentProvider = "paypal"
	ProviderPaddle     PaymentProvider = "paddle"
	ProviderManual     PaymentProvider = "manual" // Invoice/manual payment
)

// Subscription represents a user subscription
type Subscription struct {
	ID                string             `json:"id" db:"id"`
	UserID            string             `json:"user_id" db:"user_id"`
	PlanID            string             `json:"plan_id" db:"plan_id"`
	Status            SubscriptionStatus  `json:"status" db:"status"`
	Tier              SubscriptionTier    `json:"tier" db:"tier"`
	BillingPeriod     BillingPeriod       `json:"billing_period" db:"billing_period"`
	Provider          PaymentProvider     `json:"provider" db:"provider"`
	ProviderSubscriptionID string         `json:"provider_subscription_id" db:"provider_subscription_id"`
	ProviderCustomerID string              `json:"provider_customer_id" db:"provider_customer_id"`
	
	// Pricing
	Amount            float64            `json:"amount" db:"amount"`
	Currency          string             `json:"currency" db:"currency"`
	TaxRate           float64            `json:"tax_rate" db:"tax_rate"`
	TaxAmount         float64            `json:"tax_amount" db:"tax_amount"`
	TotalAmount       float64            `json:"total_amount" db:"total_amount"`
	
	// Dates
	StartDate         time.Time          `json:"start_date" db:"start_date"`
	CurrentPeriodStart time.Time          `json:"current_period_start" db:"current_period_start"`
	CurrentPeriodEnd  time.Time          `json:"current_period_end" db:"current_period_end"`
	TrialStart        *time.Time         `json:"trial_start,omitempty" db:"trial_start"`
	TrialEnd          *time.Time         `json:"trial_end,omitempty" db:"trial_end"`
	CanceledAt        *time.Time         `json:"canceled_at,omitempty" db:"canceled_at"`
	EndedAt           *time.Time         `json:"ended_at,omitempty" db:"ended_at"`
	
	// Features
	FeatureUsage      map[string]int     `json:"feature_usage" db:"feature_usage"`
	FeatureLimits     map[string]int     `json:"feature_limits" db:"feature_limits"`
	
	// Metadata
	CreatedAt         time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at" db:"updated_at"`
	CancellationReason string             `json:"cancellation_reason,omitempty" db:"cancellation_reason"`
	AutoRenew         bool                `json:"auto_renew" db:"auto_renew"`
}

// Plan represents a subscription plan
type Plan struct {
	ID            string             `json:"id" db:"id"`
	Name          string             `json:"name" db:"name"`
	Description   string             `json:"description" db:"description"`
	Tier          SubscriptionTier    `json:"tier" db:"tier"`
	BillingPeriod BillingPeriod       `json:"billing_period" db:"billing_period"`
	Price         float64             `json:"price" db:"price"`
	Currency      string              `json:"currency" db:"currency"`
	
	// Provider IDs
	StripePriceID string             `json:"stripe_price_id" db:"stripe_price_id"`
	PayPalPlanID  string             `json:"paypal_plan_id" db:"paypal_plan_id"`
	
	// Features
	MaxScansPerDay   int              `json:"max_scans_per_day" db:"max_scans_per_day"`
	MaxPagesPerScan  int              `json:"max_pages_per_scan" db:"max_pages_per_scan"`
	MaxDomains       int              `json:"max_domains" db:"max_domains"`
	ApiRequestsPerHour int            `json:"api_requests_per_hour" db:"api_requests_per_hour"`
	TeamMembers      int              `json:"team_members" db:"team_members"`
	
	// Features flags
	HasAdvancedReporting bool          `json:"has_advanced_reporting" db:"has_advanced_reporting"`
	HasScheduledScans   bool           `json:"has_scheduled_scans" db:"has_scheduled_scans"`
	HasApiAccess        bool           `json:"has_api_access" db:"has_api_access"`
	HasWhiteLabel       bool           `json:"has_white_label" db:"has_white_label"`
	HasPrioritySupport   bool           `json:"has_priority_support" db:"has_priority_support"`
	
	// Metadata
	IsActive        bool               `json:"is_active" db:"is_active"`
	IsPopular       bool               `json:"is_popular" db:"is_popular"`
	SortOrder       int                `json:"sort_order" db:"sort_order"`
	CreatedAt       time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at" db:"updated_at"`
}

// Invoice represents a billing invoice
type Invoice struct {
	ID              string         `json:"id" db:"id"`
	UserID          string         `json:"user_id" db:"user_id"`
	SubscriptionID  string         `json:"subscription_id" db:"subscription_id"`
	InvoiceNumber   string         `json:"invoice_number" db:"invoice_number"`
	Status          PaymentStatus  `json:"status" db:"status"`
	
	// Amounts
	Subtotal        float64        `json:"subtotal" db:"subtotal"`
	Discount        float64        `json:"discount" db:"discount"`
	Tax             float64        `json:"tax" db:"tax"`
	Total           float64        `json:"total" db:"total"`
	Currency        string         `json:"currency" db:"currency"`
	
	// Payment
	Provider        PaymentProvider `json:"provider" db:"provider"`
	ProviderInvoiceID string       `json:"provider_invoice_id" db:"provider_invoice_id"`
	PaymentMethod   string         `json:"payment_method" db:"payment_method"`
	PaidAt          *time.Time     `json:"paid_at,omitempty" db:"paid_at"`
	
	// Dates
	IssueDate       time.Time      `json:"issue_date" db:"issue_date"`
	DueDate         time.Time      `json:"due_date" db:"due_date"`
	
	// Items
	Items           []InvoiceItem  `json:"items" db:"-"`
	Metadata        map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	
	// Files
	PDFURL          string         `json:"pdf_url" db:"pdf_url"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
}

// InvoiceItem represents a line item on an invoice
type InvoiceItem struct {
	ID          string  `json:"id" db:"id"`
	InvoiceID   string  `json:"invoice_id" db:"invoice_id"`
	Description string  `json:"description" db:"description"`
	Quantity    int     `json:"quantity" db:"quantity"`
	UnitPrice   float64 `json:"unit_price" db:"unit_price"`
	Amount      float64 `json:"amount" db:"amount"`
	IsTaxable   bool    `json:"is_taxable" db:"is_taxable"`
	PeriodStart *time.Time `json:"period_start,omitempty" db:"period_start"`
	PeriodEnd   *time.Time `json:"period_end,omitempty" db:"period_end"`
}

// Payment represents a payment transaction
type Payment struct {
	ID              string         `json:"id" db:"id"`
	UserID          string         `json:"user_id" db:"user_id"`
	InvoiceID       string         `json:"invoice_id" db:"invoice_id"`
	SubscriptionID  string         `json:"subscription_id" db:"subscription_id"`
	Amount          float64        `json:"amount" db:"amount"`
	Currency        string         `json:"currency" db:"currency"`
	Status          PaymentStatus  `json:"status" db:"status"`
	Provider        PaymentProvider `json:"provider" db:"provider"`
	ProviderPaymentID string       `json:"provider_payment_id" db:"provider_payment_id"`
	PaymentMethod   string         `json:"payment_method" db:"payment_method"`
	CardLastFour    string         `json:"card_last_four,omitempty" db:"card_last_four"`
	CardBrand       string         `json:"card_brand,omitempty" db:"card_brand"`
	ErrorMessage    string         `json:"error_message,omitempty" db:"error_message"`
	RefundedAt      *time.Time     `json:"refunded_at,omitempty" db:"refunded_at"`
	RefundAmount    float64        `json:"refund_amount,omitempty" db:"refund_amount"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
}

// Coupon represents a discount coupon
type Coupon struct {
	ID              string     `json:"id" db:"id"`
	Code            string     `json:"code" db:"code"`
	Description     string     `json:"description" db:"description"`
	DiscountType    string     `json:"discount_type" db:"discount_type"` // percentage, fixed
	DiscountValue   float64    `json:"discount_value" db:"discount_value"`
	Currency        string     `json:"currency,omitempty" db:"currency"` // for fixed discounts
	
	// Restrictions
	MaxRedemptions  int        `json:"max_redemptions" db:"max_redemptions"`
	RedemptionCount int        `json:"redemption_count" db:"redemption_count"`
	MinAmount       float64    `json:"min_amount,omitempty" db:"min_amount"`
	ValidForPlans   []string   `json:"valid_for_plans,omitempty" db:"valid_for_plans"`
	ValidForUsers   []string   `json:"valid_for_users,omitempty" db:"valid_for_users"` // specific users
	
	// Dates
	ValidFrom       *time.Time `json:"valid_from,omitempty" db:"valid_from"`
	ValidUntil      *time.Time `json:"valid_until,omitempty" db:"valid_until"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	IsActive        bool       `json:"is_active" db:"is_active"`
}

// PaymentMethod represents a saved payment method
type PaymentMethod struct {
	ID              string         `json:"id" db:"id"`
	UserID          string         `json:"user_id" db:"user_id"`
	Provider        PaymentProvider `json:"provider" db:"provider"`
	ProviderMethodID string        `json:"provider_method_id" db:"provider_method_id"`
	Type            string         `json:"type" db:"type"` // card, paypal, bank
	IsDefault       bool           `json:"is_default" db:"is_default"`
	
	// Card details (partial)
	CardLastFour    string         `json:"card_last_four,omitempty" db:"card_last_four"`
	CardBrand       string         `json:"card_brand,omitempty" db:"card_brand"`
	CardExpMonth    int            `json:"card_exp_month,omitempty" db:"card_exp_month"`
	CardExpYear     int            `json:"card_exp_year,omitempty" db:"card_exp_year"`
	
	// Billing details
	BillingName     string         `json:"billing_name" db:"billing_name"`
	BillingEmail    string         `json:"billing_email" db:"billing_email"`
	BillingAddress  *Address       `json:"billing_address,omitempty" db:"billing_address"`
	
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
}

// Address represents a physical address
type Address struct {
	Line1      string `json:"line1" db:"line1"`
	Line2      string `json:"line2,omitempty" db:"line2"`
	City       string `json:"city" db:"city"`
	State      string `json:"state" db:"state"`
	PostalCode string `json:"postal_code" db:"postal_code"`
	Country    string `json:"country" db:"country"`
}

// SubscriptionChange represents a subscription change request
type SubscriptionChange struct {
	UserID         string          `json:"user_id"`
	SubscriptionID string          `json:"subscription_id"`
	NewPlanID      string          `json:"new_plan_id"`
	BillingPeriod  BillingPeriod   `json:"billing_period"`
	Prorate        bool            `json:"prorate"`
	CouponCode     string          `json:"coupon_code,omitempty"`
	Immediate      bool            `json:"immediate"` // Change now vs end of period
}

// ==================== HELPER METHODS ====================

// IsActive checks if subscription is active
func (s *Subscription) IsActive() bool {
	return s.Status == SubscriptionActive || s.Status == SubscriptionTrialing
}

// IsCanceled checks if subscription is canceled
func (s *Subscription) IsCanceled() bool {
	return s.Status == SubscriptionCanceled
}

// DaysRemaining returns days left in current period
func (s *Subscription) DaysRemaining() int {
	if time.Now().After(s.CurrentPeriodEnd) {
		return 0
	}
	return int(time.Until(s.CurrentPeriodEnd).Hours() / 24)
}

// IsTrialing checks if subscription is in trial
func (s *Subscription) IsTrialing() bool {
	return s.Status == SubscriptionTrialing
}

// GetFeatureLimit returns limit for a specific feature
func (p *Plan) GetFeatureLimit(feature string) int {
	switch feature {
	case "scans_per_day":
		return p.MaxScansPerDay
	case "pages_per_scan":
		return p.MaxPagesPerScan
	case "max_domains":
		return p.MaxDomains
	case "api_requests":
		return p.ApiRequestsPerHour
	case "team_members":
		return p.TeamMembers
	default:
		return 0
	}
}