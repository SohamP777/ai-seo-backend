// payments/plans.go
package payments

import (
    "encoding/json"
    "fmt"
    "strings"
    "time"
    
    "github.com/google/uuid"
    "go.uber.org/zap"
    "gorm.io/gorm"
)

// PaymentPlan represents a SEOSPS pricing plan
type PaymentPlan struct {
    gorm.Model
    Name              string          `gorm:"uniqueIndex;not null"`
    Description       string          `gorm:"not null"`
    RazorpayPlanID    string          `gorm:"uniqueIndex;not null"`
    Amount            int64           `gorm:"not null"` // Monthly amount in cents
    YearlyAmount      int64           `gorm:"not null"` // Yearly amount in cents
    Currency          string          `gorm:"default:'USD';not null"`
    Interval          string          `gorm:"not null"`
    IntervalCount     int             `gorm:"default:1"`
    TrialPeriodDays   int             `gorm:"default:7"`
    Features          json.RawMessage `gorm:"type:jsonb;not null"`
    WebsiteLimit      int             `gorm:"default:1"`
    TeamMembers       int             `gorm:"default:1"`
    AIAutomation      bool            `gorm:"default:true"`
    PrioritySupport   bool            `gorm:"default:false"`
    APIAccess         bool            `gorm:"default:false"`
    WhiteLabel        bool            `gorm:"default:false"`
    RealTimeMonitoring bool           `gorm:"default:false"`
    CustomWorkflows   bool            `gorm:"default:false"`
    DedicatedManager  bool            `gorm:"default:false"`
    SLAGuarantee      bool            `gorm:"default:false"`
    WeeklyScans       bool            `gorm:"default:true"`
    IsActive          bool            `gorm:"default:true"`
    IsDefault         bool            `gorm:"default:false"`
    Recommended       bool            `gorm:"default:false"`
    SortOrder         int             `gorm:"default:0"`
    Tier              string          `gorm:"index;not null"` // essential, pro, agency
    PerfectFor        string          `gorm:"not null"`
    Metadata          json.RawMessage `gorm:"type:jsonb"`
}

// PlanFeature represents a feature in SEOSPS plan
type PlanFeature struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Included    bool   `json:"included"`
    Icon        string `json:"icon,omitempty"`
    Highlight   bool   `json:"highlight,omitempty"`
}

// PlanCreateRequest for creating new plans
type PlanCreateRequest struct {
    Name              string        `json:"name"`
    Tier              string        `json:"tier"`
    Description       string        `json:"description"`
    PerfectFor        string        `json:"perfect_for"`
    MonthlyAmount     float64       `json:"monthly_amount"`
    YearlyAmount      float64       `json:"yearly_amount"`
    Amount            float64       `json:"amount"` // Added for compatibility
    Currency          string        `json:"currency"`
    Interval          string        `json:"interval"`
    IntervalCount     int           `json:"interval_count"`
    TrialPeriodDays   int           `json:"trial_period_days"`
    Features          []PlanFeature `json:"features"`
    WebsiteLimit      int           `json:"website_limit"`
    TeamMembers       int           `json:"team_members"`
    AIAutomation      bool          `json:"ai_automation"`
    PrioritySupport   bool          `json:"priority_support"`
    APIAccess         bool          `json:"api_access"`
    WhiteLabel        bool          `json:"white_label"`
    RealTimeMonitoring bool         `json:"real_time_monitoring"`
    CustomWorkflows   bool          `json:"custom_workflows"`
    DedicatedManager  bool          `json:"dedicated_manager"`
    SLAGuarantee      bool          `json:"sla_guarantee"`
    WeeklyScans       bool          `json:"weekly_scans"`
    IsActive          bool          `json:"is_active"`
    IsDefault         bool          `json:"is_default"`
    Recommended       bool          `json:"recommended"`
}

// PlanManager manages payment plans
type PlanManager struct {
    db             *gorm.DB
    razorpayClient *RazorpayClient
    initialized    bool
    logger         *zap.Logger
}

// NewPlanManager creates a new plan manager
func NewPlanManager(db *gorm.DB, razorpayClient *RazorpayClient, logger *zap.Logger) *PlanManager {
    return &PlanManager{
        db:             db,
        razorpayClient: razorpayClient,
        logger:         logger,
        initialized:    false,
    }
}

