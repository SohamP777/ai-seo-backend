// payments/middleware.go
package payments

import (
    "context"
    "fmt"
    "net/http"
    "strings"
    "time"
    "io"
    "bytes"
    
    "github.com/google/uuid"
    "github.com/golang-jwt/jwt/v5"
    "gorm.io/gorm"
)

// Context keys for storing payment/subscription data in request context
type contextKey string

const (
    UserIDKey          contextKey = "user_id"
    SubscriptionKey    contextKey = "subscription"
    PlanKey           contextKey = "plan"
    PaymentKey        contextKey = "payment"
    HasAccessKey      contextKey = "has_access"
)

// MiddlewareConfig holds configuration for payment middleware
type MiddlewareConfig struct {
    JWTSecretKey      string
    RequirePayment    bool          // Whether payment is required for all routes
    FreeTrialDays     int           // Days of free trial
    GracePeriodDays   int           // Grace period after subscription expires
    WhitelistedPaths  []string      // Paths that don't require payment
    AdminPaths        []string      // Admin paths with different access rules
}

// AuthMiddleware authenticates users and extracts user ID from JWT
func AuthMiddleware(config *MiddlewareConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Skip auth for whitelisted paths
            for _, path := range config.WhitelistedPaths {
                if strings.HasPrefix(r.URL.Path, path) {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            
            // Extract JWT token
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                http.Error(w, "Authorization header required", http.StatusUnauthorized)
                return
            }
            
            // Expecting "Bearer <token>"
            parts := strings.Split(authHeader, " ")
            if len(parts) != 2 || parts[0] != "Bearer" {
                http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
                return
            }
            
            tokenString := parts[1]
            
            // Parse and validate JWT
            token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
                if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                    return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
                }
                return []byte(config.JWTSecretKey), nil
            })
            
            if err != nil || !token.Valid {
                http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
                return
            }
            
            // Extract claims
            claims, ok := token.Claims.(jwt.MapClaims)
            if !ok {
                http.Error(w, "Invalid token claims", http.StatusUnauthorized)
                return
            }
            
            // Extract user ID
            userIDStr, ok := claims["user_id"].(string)
            if !ok {
                http.Error(w, "User ID not found in token", http.StatusUnauthorized)
                return
            }
            
            userID, err := uuid.Parse(userIDStr)
            if err != nil {
                http.Error(w, "Invalid user ID format", http.StatusUnauthorized)
                return
            }
            
            // Store user ID in context
            ctx := context.WithValue(r.Context(), UserIDKey, userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// SubscriptionMiddleware checks user's subscription status for SEOSPS features
func SubscriptionMiddleware(sm *SubscriptionManager, pm *PlanManager, config *MiddlewareConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Skip subscription check for whitelisted paths
            for _, path := range config.WhitelistedPaths {
                if strings.HasPrefix(r.URL.Path, path) {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            
            // Get user ID from context (set by AuthMiddleware)
            ctx := r.Context()
            userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
            if !ok {
                http.Error(w, "User not authenticated", http.StatusUnauthorized)
                return
            }
            
            // Check if path requires subscription
            requiresSubscription := true
            for _, path := range config.WhitelistedPaths {
                if strings.HasPrefix(r.URL.Path, path) {
                    requiresSubscription = false
                    break
                }
            }
            
            if !requiresSubscription {
                next.ServeHTTP(w, r)
                return
            }
            
            // Get user's subscription
            subscription, err := sm.GetUserSubscription(userID)
            if err != nil {
                // No subscription found - check if payment is required
                if config.RequirePayment {
                    // Check if user is within free trial period
                    if config.FreeTrialDays > 0 {
                        // Get user creation date (you need to implement this)
                        // userCreatedAt := getUserCreatedAt(userID)
                        // trialEnd := userCreatedAt.AddDate(0, 0, config.FreeTrialDays)
                        
                        // if time.Now().Before(trialEnd) {
                        //     // User is in free trial, allow access
                        //     ctx = context.WithValue(ctx, HasAccessKey, true)
                        //     ctx = context.WithValue(ctx, SubscriptionKey, &Subscription{
                        //         Status: "trial",
                        //         TrialEnd: &trialEnd,
                        //     })
                        //     next.ServeHTTP(w, r.WithContext(ctx))
                        //     return
                        // }
                        
                        // Trial expired, require payment
                        http.Error(w, "Free trial expired. Please subscribe to continue using SEOSPS.", http.StatusPaymentRequired)
                        return
                    }
                    
                    // No free trial, require payment immediately
                    http.Error(w, "Subscription required to access SEOSPS features", http.StatusPaymentRequired)
                    return
                }
                
                // Payment not required, allow access
                ctx = context.WithValue(ctx, HasAccessKey, true)
                next.ServeHTTP(w, r.WithContext(ctx))
                return
            }
            
            // Check subscription status
            if !sm.isSubscriptionActive(subscription) {
                // Subscription is not active
                if config.GracePeriodDays > 0 {
                    // Check if within grace period
                    if subscription.CurrentEndDate != nil {
                        gracePeriodEnd := subscription.CurrentEndDate.AddDate(0, 0, config.GracePeriodDays)
                        if time.Now().Before(gracePeriodEnd) {
                            // In grace period, allow access but mark as limited
                            ctx = context.WithValue(ctx, HasAccessKey, true)
                            ctx = context.WithValue(ctx, SubscriptionKey, subscription)
                            ctx = context.WithValue(ctx, "grace_period", true)
                            next.ServeHTTP(w, r.WithContext(ctx))
                            return
                        }
                    }
                }
                
                http.Error(w, "Your subscription has expired. Please renew to continue using SEOSPS.", http.StatusPaymentRequired)
                return
            }
            
            // Get plan details
            plan, err := pm.GetPlanByID(subscription.PlanID)
            if err != nil {
                http.Error(w, "Failed to retrieve plan details", http.StatusInternalServerError)
                return
            }
            
            // Store subscription and plan in context
            ctx = context.WithValue(ctx, SubscriptionKey, subscription)
            ctx = context.WithValue(ctx, PlanKey, plan)
            ctx = context.WithValue(ctx, HasAccessKey, true)
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// PaymentRequiredMiddleware ensures user has made at least one successful payment
func PaymentRequiredMiddleware(pc *RazorpayClient, config *MiddlewareConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Skip payment check for whitelisted paths
            for _, path := range config.WhitelistedPaths {
                if strings.HasPrefix(r.URL.Path, path) {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            
            // Get user ID from context
            ctx := r.Context()
            userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
            if !ok {
                http.Error(w, "User not authenticated", http.StatusUnauthorized)
                return
            }
            
            // Check if user has any successful payments
            payments, err := pc.GetUserPayments(userID)
            if err != nil {
                http.Error(w, "Failed to retrieve payment history", http.StatusInternalServerError)
                return
            }
            
            hasSuccessfulPayment := false
            for _, payment := range payments {
                if payment.Status == "captured" && payment.Captured {
                    hasSuccessfulPayment = true
                    break
                }
            }
            
            if !hasSuccessfulPayment {
                // Check if user is within free trial period
                if config.FreeTrialDays > 0 {
                    // Similar free trial logic as in SubscriptionMiddleware
                    // ...
                    
                    // If not in free trial, require payment
                    http.Error(w, "Payment required to access this feature", http.StatusPaymentRequired)
                    return
                }
                
                http.Error(w, "Payment required to access this feature", http.StatusPaymentRequired)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// FeatureAccessMiddleware checks if user has access to specific SEOSPS features
func FeatureAccessMiddleware(sm *SubscriptionManager, feature string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get user ID from context
            ctx := r.Context()
            userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
            if !ok {
                http.Error(w, "User not authenticated", http.StatusUnauthorized)
                return
            }
            
            // Check if user has access to the requested feature
            hasAccess, err := sm.CheckSubscriptionAccess(userID, feature)
            if err != nil || !hasAccess {
                // Get user's plan to provide helpful error message
                subscription, _ := sm.GetUserSubscription(userID)
                var planName string
                if subscription != nil {
                    plan, _ := sm.planManager.GetPlanByID(subscription.PlanID)
                    if plan != nil {
                        planName = plan.Name
                    }
                }
                
                errorMsg := fmt.Sprintf(
                    "The '%s' feature is not available with your %s plan. "+
                    "Please upgrade your subscription to access this feature.",
                    feature, planName,
                )
                
                http.Error(w, errorMsg, http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// RateLimitMiddleware implements rate limiting based on subscription tier
func RateLimitMiddleware(sm *SubscriptionManager, limits map[string]int) func(http.Handler) http.Handler {
    // In-memory rate limiter (use Redis in production)
    rateLimiter := make(map[uuid.UUID]map[string]*RateLimitTracker)
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get user ID from context
            ctx := r.Context()
            userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
            if !ok {
                http.Error(w, "User not authenticated", http.StatusUnauthorized)
                return
            }
            
            // Get user's subscription
            subscription, err := sm.GetUserSubscription(userID)
            if err != nil {
                // No subscription, use free tier limits
                subscription = &Subscription{
                    Status: "free",
                }
            }
            
            // Determine rate limit based on subscription
            var limitKey string
            switch subscription.Status {
            case "active":
                // Get plan to determine tier
                if subscription.PlanID != uuid.Nil {
                    plan, err := sm.planManager.GetPlanByID(subscription.PlanID)
                    if err == nil {
                        limitKey = strings.ToLower(plan.Name)
                    }
                }
            case "trial":
                limitKey = "trial"
            default:
                limitKey = "free"
            }
            
            // Get limit for this tier
            limit, ok := limits[limitKey]
            if !ok {
                limit = limits["free"] // Default to free tier
            }
            
            // Initialize rate limiter for user if needed
            if _, userExists := rateLimiter[userID]; !userExists {
                rateLimiter[userID] = make(map[string]*RateLimitTracker)
            }
            
            // Get or create tracker for this endpoint
            endpoint := r.URL.Path
            tracker, exists := rateLimiter[userID][endpoint]
            if !exists {
                tracker = &RateLimitTracker{
                    Count:     0,
                    ResetTime: time.Now().Add(time.Hour),
                }
                rateLimiter[userID][endpoint] = tracker
            }
            
            // Check if reset time has passed
            if time.Now().After(tracker.ResetTime) {
                tracker.Count = 0
                tracker.ResetTime = time.Now().Add(time.Hour)
            }
            
            // Check if limit exceeded
            if tracker.Count >= limit {
                w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
                w.Header().Set("X-RateLimit-Remaining", "0")
                w.Header().Set("X-RateLimit-Reset", tracker.ResetTime.Format(time.RFC3339))
                
                http.Error(w, 
                    fmt.Sprintf("Rate limit exceeded. %d requests per hour allowed. Please upgrade your plan for higher limits.", limit),
                    http.StatusTooManyRequests,
                )
                return
            }
            
            // Increment count
            tracker.Count++
            
            // Set rate limit headers
            w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
            w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", limit-tracker.Count))
            w.Header().Set("X-RateLimit-Reset", tracker.ResetTime.Format(time.RFC3339))
            
            next.ServeHTTP(w, r)
        })
    }
}

// RateLimitTracker tracks rate limiting for a user
type RateLimitTracker struct {
    Count     int
    ResetTime time.Time
}

// WebsiteLimitMiddleware checks if user has reached website limit for their plan
func WebsiteLimitMiddleware(sm *SubscriptionManager) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Only check for website-related routes
            if !strings.Contains(r.URL.Path, "/websites") && 
               !strings.Contains(r.URL.Path, "/scan") &&
               r.Method != "POST" && r.Method != "PUT" {
                next.ServeHTTP(w, r)
                return
            }
            
            // Get user ID from context
            ctx := r.Context()
            userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
            if !ok {
                http.Error(w, "User not authenticated", http.StatusUnauthorized)
                return
            }
            
            // Get user's subscription
            subscription, err := sm.GetUserSubscription(userID)
            if err != nil {
                http.Error(w, "No active subscription found", http.StatusPaymentRequired)
                return
            }
            
            // Get plan details
            plan, err := sm.planManager.GetPlanByID(subscription.PlanID)
            if err != nil {
                http.Error(w, "Failed to retrieve plan details", http.StatusInternalServerError)
                return
            }
            
            // Check if plan has website limit
            if plan.WebsiteLimit > 0 {
                // Get current website count for user (implement this based on your system)
                currentWebsiteCount := getWebsiteCount(userID)
                
                if currentWebsiteCount >= plan.WebsiteLimit {
                    errorMsg := fmt.Sprintf(
                        "You have reached the limit of %d websites for your %s plan. "+
                        "Please upgrade to a higher plan or remove some websites.",
                        plan.WebsiteLimit, plan.Name,
                    )
                    
                    http.Error(w, errorMsg, http.StatusForbidden)
                    return
                }
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// ScanLimitMiddleware checks if user has reached scan limit for their plan
func ScanLimitMiddleware(sm *SubscriptionManager, db *gorm.DB) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Only check for scan-related routes
            if !strings.Contains(r.URL.Path, "/scan") && r.Method != "POST" {
                next.ServeHTTP(w, r)
                return
            }
            
            // Get user ID from context
            ctx := r.Context()
            userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
            if !ok {
                http.Error(w, "User not authenticated", http.StatusUnauthorized)
                return
            }
            
            // Get user's subscription
            subscription, err := sm.GetUserSubscription(userID)
            if err != nil {
                http.Error(w, "No active subscription found", http.StatusPaymentRequired)
                return
            }
            
            // Get plan details
            plan, err := sm.planManager.GetPlanByID(subscription.PlanID)
            if err != nil {
                http.Error(w, "Failed to retrieve plan details", http.StatusInternalServerError)
                return
            }
            
            // Check if plan has scan limit - using correct field name
            // Assuming the field is named 'MonthlyScanLimit' or similar - adjust as needed
            scanLimit := plan.GetScanLimit() // Using a method if available, or direct field
            
            if scanLimit > 0 {
                // Get scans count for current month
                startOfMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
                endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond)
                
                // Use endOfMonth to avoid unused variable warning
                _ = endOfMonth
                
                var scanCount int64
                // Uncomment and adjust this query based on your schema
                // err = db.Model(&Scan{}).
                //     Where("user_id = ? AND created_at BETWEEN ? AND ?", userID, startOfMonth, endOfMonth).
                //     Count(&scanCount).Error
                // if err != nil {
                //     http.Error(w, "Failed to check scan count", http.StatusInternalServerError)
                //     return
                // }
                
                // For now, use placeholder (replace with actual DB query when ready)
                scanCount = 0
                
                if scanCount >= int64(scanLimit) {
                    errorMsg := fmt.Sprintf(
                        "You have reached your monthly scan limit of %d for your %s plan. "+
                        "The limit will reset on the 1st of next month, or you can upgrade your plan.",
                        scanLimit, plan.Name,
                    )
                    
                    http.Error(w, errorMsg, http.StatusForbidden)
                    return
                }
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// AdminMiddleware restricts access to admin users only
func AdminMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get user ID from context
            ctx := r.Context()
            userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
            if !ok {
                http.Error(w, "User not authenticated", http.StatusUnauthorized)
                return
            }
            
            // Check if user is admin (implement based on your user system)
            isAdmin := isUserAdmin(userID)
            if !isAdmin {
                http.Error(w, "Admin access required", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// PaymentWebhookMiddleware validates Razorpay webhook signatures
func PaymentWebhookMiddleware(wh *WebhookHandler) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Only apply to webhook endpoints
            if !strings.Contains(r.URL.Path, "/webhooks/") {
                next.ServeHTTP(w, r)
                return
            }
            
            // Verify webhook signature
            body, err := io.ReadAll(r.Body)
            if err != nil {
                http.Error(w, "Failed to read request body", http.StatusBadRequest)
                return
            }
            
            // Restore body for handler
            r.Body = io.NopCloser(bytes.NewBuffer(body))
            
            signature := r.Header.Get("X-Razorpay-Signature")
            if !wh.VerifySignature(signature, body) {
                http.Error(w, "Invalid webhook signature", http.StatusUnauthorized)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// Helper functions (implement based on your system)

// getWebsiteCount returns the number of websites a user has
func getWebsiteCount(userID uuid.UUID) int {
    // Implement based on your database schema
    // Example: SELECT COUNT(*) FROM websites WHERE user_id = ?
    return 0
}

// isUserAdmin checks if a user has admin privileges
func isUserAdmin(userID uuid.UUID) bool {
    // Implement based on your user system
    // Example: SELECT is_admin FROM users WHERE id = ?
    return false
}

// getUserCreatedAt gets the user's account creation date
func getUserCreatedAt(userID uuid.UUID) time.Time {
    // Implement based on your user system
    // Example: SELECT created_at FROM users WHERE id = ?
    return time.Now()
}

// Context helper functions to extract values from request context

// GetUserIDFromContext extracts user ID from request context
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
    userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
    return userID, ok
}

// GetSubscriptionFromContext extracts subscription from request context
func GetSubscriptionFromContext(ctx context.Context) (*Subscription, bool) {
    subscription, ok := ctx.Value(SubscriptionKey).(*Subscription)
    return subscription, ok
}

// GetPlanFromContext extracts plan from request context
func GetPlanFromContext(ctx context.Context) (*PaymentPlan, bool) {
    plan, ok := ctx.Value(PlanKey).(*PaymentPlan)
    return plan, ok
}

// HasAccessFromContext checks if user has access from request context
func HasAccessFromContext(ctx context.Context) bool {
    hasAccess, ok := ctx.Value(HasAccessKey).(bool)
    return ok && hasAccess
}

// IsInGracePeriodFromContext checks if user is in grace period
func IsInGracePeriodFromContext(ctx context.Context) bool {
    inGracePeriod, ok := ctx.Value("grace_period").(bool)
    return ok && inGracePeriod
}

// Add this helper method to PaymentPlan if it doesn't exist
// This assumes you have a field like MonthlyScanLimit or similar
func (p *PaymentPlan) GetScanLimit() int {
    // Replace with actual field name from your PaymentPlan struct
    // For example: return p.MonthlyScanLimit
    // If the field doesn't exist, you might need to add it to your struct
    return 0
}

// Example usage in router setup - commented out to avoid undefined handler errors
/*
func SetupPaymentMiddleware(router *http.ServeMux, db *gorm.DB, config *MiddlewareConfig) {
    // Initialize payment services
    razorpayConfig := &RazorpayConfig{
        KeyID:        "your_key_id",
        KeySecret:    "your_key_secret",
        WebhookSecret: "your_webhook_secret",
        BaseURL:      "https://api.razorpay.com/v1",
    }
    
    razorpayClient := NewRazorpayClient(razorpayConfig, db)
    planManager := NewPlanManager(db, razorpayClient)
    
    // Fix: NewSubscriptionManager should only take the required parameters
    // Adjust based on actual NewSubscriptionManager implementation
    subscriptionManager := NewSubscriptionManager(db, planManager) // Removed razorpayClient if not needed
    
    webhookHandler := NewWebhookHandler(razorpayConfig, db, razorpayClient)
    
    // Define rate limits per subscription tier
    rateLimits := map[string]int{
        "free":     100,  // 100 requests/hour
        "trial":    500,  // 500 requests/hour
        "starter":  1000, // 1000 requests/hour
        "gold":     5000, // 5000 requests/hour
        "platinum": 10000, // 10000 requests/hour
        "diamond":  50000, // 50000 requests/hour
    }
    
    // Define whitelisted paths (no auth/payment required)
    config.WhitelistedPaths = []string{
        "/api/health",
        "/api/auth/login",
        "/api/auth/register",
        "/api/pricing",
        "/webhooks/razorpay",
    }
    
    // Apply middleware chain
    router.Handle("/api/", 
        AuthMiddleware(config)(
        SubscriptionMiddleware(subscriptionManager, planManager, config)(
        RateLimitMiddleware(subscriptionManager, rateLimits)(
        WebsiteLimitMiddleware(subscriptionManager)(
        http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Your API handler implementation
            w.Write([]byte("API endpoint"))
        }))))))
    
    // Apply feature-specific middleware
    router.Handle("/api/ai-automation/",
        AuthMiddleware(config)(
        FeatureAccessMiddleware(subscriptionManager, "ai_automation")(
        http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("AI Automation endpoint"))
        }))))
    
    router.Handle("/api/priority-support/",
        AuthMiddleware(config)(
        FeatureAccessMiddleware(subscriptionManager, "priority_support")(
        http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("Priority Support endpoint"))
        }))))
    
    // Apply admin middleware
    router.Handle("/api/admin/",
        AuthMiddleware(config)(
        AdminMiddleware()(
        http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("Admin endpoint"))
        }))))
    
    // Apply payment webhook middleware
    router.Handle("/webhooks/razorpay",
        PaymentWebhookMiddleware(webhookHandler)(
        http.HandlerFunc(webhookHandler.HandleWebhook)))
    
    // Apply scan limit middleware to scan endpoints
    router.Handle("/api/scan/",
        AuthMiddleware(config)(
        ScanLimitMiddleware(subscriptionManager, db)(
        http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("Scan endpoint"))
        }))))
}
*/