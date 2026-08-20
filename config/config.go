package config

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    "strings"
    "time"

    "gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
    App          AppConfig          `yaml:"app" json:"app"`
    Database     DatabaseConfig     `yaml:"database" json:"database"`
    Redis        RedisConfig        `yaml:"redis" json:"redis"`
    Queue        QueueConfig        `yaml:"queue" json:"queue"`
    Cache        CacheConfig        `yaml:"cache" json:"cache"`
    Email        EmailConfig        `yaml:"email" json:"email"`
    JWT          JWTConfig          `yaml:"jwt" json:"jwt"`
    APIKeys      APIKeysConfig      `yaml:"api_keys" json:"api_keys"`
    RateLimit    RateLimitConfig    `yaml:"rate_limit" json:"rate_limit"`
    SEO          SEOConfig          `yaml:"seo" json:"seo"`
    Plans        PlansConfig        `yaml:"plans" json:"plans"`
    Monitoring   MonitoringConfig   `yaml:"monitoring" json:"monitoring"`
    Logging      LoggingConfig      `yaml:"logging" json:"logging"`
    CORS         CORSConfig         `yaml:"cors" json:"cors"`
    Security     SecurityConfig     `yaml:"security" json:"security"`
    Uploads      UploadsConfig      `yaml:"uploads" json:"uploads"`
    Notifications NotificationsConfig `yaml:"notifications" json:"notifications"`
    Google       GoogleConfig       `yaml:"google" json:"google"`
}

// AppConfig holds application settings
type AppConfig struct {
    Name        string `yaml:"name" json:"name"`
    Version     string `yaml:"version" json:"version"`
    Environment string `yaml:"environment" json:"environment"`
    Debug       bool   `yaml:"debug" json:"debug"`
    Port        int    `yaml:"port" json:"port"`
    Host        string `yaml:"host" json:"host"`
}

// DatabaseConfig holds database settings
type DatabaseConfig struct {
    Driver          string        `yaml:"driver" json:"driver"`
    Path            string        `yaml:"path" json:"path"`
    Host            string        `yaml:"host" json:"host"`
    Port            int           `yaml:"port" json:"port"`
    User            string        `yaml:"user" json:"user"`
    Password        string        `yaml:"password" json:"-"`
    DBName          string        `yaml:"dbname" json:"dbname"`
    SSLMode         string        `yaml:"sslmode" json:"sslmode"`
    MaxOpenConns    int           `yaml:"max_open_conns" json:"max_open_conns"`
    MaxIdleConns    int           `yaml:"max_idle_conns" json:"max_idle_conns"`
    ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
}

// RedisConfig holds Redis cache settings
type RedisConfig struct {
    Enabled       bool          `yaml:"enabled" json:"enabled"`
    Host          string        `yaml:"host" json:"host"`
    Port          int           `yaml:"port" json:"port"`
    Password      string        `yaml:"password" json:"-"`
    DB            int           `yaml:"db" json:"db"`
    MaxRetries    int           `yaml:"max_retries" json:"max_retries"`
    PoolSize      int           `yaml:"pool_size" json:"pool_size"`
    MinIdleConns  int           `yaml:"min_idle_conns" json:"min_idle_conns"`
    DialTimeout   time.Duration `yaml:"dial_timeout" json:"dial_timeout"`
    ReadTimeout   time.Duration `yaml:"read_timeout" json:"read_timeout"`
    WriteTimeout  time.Duration `yaml:"write_timeout" json:"write_timeout"`
}

// QueueConfig holds queue settings
type QueueConfig struct {
    Workers       int           `yaml:"workers" json:"workers"`
    MaxQueueSize  int           `yaml:"max_queue_size" json:"max_queue_size"`
    JobTimeout    time.Duration `yaml:"job_timeout" json:"job_timeout"`
    RetryAttempts int           `yaml:"retry_attempts" json:"retry_attempts"`
    RetryDelay    time.Duration `yaml:"retry_delay" json:"retry_delay"`
}