// InitializePlans creates default SEOSPS pricing plans
func (pm *PlanManager) InitializePlans() error {
    if pm.initialized {
        return nil
    }

    // Check if plans already exist
    var count int64
    if err := pm.db.Model(&PaymentPlan{}).Count(&count).Error; err != nil {
        return fmt.Errorf("failed to check existing plans: %v", err)
    }
    
    if count > 0 {
        pm.initialized = true
        return nil
    }

    // SEOSPS TIERED PRICING PLANS
    defaultPlans := []PlanCreateRequest{
        {
            Name:              "Essential",
            Tier:              "essential",
            Description:       "AI-Powered SEO automation for solo entrepreneurs",
            PerfectFor:        "Small businesses, freelancers, solo entrepreneurs",
            MonthlyAmount:     49.00,
            YearlyAmount:      470.00, // Save 20%
            Currency:          "USD",
            Interval:          "monthly",
            IntervalCount:     1,
            TrialPeriodDays:   7,
            WebsiteLimit:      1,
            TeamMembers:       1,
            AIAutomation:      true,
            PrioritySupport:   false,
            APIAccess:         false,
            WhiteLabel:        false,
            RealTimeMonitoring: false,
            CustomWorkflows:   false,
            DedicatedManager:  false,
            SLAGuarantee:      false,
            WeeklyScans:       true,
            Recommended:       false,
            IsActive:          true,
            IsDefault:         false,
            Features: []PlanFeature{
                {Name: "1 Website", Description: "Monitor 1 website", Included: true, Icon: "globe", Highlight: true},
                {Name: "AI-Powered SEO Diagnostics", Description: "Advanced AI diagnostics", Included: true, Icon: "cpu", Highlight: true},
                {Name: "1-Click Fix for 45+ Issues", Description: "Automated issue resolution", Included: true, Icon: "zap", Highlight: true},
                {Name: "Weekly Automated Scans", Description: "Weekly SEO scans", Included: true, Icon: "refresh-cw"},
                {Name: "Basic SEO Reports", Description: "Essential performance reports", Included: true, Icon: "file-text"},
                {Name: "Email Support", Description: "Standard email support", Included: true, Icon: "mail"},
                {Name: "Real-time Monitoring", Description: "24/7 real-time monitoring", Included: false, Icon: "activity"},
                {Name: "Team Collaboration", Description: "Multiple team members", Included: false, Icon: "users"},
                {Name: "API Access", Description: "API integration", Included: false, Icon: "code"},
                {Name: "White-label Reports", Description: "Branded reports", Included: false, Icon: "package"},
            },
        },
        {
            Name:              "Pro",
            Tier:              "pro",
            Description:       "Advanced SEO automation for growing businesses",
            PerfectFor:        "Agencies, growing businesses, consultants",
            MonthlyAmount:     149.00,
            YearlyAmount:      1430.00, // Save 20%
            Currency:          "USD",
            Interval:          "monthly",
            IntervalCount:     1,
            TrialPeriodDays:   14,
            WebsiteLimit:      10,
            TeamMembers:       3,
            AIAutomation:      true,
            PrioritySupport:   true,
            APIAccess:         true,
            WhiteLabel:        false,
            RealTimeMonitoring: true,
            CustomWorkflows:   false,
            DedicatedManager:  false,
            SLAGuarantee:      false,
            WeeklyScans:       true,
            Recommended:       true, // MOST POPULAR
            IsActive:          true,
            IsDefault:         false,
            Features: []PlanFeature{
                {Name: "10 Websites", Description: "Monitor up to 10 websites", Included: true, Icon: "globe", Highlight: true},
                {Name: "Everything in Essential", Description: "All Essential features", Included: true, Icon: "check-circle", Highlight: true},
                {Name: "Real-time Monitoring", Description: "24/7 real-time monitoring", Included: true, Icon: "activity", Highlight: true},
                {Name: "Advanced AI Diagnostics", Description: "Enhanced AI diagnostics", Included: true, Icon: "cpu"},
                {Name: "Priority 1-Click Fixes", Description: "Priority issue resolution", Included: true, Icon: "zap"},
                {Name: "Team Collaboration (3 users)", Description: "3 team members", Included: true, Icon: "users"},
                {Name: "API Access", Description: "Full API integration", Included: true, Icon: "code"},
                {Name: "Priority Email Support", Description: "Priority support", Included: true, Icon: "mail"},
                {Name: "White-label Reports", Description: "Branded reports", Included: false, Icon: "package"},
                {Name: "Custom Workflows", Description: "Custom automation", Included: false, Icon: "workflow"},
            },
        },
        {
            Name:              "Agency",
            Tier:              "agency",
            Description:       "Enterprise-grade SEO automation for agencies",
            PerfectFor:        "SEO agencies, enterprise teams, multiple clients",
            MonthlyAmount:     299.00,
            YearlyAmount:      2870.00, // Save 20%
            Currency:          "USD",
            Interval:          "monthly",
            IntervalCount:     1,
            TrialPeriodDays:   14,
            WebsiteLimit:      25,
            TeamMembers:       0, // 0 = unlimited
            AIAutomation:      true,
            PrioritySupport:   true,
            APIAccess:         true,
            WhiteLabel:        true,
            RealTimeMonitoring: true,
            CustomWorkflows:   true,
            DedicatedManager:  true,
            SLAGuarantee:      true,
            WeeklyScans:       true,
            Recommended:       false,
            IsActive:          true,
            IsDefault:         false,
            Features: []PlanFeature{
                {Name: "25 Websites", Description: "Monitor up to 25 websites", Included: true, Icon: "globe", Highlight: true},
                {Name: "Everything in Pro", Description: "All Pro features", Included: true, Icon: "check-circle", Highlight: true},
                {Name: "White-label Reports", Description: "Fully branded reports", Included: true, Icon: "package", Highlight: true},
                {Name: "Unlimited Team Members", Description: "Unlimited team access", Included: true, Icon: "users"},
                {Name: "Custom Automation Workflows", Description: "Custom workflows", Included: true, Icon: "workflow"},
                {Name: "Dedicated Account Manager", Description: "Personal account manager", Included: true, Icon: "user-check"},
                {Name: "24/7 Priority Support", Description: "24/7 priority support", Included: true, Icon: "headphones"},
                {Name: "SLA Guarantee", Description: "Service level agreement", Included: true, Icon: "shield"},
                {Name: "Advanced Security", Description: "Enterprise security", Included: true, Icon: "lock"},
                {Name: "Custom Integrations", Description: "Custom API integrations", Included: true, Icon: "git-merge"},
            },
        },
    }

    // Create each plan (both monthly and yearly)
    for _, planReq := range defaultPlans {
        // Create monthly plan
        monthlyReq := planReq
        monthlyReq.Interval = "monthly"
        monthlyReq.Amount = planReq.MonthlyAmount
        _, err := pm.CreatePlan(&monthlyReq)
        if err != nil {
            return fmt.Errorf("failed to create monthly plan %s: %v", planReq.Name, err)
        }

        // Create yearly plan
        yearlyReq := planReq
        yearlyReq.Name = planReq.Name + " (Yearly)"
        yearlyReq.Interval = "yearly"
        yearlyReq.IntervalCount = 1
        yearlyReq.Amount = planReq.YearlyAmount
        yearlyReq.Recommended = false // Only mark monthly as recommended
        _, err = pm.CreatePlan(&yearlyReq)
        if err != nil {
            return fmt.Errorf("failed to create yearly plan %s: %v", planReq.Name, err)
        }
    }

    pm.initialized = true
    fmt.Println("✅ SEOSPS tiered pricing plans initialized successfully")
    return nil
}

// CreatePlan creates a new SEOSPS plan
func (pm *PlanManager) CreatePlan(req *PlanCreateRequest) (*PaymentPlan, error) {
    // Validate required fields
    if req.Name == "" {
        return nil, fmt.Errorf("plan name is required")
    }
    if req.Tier == "" {
        return nil, fmt.Errorf("plan tier is required")
    }
    if req.Amount <= 0 {
        return nil, fmt.Errorf("plan amount must be greater than 0")
    }
    if req.WebsiteLimit <= 0 {
        return nil, fmt.Errorf("website limit must be greater than 0")
    }

    // Convert amount to cents
    amountInCents := int64(req.Amount * 100)
    yearlyAmountInCents := int64(req.YearlyAmount * 100)

    // Check if plan already exists
    var existingPlan PaymentPlan
    if err := pm.db.Where("name = ? AND interval = ?", req.Name, req.Interval).First(&existingPlan).Error; err == nil {
        return &existingPlan, nil // Plan already exists
    }

    // Create plan in Razorpay
    razorpayPlan, err := pm.createRazorpayPlan(req, amountInCents)
    if err != nil {
        return nil, fmt.Errorf("failed to create Razorpay plan: %v", err)
    }

    // Convert features to JSON
    featuresJSON, err := json.Marshal(req.Features)
    if err != nil {
        pm.deleteRazorpayPlan(razorpayPlan["id"].(string))
        return nil, fmt.Errorf("failed to marshal features: %v", err)
    }

    // Create metadata
    metadata := map[string]interface{}{
        "created_at":          time.Now().Format(time.RFC3339),
        "tier":                req.Tier,
        "website_limit":       req.WebsiteLimit,
        "team_members":        req.TeamMembers,
        "ai_automation":       req.AIAutomation,
        "priority_support":    req.PrioritySupport,
        "api_access":          req.APIAccess,
        "white_label":         req.WhiteLabel,
        "real_time_monitoring": req.RealTimeMonitoring,
        "custom_workflows":    req.CustomWorkflows,
        "dedicated_manager":   req.DedicatedManager,
        "sla_guarantee":       req.SLAGuarantee,
        "weekly_scans":        req.WeeklyScans,
        "recommended":         req.Recommended,
        "perfect_for":         req.PerfectFor,
        "yearly_amount":       yearlyAmountInCents,
        "yearly_savings":      20, // 20% savings
    }
    metadataJSON, _ := json.Marshal(metadata)

    // Create local plan record
    plan := &PaymentPlan{
        Name:               req.Name,
        Tier:               req.Tier,
        Description:        req.Description,
        PerfectFor:         req.PerfectFor,
        RazorpayPlanID:     razorpayPlan["id"].(string),
        Amount:             amountInCents,
        YearlyAmount:       yearlyAmountInCents,
        Currency:           req.Currency,
        Interval:           req.Interval,
        IntervalCount:      req.IntervalCount,
        TrialPeriodDays:    req.TrialPeriodDays,
        Features:           featuresJSON,
        WebsiteLimit:       req.WebsiteLimit,
        TeamMembers:        req.TeamMembers,
        AIAutomation:       req.AIAutomation,
        PrioritySupport:    req.PrioritySupport,
        APIAccess:          req.APIAccess,
        WhiteLabel:         req.WhiteLabel,
        RealTimeMonitoring: req.RealTimeMonitoring,
        CustomWorkflows:    req.CustomWorkflows,
        DedicatedManager:   req.DedicatedManager,
        SLAGuarantee:       req.SLAGuarantee,
        WeeklyScans:        req.WeeklyScans,
        IsActive:           req.IsActive,
        IsDefault:          req.IsDefault,
        Recommended:        req.Recommended,
        SortOrder:          pm.getSortOrder(req.Tier, req.Interval),
        Metadata:           metadataJSON,
    }

    // Save to database
    if err := pm.db.Create(plan).Error; err != nil {
        pm.deleteRazorpayPlan(razorpayPlan["id"].(string))
        return nil, fmt.Errorf("failed to save plan to database: %v", err)
    }

    fmt.Printf("✅ Created %s plan: %s - $%.2f/%s (%d websites)\n", 
        req.Tier, req.Name, req.Amount, req.Interval, req.WebsiteLimit)
    
    return plan, nil
}