// CacheConfig holds cache settings
type CacheConfig struct {
    DefaultTTL      time.Duration `yaml:"default_ttl" json:"default_ttl"`
    MaxSize         int           `yaml:"max_size" json:"max_size"`
    CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`
}

// EmailConfig holds email settings
type EmailConfig struct {
    Enabled      bool   `yaml:"enabled" json:"enabled"`
    Host         string `yaml:"host" json:"host"`
    Port         int    `yaml:"port" json:"port"`
    Username     string `yaml:"username" json:"username"`
    Password     string `yaml:"password" json:"-"`
    From         string `yaml:"from" json:"from"`
    TemplatesPath string `yaml:"templates_path" json:"templates_path"`
}

// JWTConfig holds JWT settings
type JWTConfig struct {
    Secret             string        `yaml:"secret" json:"-"`
    AccessTokenExpiry  time.Duration `yaml:"access_token_expiry" json:"access_token_expiry"`
    RefreshTokenExpiry time.Duration `yaml:"refresh_token_expiry" json:"refresh_token_expiry"`
    Issuer             string        `yaml:"issuer" json:"issuer"`
}

// APIKeysConfig holds external API keys
type APIKeysConfig struct {
    GoogleAds string `yaml:"google_ads" json:"-"`
    Ahrefs    string `yaml:"ahrefs" json:"-"`
    Semrush   string `yaml:"semrush" json:"-"`
    OpenAI    string `yaml:"openai" json:"-"`
}

// ✅ Google OAuth Config
type GoogleConfig struct {
    ClientID     string   `yaml:"client_id" json:"client_id"`
    ClientSecret string   `yaml:"client_secret" json:"-"`
    RedirectURL  string   `yaml:"redirect_url" json:"redirect_url"`
    Scopes       []string `yaml:"scopes" json:"scopes"`
    Enabled      bool     `yaml:"enabled" json:"enabled"`
}

// RateLimitConfig holds rate limiting settings
type RateLimitConfig struct {
    Enabled           bool    `yaml:"enabled" json:"enabled"`
    RequestsPerSecond float64 `yaml:"requests_per_second" json:"requests_per_second"`
    Burst             int     `yaml:"burst" json:"burst"`
    MaxRequestsPerDay int     `yaml:"max_requests_per_day" json:"max_requests_per_day"`
}

// SEOConfig holds SEO analysis settings
type SEOConfig struct {
    MinKeywordLength      int           `yaml:"min_keyword_length" json:"min_keyword_length"`
    MaxKeywordsPerPage    int           `yaml:"max_keywords_per_page" json:"max_keywords_per_page"`
    KeywordDensityMin     float64       `yaml:"keyword_density_min" json:"keyword_density_min"`
    KeywordDensityMax     float64       `yaml:"keyword_density_max" json:"keyword_density_max"`
    CompetitorAnalysisDepth int         `yaml:"competitor_analysis_depth" json:"competitor_analysis_depth"`
    CacheResults          bool          `yaml:"cache_results" json:"cache_results"`
    ResultCacheTTL        time.Duration `yaml:"result_cache_ttl" json:"result_cache_ttl"`
}

// PlanConfig holds individual plan settings
type PlanConfig struct {
    Name     string   `yaml:"name" json:"name"`
    Price    float64  `yaml:"price" json:"price"`
    Credits  int      `yaml:"credits" json:"credits"`
    Features []string `yaml:"features" json:"features"`
}

// PlansConfig holds all pricing plans
type PlansConfig struct {
    Free       PlanConfig `yaml:"free" json:"free"`
    Basic      PlanConfig `yaml:"basic" json:"basic"`
    Pro        PlanConfig `yaml:"pro" json:"pro"`
    Enterprise PlanConfig `yaml:"enterprise" json:"enterprise"`
}

// MonitoringConfig holds monitoring settings
type MonitoringConfig struct {
    SentryDSN        string `yaml:"sentry_dsn" json:"-"`
    PrometheusEnabled bool   `yaml:"prometheus_enabled" json:"prometheus_enabled"`
    MetricsPath      string `yaml:"metrics_path" json:"metrics_path"`
    HealthCheckPath  string `yaml:"health_check_path" json:"health_check_path"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
    Level      string `yaml:"level" json:"level"`
    Format     string `yaml:"format" json:"format"`
    Output     string `yaml:"output" json:"output"`
    FilePath   string `yaml:"file_path" json:"file_path"`
    MaxSizeMB  int    `yaml:"max_size_mb" json:"max_size_mb"`
    MaxAgeDays int    `yaml:"max_age_days" json:"max_age_days"`
    MaxBackups int    `yaml:"max_backups" json:"max_backups"`
    Compress   bool   `yaml:"compress" json:"compress"`
}