// createRazorpayPlan creates a plan in Razorpay
func (pm *PlanManager) createRazorpayPlan(req *PlanCreateRequest, amount int64) (map[string]interface{}, error) {
    // Build plan data for Razorpay
    planData := map[string]interface{}{
        "period":   req.Interval,
        "interval": req.IntervalCount,
        "item": map[string]interface{}{
            "name":        fmt.Sprintf("SEOSPS %s - %s", req.Tier, req.Name),
            "description": fmt.Sprintf("%s - %d websites - %s", req.Description, req.WebsiteLimit, req.PerfectFor),
            "amount":      amount,
            "currency":    req.Currency,
        },
        "notes": map[string]interface{}{
            "service":        "seosps",
            "tier":           req.Tier,
            "website_limit":  req.WebsiteLimit,
            "team_members":   req.TeamMembers,
            "ai_automation":  req.AIAutomation,
            "api_access":     req.APIAccess,
            "white_label":    req.WhiteLabel,
            "perfect_for":    req.PerfectFor,
        },
    }

    // Add trial period if specified
    if req.TrialPeriodDays > 0 {
        planData["trial_period"] = req.TrialPeriodDays
    }

    // Make API request to Razorpay
    resp, err := pm.razorpayClient.makeRequest("POST", "/v1/plans", planData)
    if err != nil {
        return nil, fmt.Errorf("razorpay API error: %v", err)
    }

    var razorpayPlan map[string]interface{}
    if err := json.Unmarshal(resp, &razorpayPlan); err != nil {
        return nil, fmt.Errorf("failed to parse Razorpay response: %v", err)
    }

    return razorpayPlan, nil
}