// CORSConfig holds CORS settings
type CORSConfig struct {
    Enabled          bool     `yaml:"enabled" json:"enabled"`
    AllowedOrigins   []string `yaml:"allowed_origins" json:"allowed_origins"`
    AllowedMethods   []string `yaml:"allowed_methods" json:"allowed_methods"`
    AllowedHeaders   []string `yaml:"allowed_headers" json:"allowed_headers"`
    AllowCredentials bool     `yaml:"allow_credentials" json:"allow_credentials"`
    MaxAge           int      `yaml:"max_age" json:"max_age"`
}

// SecurityConfig holds security settings
type SecurityConfig struct {
    BcryptCost       int           `yaml:"bcrypt_cost" json:"bcrypt_cost"`
    MinPasswordLength int          `yaml:"min_password_length" json:"min_password_length"`
    MaxLoginAttempts int           `yaml:"max_login_attempts" json:"max_login_attempts"`
    LockoutDuration  time.Duration `yaml:"lockout_duration" json:"lockout_duration"`
    SessionTimeout   time.Duration `yaml:"session_timeout" json:"session_timeout"`
    TLS              TLSConfig     `yaml:"tls" json:"tls"`
}

// TLSConfig holds TLS settings
type TLSConfig struct {
    Enabled  bool   `yaml:"enabled" json:"enabled"`
    CertFile string `yaml:"cert_file" json:"cert_file"`
    KeyFile  string `yaml:"key_file" json:"key_file"`
}

// UploadsConfig holds file upload settings
type UploadsConfig struct {
    MaxSizeMB    int      `yaml:"max_size_mb" json:"max_size_mb"`
    AllowedTypes []string `yaml:"allowed_types" json:"allowed_types"`
    UploadPath   string   `yaml:"upload_path" json:"upload_path"`
    TempPath     string   `yaml:"temp_path" json:"temp_path"`
}

// NotificationsConfig holds notification settings
type NotificationsConfig struct {
    Enabled      bool     `yaml:"enabled" json:"enabled"`
    SlackWebhook string   `yaml:"slack_webhook" json:"-"`
    DiscordWebhook string `yaml:"discord_webhook" json:"-"`
    EmailAlerts  EmailAlertsConfig `yaml:"email_alerts" json:"email_alerts"`
}

// EmailAlertsConfig holds email alert settings
type EmailAlertsConfig struct {
    OnError    bool     `yaml:"on_error" json:"on_error"`
    OnComplete bool     `yaml:"on_complete" json:"on_complete"`
    Recipients []string `yaml:"recipients" json:"recipients"`
}

// ConfigLoader handles loading and parsing configuration
type ConfigLoader struct {
    configPath string
    config     *Config
}

// NewConfigLoader creates a new config loader
func NewConfigLoader(configPath string) *ConfigLoader {
    return &ConfigLoader{
        configPath: configPath,
        config:     &Config{},
    }
}

// Load loads and parses the configuration file
func (l *ConfigLoader) Load() (*Config, error) {
    // Read config file
    data, err := ioutil.ReadFile(l.configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    // Parse YAML
    err = yaml.Unmarshal(data, l.config)
    if err != nil {
        return nil, fmt.Errorf("failed to parse config file: %w", err)
    }

    // Replace environment variables
    l.replaceEnvVars()

    // Set defaults for missing values
    l.setDefaults()

    // Validate config
    if err := l.validate(); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }

    return l.config, nil
}

// replaceEnvVars replaces ${VAR} placeholders with environment variables
func (l *ConfigLoader) replaceEnvVars() {
    // Database password
    l.config.Database.Password = l.replaceEnvVar(l.config.Database.Password)
    
    // Redis password
    l.config.Redis.Password = l.replaceEnvVar(l.config.Redis.Password)
    
    // Email password
    l.config.Email.Password = l.replaceEnvVar(l.config.Email.Password)
    
    // JWT secret
    l.config.JWT.Secret = l.replaceEnvVar(l.config.JWT.Secret)
    
    // API keys
    l.config.APIKeys.GoogleAds = l.replaceEnvVar(l.config.APIKeys.GoogleAds)
    l.config.APIKeys.Ahrefs = l.replaceEnvVar(l.config.APIKeys.Ahrefs)
    l.config.APIKeys.Semrush = l.replaceEnvVar(l.config.APIKeys.Semrush)
    l.config.APIKeys.OpenAI = l.replaceEnvVar(l.config.APIKeys.OpenAI)
    
    // Monitoring
    l.config.Monitoring.SentryDSN = l.replaceEnvVar(l.config.Monitoring.SentryDSN)
    
    // Notifications
    l.config.Notifications.SlackWebhook = l.replaceEnvVar(l.config.Notifications.SlackWebhook)
    l.config.Notifications.DiscordWebhook = l.replaceEnvVar(l.config.Notifications.DiscordWebhook)
    
    // ✅ Google OAuth - Read from environment variables
    l.config.Google.ClientID = l.replaceEnvVar(l.config.Google.ClientID)
    l.config.Google.ClientSecret = l.replaceEnvVar(l.config.Google.ClientSecret)
    l.config.Google.RedirectURL = l.replaceEnvVar(l.config.Google.RedirectURL)
}

// replaceEnvVar replaces ${VAR} with environment variable
func (l *ConfigLoader) replaceEnvVar(value string) string {
    if value == "" {
        return value
    }

    // Check if value contains ${VAR}
    if strings.Contains(value, "${") && strings.Contains(value, "}") {
        start := strings.Index(value, "${") + 2
        end := strings.Index(value, "}")
        if start > 1 && end > start {
            envVar := value[start:end]
            envValue := os.Getenv(envVar)
            if envValue != "" {
                return strings.Replace(value, "${"+envVar+"}", envValue, -1)
            }
        }
    }

    return value
}