// deleteRazorpayPlan deletes a plan from Razorpay (for cleanup on failure)
func (pm *PlanManager) deleteRazorpayPlan(planID string) {
    if planID == "" {
        return
    }
    
    _, err := pm.razorpayClient.makeRequest("DELETE", "/v1/plans/"+planID, nil)
    if err != nil && pm.logger != nil {
        pm.logger.Error("Failed to delete Razorpay plan during cleanup", 
            zap.String("plan_id", planID), 
            zap.Error(err))
    }
}

// getSortOrder determines display order
func (pm *PlanManager) getSortOrder(tier, interval string) int {
    tierOrder := map[string]int{
        "essential": 1,
        "pro":       2,
        "agency":    3,
    }
    
    intervalBonus := 0
    if interval == "yearly" {
        intervalBonus = 10 // Yearly plans come after monthly
    }
    
    if order, exists := tierOrder[tier]; exists {
        return order + intervalBonus
    }
    return 0
}

// GetPlanByID retrieves a plan by ID
func (pm *PlanManager) GetPlanByID(planID uuid.UUID) (*PaymentPlan, error) {
    if planID == uuid.Nil {
        return nil, fmt.Errorf("invalid plan ID")
    }

    var plan PaymentPlan
    if err := pm.db.Where("id = ?", planID).First(&plan).Error; err != nil {
        return nil, fmt.Errorf("plan not found: %v", err)
    }
    return &plan, nil
}

// GetPlanByTier returns the monthly plan for a specific tier
func (pm *PlanManager) GetPlanByTier(tier string, interval string) (*PaymentPlan, error) {
    if tier == "" {
        return nil, fmt.Errorf("tier is required")
    }
    if interval == "" {
        interval = "monthly"
    }

    var plan PaymentPlan
    if err := pm.db.Where("tier = ? AND interval = ?", tier, interval).First(&plan).Error; err != nil {
        return nil, fmt.Errorf("%s %s plan not found: %v", tier, interval, err)
    }
    return &plan, nil
}

// GetRecommendedPlan returns the recommended plan (Pro monthly)
func (pm *PlanManager) GetRecommendedPlan() (*PaymentPlan, error) {
    var plan PaymentPlan
    if err := pm.db.Where("recommended = ? AND interval = ?", true, "monthly").
        First(&plan).Error; err != nil {
        // Fallback: Get Pro monthly plan
        return pm.GetPlanByTier("pro", "monthly")
    }
    
    return &plan, nil
}

// GetAllPlansByTier returns all plans grouped by tier
func (pm *PlanManager) GetAllPlansByTier(includeInactive bool) (map[string][]PaymentPlan, error) {
    var plans []PaymentPlan
    query := pm.db.Model(&PaymentPlan{})
    
    if !includeInactive {
        query = query.Where("is_active = ?", true)
    }
    
    if err := query.Order("tier, sort_order").Find(&plans).Error; err != nil {
        return nil, fmt.Errorf("failed to fetch plans: %v", err)
    }
    
    // Group by tier
    grouped := make(map[string][]PaymentPlan)
    for _, plan := range plans {
        grouped[plan.Tier] = append(grouped[plan.Tier], plan)
    }
    
    return grouped, nil
}

// GetYearlySavings calculates savings percentage for yearly plan
func (p *PaymentPlan) GetYearlySavings() float64 {
    if p.Interval == "yearly" {
        return 20.0 // Fixed 20% savings
    }
    return 0.0
}