// setDefaults sets default values for missing config fields
func (l *ConfigLoader) setDefaults() {
    // App defaults
    if l.config.App.Name == "" {
        l.config.App.Name = "AI SEO Tool"
    }
    if l.config.App.Version == "" {
        l.config.App.Version = "2.0.0"
    }
    if l.config.App.Environment == "" {
        l.config.App.Environment = "development"
    }
    if l.config.App.Port == 0 {
        l.config.App.Port = 8080
    }
    if l.config.App.Host == "" {
        l.config.App.Host = "0.0.0.0"
    }

    // Database defaults
    if l.config.Database.Driver == "" {
        l.config.Database.Driver = "sqlite3"
    }
    if l.config.Database.Path == "" {
        l.config.Database.Path = "data/seo.db"
    }
    if l.config.Database.MaxOpenConns == 0 {
        l.config.Database.MaxOpenConns = 25
    }
    if l.config.Database.MaxIdleConns == 0 {
        l.config.Database.MaxIdleConns = 5
    }
    if l.config.Database.ConnMaxLifetime == 0 {
        l.config.Database.ConnMaxLifetime = 5 * time.Minute
    }

    // Redis defaults
    if l.config.Redis.Host == "" {
        l.config.Redis.Host = "localhost"
    }
    if l.config.Redis.Port == 0 {
        l.config.Redis.Port = 6379
    }
    if l.config.Redis.DB == 0 {
        l.config.Redis.DB = 0
    }
    if l.config.Redis.MaxRetries == 0 {
        l.config.Redis.MaxRetries = 3
    }
    if l.config.Redis.PoolSize == 0 {
        l.config.Redis.PoolSize = 10
    }
    if l.config.Redis.DialTimeout == 0 {
        l.config.Redis.DialTimeout = 5 * time.Second
    }
    if l.config.Redis.ReadTimeout == 0 {
        l.config.Redis.ReadTimeout = 3 * time.Second
    }
    if l.config.Redis.WriteTimeout == 0 {
        l.config.Redis.WriteTimeout = 3 * time.Second
    }

    // Queue defaults
    if l.config.Queue.Workers == 0 {
        l.config.Queue.Workers = 10
    }
    if l.config.Queue.MaxQueueSize == 0 {
        l.config.Queue.MaxQueueSize = 1000
    }
    if l.config.Queue.JobTimeout == 0 {
        l.config.Queue.JobTimeout = 5 * time.Minute
    }
    if l.config.Queue.RetryAttempts == 0 {
        l.config.Queue.RetryAttempts = 3
    }
    if l.config.Queue.RetryDelay == 0 {
        l.config.Queue.RetryDelay = 30 * time.Second
    }

    // Cache defaults
    if l.config.Cache.DefaultTTL == 0 {
        l.config.Cache.DefaultTTL = 10 * time.Minute
    }
    if l.config.Cache.MaxSize == 0 {
        l.config.Cache.MaxSize = 10000
    }
    if l.config.Cache.CleanupInterval == 0 {
        l.config.Cache.CleanupInterval = 5 * time.Minute
    }

    // JWT defaults
    if l.config.JWT.AccessTokenExpiry == 0 {
        l.config.JWT.AccessTokenExpiry = 24 * time.Hour
    }
    if l.config.JWT.RefreshTokenExpiry == 0 {
        l.config.JWT.RefreshTokenExpiry = 720 * time.Hour
    }
    if l.config.JWT.Issuer == "" {
        l.config.JWT.Issuer = "seosps"
    }

    // Rate limit defaults
    if l.config.RateLimit.RequestsPerSecond == 0 {
        l.config.RateLimit.RequestsPerSecond = 10
    }
    if l.config.RateLimit.Burst == 0 {
        l.config.RateLimit.Burst = 20
    }
    if l.config.RateLimit.MaxRequestsPerDay == 0 {
        l.config.RateLimit.MaxRequestsPerDay = 10000
    }

    // SEO defaults
    if l.config.SEO.MinKeywordLength == 0 {
        l.config.SEO.MinKeywordLength = 3
    }
    if l.config.SEO.MaxKeywordsPerPage == 0 {
        l.config.SEO.MaxKeywordsPerPage = 20
    }
    if l.config.SEO.KeywordDensityMin == 0 {
        l.config.SEO.KeywordDensityMin = 1.0
    }
    if l.config.SEO.KeywordDensityMax == 0 {
        l.config.SEO.KeywordDensityMax = 2.5
    }
    if l.config.SEO.CompetitorAnalysisDepth == 0 {
        l.config.SEO.CompetitorAnalysisDepth = 10
    }
    if l.config.SEO.ResultCacheTTL == 0 {
        l.config.SEO.ResultCacheTTL = 24 * time.Hour
    }

    // Logging defaults
    if l.config.Logging.Level == "" {
        l.config.Logging.Level = "info"
    }
    if l.config.Logging.Format == "" {
        l.config.Logging.Format = "json"
    }
    if l.config.Logging.Output == "" {
        l.config.Logging.Output = "stdout"
    }
    if l.config.Logging.MaxSizeMB == 0 {
        l.config.Logging.MaxSizeMB = 100
    }
    if l.config.Logging.MaxAgeDays == 0 {
        l.config.Logging.MaxAgeDays = 30
    }
    if l.config.Logging.MaxBackups == 0 {
        l.config.Logging.MaxBackups = 5
    }

    // Security defaults
    if l.config.Security.BcryptCost == 0 {
        l.config.Security.BcryptCost = 10
    }
    if l.config.Security.MinPasswordLength == 0 {
        l.config.Security.MinPasswordLength = 8
    }
    if l.config.Security.MaxLoginAttempts == 0 {
        l.config.Security.MaxLoginAttempts = 5
    }
    if l.config.Security.LockoutDuration == 0 {
        l.config.Security.LockoutDuration = 15 * time.Minute
    }
    if l.config.Security.SessionTimeout == 0 {
        l.config.Security.SessionTimeout = 2 * time.Hour
    }

    // Uploads defaults
    if l.config.Uploads.MaxSizeMB == 0 {
        l.config.Uploads.MaxSizeMB = 10
    }
    if l.config.Uploads.UploadPath == "" {
        l.config.Uploads.UploadPath = "uploads/"
    }
    if l.config.Uploads.TempPath == "" {
        l.config.Uploads.TempPath = "tmp/"
    }

    // ✅ Google OAuth defaults
    if l.config.Google.Scopes == nil || len(l.config.Google.Scopes) == 0 {
        l.config.Google.Scopes = []string{"email", "profile"}
    }
    
    // ✅ Set redirect URL with fallback
    if l.config.Google.RedirectURL == "" {
        // Check if running in production
        if os.Getenv("ENVIRONMENT") == "production" {
            // Use custom domain or Render domain
            backendURL := os.Getenv("BACKEND_URL")
            if backendURL != "" {
                l.config.Google.RedirectURL = backendURL + "/api/auth/google/callback"
            } else {
                l.config.Google.RedirectURL = "https://ai-seo-backend-q95e.onrender.com/api/auth/google/callback"
            }
        } else {
            l.config.Google.RedirectURL = "http://localhost:8080/api/auth/google/callback"
        }
    }
    
    // Enabled defaults to true if client_id is set
    if l.config.Google.ClientID != "" && !l.config.Google.Enabled {
        l.config.Google.Enabled = true
    }
}

// validate validates the configuration
func (l *ConfigLoader) validate() error {
    // Validate environment
    validEnvs := map[string]bool{
        "development": true,
        "staging":     true,
        "production":  true,
    }
    if !validEnvs[l.config.App.Environment] {
        return fmt.Errorf("invalid environment: %s", l.config.App.Environment)
    }

    // Validate port
    if l.config.App.Port < 1 || l.config.App.Port > 65535 {
        return fmt.Errorf("invalid port: %d", l.config.App.Port)
    }

    // Validate database driver
    validDrivers := map[string]bool{
        "sqlite3":  true,
        "postgres": true,
        "mysql":    true,
    }
    if !validDrivers[l.config.Database.Driver] {
        return fmt.Errorf("invalid database driver: %s", l.config.Database.Driver)
    }

    // Validate paths
    if l.config.Database.Driver == "sqlite3" && l.config.Database.Path == "" {
        return fmt.Errorf("database path is required for sqlite3")
    }

    // Validate required secrets in production
    if l.config.App.Environment == "production" {
        if l.config.JWT.Secret == "" {
            return fmt.Errorf("JWT secret is required in production")
        }
        if len(l.config.JWT.Secret) < 32 {
            return fmt.Errorf("JWT secret must be at least 32 characters in production")
        }
    }

    // Validate email config if enabled
    if l.config.Email.Enabled {
        if l.config.Email.Host == "" {
            return fmt.Errorf("email host is required when email is enabled")
        }
        if l.config.Email.Port == 0 {
            return fmt.Errorf("email port is required when email is enabled")
        }
        if l.config.Email.Username == "" {
            return fmt.Errorf("email username is required when email is enabled")
        }
        if l.config.Email.Password == "" {
            return fmt.Errorf("email password is required when email is enabled")
        }
    }

    // Validate Redis config if enabled
    if l.config.Redis.Enabled {
        if l.config.Redis.Host == "" {
            return fmt.Errorf("redis host is required when redis is enabled")
        }
        if l.config.Redis.Port == 0 {
            return fmt.Errorf("redis port is required when redis is enabled")
        }
    }

    // ✅ Validate Google OAuth config if enabled
    if l.config.Google.Enabled {
        if l.config.Google.ClientID == "" {
            return fmt.Errorf("Google client ID is required when Google OAuth is enabled")
        }
        if l.config.Google.ClientSecret == "" {
            return fmt.Errorf("Google client secret is required when Google OAuth is enabled")
        }
        if l.config.Google.RedirectURL == "" {
            return fmt.Errorf("Google redirect URL is required when Google OAuth is enabled")
        }
    }

    return nil
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
    switch c.Driver {
    case "sqlite3":
        return c.Path
    case "postgres":
        return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
            c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
    case "mysql":
        return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
            c.User, c.Password, c.Host, c.Port, c.DBName)
    default:
        return c.Path
    }
}