// GetDisplayAmount returns formatted amount
func (p *PaymentPlan) GetDisplayAmount() string {
    amount := float64(p.Amount) / 100
    return fmt.Sprintf("$%.2f", amount)
}

// GetYearlyDisplayAmount returns formatted yearly amount
func (p *PaymentPlan) GetYearlyDisplayAmount() string {
    yearlyAmount := float64(p.YearlyAmount) / 100
    return fmt.Sprintf("$%.2f", yearlyAmount)
}

// GetMonthlyEquivalent returns monthly equivalent for yearly plans
func (p *PaymentPlan) GetMonthlyEquivalent() float64 {
    if p.Interval == "yearly" {
        return float64(p.YearlyAmount) / 100 / 12
    }
    return float64(p.Amount) / 100
}

// GetPricingDisplay returns pricing info for display
func (p *PaymentPlan) GetPricingDisplay() map[string]string {
    result := map[string]string{
        "monthly": p.GetDisplayAmount() + "/month",
    }
    
    if p.YearlyAmount > 0 {
        yearlyMonthly := p.GetMonthlyEquivalent()
        savings := p.GetYearlySavings()
        result["yearly"] = fmt.Sprintf("$%.0f/year (Save %.0f%%)", 
            float64(p.YearlyAmount)/100, savings)
        result["yearly_monthly"] = fmt.Sprintf("$%.2f/month", yearlyMonthly)
    }
    
    return result
}

// HasFeature checks if plan includes a feature
func (p *PaymentPlan) HasFeature(featureName string) bool {
    var features []PlanFeature
    if err := json.Unmarshal(p.Features, &features); err != nil {
        return false
    }
    
    searchName := strings.ToLower(strings.TrimSpace(featureName))
    for _, feature := range features {
        if strings.ToLower(feature.Name) == searchName && feature.Included {
            return true
        }
    }
    return false
}

// GetFeaturesList returns parsed features
func (p *PaymentPlan) GetFeaturesList() ([]PlanFeature, error) {
    var features []PlanFeature
    if err := json.Unmarshal(p.Features, &features); err != nil {
        return nil, fmt.Errorf("failed to parse features: %v", err)
    }
    return features, nil
}

// GetHighlightedFeatures returns only highlighted features
func (p *PaymentPlan) GetHighlightedFeatures() ([]PlanFeature, error) {
    features, err := p.GetFeaturesList()
    if err != nil {
        return nil, err
    }
    
    var highlighted []PlanFeature
    for _, feature := range features {
        if feature.Highlight {
            highlighted = append(highlighted, feature)
        }
    }
    return highlighted, nil
}

// ValidateTeamMembers validates team member count for plan
func (pm *PlanManager) ValidateTeamMembers(planID uuid.UUID, teamCount int) (bool, string) {
    plan, err := pm.GetPlanByID(planID)
    if err != nil {
        return false, "Plan not found"
    }
    
    if plan.TeamMembers == 0 { // 0 means unlimited
        return true, ""
    }
    
    if teamCount > plan.TeamMembers {
        return false, fmt.Sprintf(
            "%s plan allows maximum %d team members. Please upgrade to Agency plan for unlimited team members.",
            plan.Name, plan.TeamMembers,
        )
    }
    
    return true, ""
}