// GetRedisAddr returns the Redis address
func (c *RedisConfig) GetRedisAddr() string {
    return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsDevelopment returns true if environment is development
func (c *AppConfig) IsDevelopment() bool {
    return c.Environment == "development"
}

// IsProduction returns true if environment is production
func (c *AppConfig) IsProduction() bool {
    return c.Environment == "production"
}

// IsStaging returns true if environment is staging
func (c *AppConfig) IsStaging() bool {
    return c.Environment == "staging"
}

// ✅ IsGoogleEnabled returns true if Google OAuth is enabled
func (c *GoogleConfig) IsEnabled() bool {
    return c.Enabled && c.ClientID != "" && c.ClientSecret != ""
}

// ✅ GetRedirectURL returns the Google OAuth redirect URL
func (c *GoogleConfig) GetRedirectURL() string {
    if c.RedirectURL == "" {
        return "http://localhost:8080/api/auth/google/callback"
    }
    return c.RedirectURL
}

// ✅ GetOAuthConfig returns the Google OAuth config for use in handlers
func (c *GoogleConfig) GetOAuthConfig() *OAuthConfig {
    return &OAuthConfig{
        ClientID:     c.ClientID,
        ClientSecret: c.ClientSecret,
        RedirectURL:  c.RedirectURL,
        Scopes:       c.Scopes,
        Enabled:      c.Enabled,
    }
}

// ✅ OAuthConfig is a helper struct for OAuth configuration
type OAuthConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURL  string
    Scopes       []string
    Enabled      bool
}

// ToJSON returns the config as JSON
func (c *Config) ToJSON() (string, error) {
    data, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return "", err
    }
    return string(data), nil
}

// Save saves the config to a file
func (c *Config) Save(path string) error {
    data, err := yaml.Marshal(c)
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    err = ioutil.WriteFile(path, data, 0644)
    if err != nil {
        return fmt.Errorf("failed to write config file: %w", err)
    }

    return nil
}

// ✅ Helper function to load Google OAuth config from environment directly
func LoadGoogleOAuthConfig() *OAuthConfig {
    redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
    if redirectURL == "" {
        // Check if using custom domain
        backendURL := os.Getenv("BACKEND_URL")
        if backendURL != "" {
            redirectURL = backendURL + "/api/auth/google/callback"
        } else if os.Getenv("ENVIRONMENT") == "production" {
            // ✅ REPLACE with your custom domain
            redirectURL = "https://api.seosps.com/api/auth/google/callback"
        } else {
            redirectURL = "http://localhost:8080/api/auth/google/callback"
        }
    }

    return &OAuthConfig{
        ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
        ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
        RedirectURL:  redirectURL,
        Scopes:       []string{"email", "profile"},
        Enabled:      true,
    }
}