// GetPlanComparison returns plans for comparison table
func (pm *PlanManager) GetPlanComparison() ([]map[string]interface{}, error) {
    // Get only monthly plans for comparison
    var plans []PaymentPlan
    if err := pm.db.Where("is_active = ? AND interval = ?", true, "monthly").
        Order("sort_order ASC").Find(&plans).Error; err != nil {
        return nil, fmt.Errorf("failed to fetch plans: %v", err)
    }
    
    var comparison []map[string]interface{}
    
    for _, plan := range plans {
        features, _ := plan.GetFeaturesList()
        
        comparison = append(comparison, map[string]interface{}{
            "id":                   plan.ID,
            "name":                 plan.Name,
            "tier":                 plan.Tier,
            "description":          plan.Description,
            "perfect_for":          plan.PerfectFor,
            "monthly_amount":       plan.GetDisplayAmount(),
            "yearly_amount":        plan.GetYearlyDisplayAmount(),
            "yearly_savings":       plan.GetYearlySavings(),
            "monthly_equivalent":   fmt.Sprintf("$%.2f", plan.GetMonthlyEquivalent()),
            "websites":             plan.WebsiteLimit,
            "team_members":         plan.TeamMembers,
            "ai_automation":        plan.AIAutomation,
            "priority_support":     plan.PrioritySupport,
            "api_access":           plan.APIAccess,
            "white_label":          plan.WhiteLabel,
            "real_time_monitoring": plan.RealTimeMonitoring,
            "custom_workflows":     plan.CustomWorkflows,
            "dedicated_manager":    plan.DedicatedManager,
            "sla_guarantee":        plan.SLAGuarantee,
            "weekly_scans":         plan.WeeklyScans,
            "recommended":          plan.Recommended,
            "most_popular":         plan.Tier == "pro" && plan.Interval == "monthly",
            "features":             features,
            "trial_days":           plan.TrialPeriodDays,
        })
    }
    
    return comparison, nil
}

// CalculateYearlyPrice calculates yearly price with savings
func (pm *PlanManager) CalculateYearlyPrice(monthlyPrice float64) (float64, float64) {
    yearlyPrice := monthlyPrice * 12 * 0.80 // 20% savings
    savings := monthlyPrice * 12 * 0.20
    return yearlyPrice, savings
}

// GetUpgradePath suggests upgrade path based on current usage
func (pm *PlanManager) GetUpgradePath(currentPlanID uuid.UUID, currentWebsites int, currentTeamMembers int) ([]PaymentPlan, error) {
    currentPlan, err := pm.GetPlanByID(currentPlanID)
    if err != nil {
        return nil, err
    }
    
    var suggestedPlans []PaymentPlan
    
    // Check if user needs more websites
    if currentWebsites >= currentPlan.WebsiteLimit {
        var higherPlans []PaymentPlan
        if err := pm.db.Where("is_active = ? AND interval = ? AND website_limit > ?", 
            true, currentPlan.Interval, currentPlan.WebsiteLimit).
            Order("website_limit ASC").Find(&higherPlans).Error; err != nil {
            return nil, err
        }
        suggestedPlans = append(suggestedPlans, higherPlans...)
    }
    
    // Check if user needs more team members
    if currentPlan.TeamMembers > 0 && currentTeamMembers >= currentPlan.TeamMembers {
        var teamPlans []PaymentPlan
        if err := pm.db.Where("is_active = ? AND interval = ? AND (team_members > ? OR team_members = 0)", 
            true, currentPlan.Interval, currentPlan.TeamMembers).
            Order("team_members ASC").Find(&teamPlans).Error; err != nil {
            return nil, err
        }
        suggestedPlans = append(suggestedPlans, teamPlans...)
    }
    
    return suggestedPlans, nil
}

// GetPlanSummary returns plan summary for display
func (p *PaymentPlan) GetPlanSummary() map[string]interface{} {
    highlighted, _ := p.GetHighlightedFeatures()
    
    return map[string]interface{}{
        "id":           p.ID,
        "name":         p.Name,
        "tier":         p.Tier,
        "description":  p.Description,
        "perfect_for":  p.PerfectFor,
        "pricing":      p.GetPricingDisplay(),
        "websites":     p.WebsiteLimit,
        "team_members": p.TeamMembers,
        "highlights":   highlighted,
        "recommended":  p.Recommended,
        "trial_days":   p.TrialPeriodDays,
        "interval":     p.Interval,
    }
}

// Business tier methods
func (pm *PlanManager) IsAgencyPlan(planID uuid.UUID) (bool, error) {
    plan, err := pm.GetPlanByID(planID)
    if err != nil {
        return false, err
    }
    return plan.Tier == "agency", nil
}

func (pm *PlanManager) IsProPlan(planID uuid.UUID) (bool, error) {
    plan, err := pm.GetPlanByID(planID)
    if err != nil {
        return false, err
    }
    return plan.Tier == "pro", nil
}

func (pm *PlanManager) IsEssentialPlan(planID uuid.UUID) (bool, error) {
    plan, err := pm.GetPlanByID(planID)
    if err != nil {
        return false, err
    }
    return plan.Tier == "essential", nil
}