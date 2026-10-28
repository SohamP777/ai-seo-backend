-- migrations/001_initial_schema.up.sql
-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "timescaledb";

-- Users Table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    password_hash VARCHAR(255),
    full_name VARCHAR(255),
    avatar_url TEXT,
    role VARCHAR(50) NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'editor', 'viewer', 'user')),
    subscription_tier VARCHAR(50) NOT NULL DEFAULT 'free' CHECK (subscription_tier IN ('free', 'basic', 'pro', 'enterprise')),
    subscription_status VARCHAR(50) NOT NULL DEFAULT 'inactive' CHECK (subscription_status IN ('active', 'inactive', 'canceled', 'past_due')),
    subscription_current_period_end TIMESTAMPTZ,
    trial_ends_at TIMESTAMPTZ,
    timezone VARCHAR(50) DEFAULT 'UTC',
    language VARCHAR(10) DEFAULT 'en',
    preferences JSONB DEFAULT '{}',
    api_key_hash VARCHAR(255),
    last_login_at TIMESTAMPTZ,
    login_count INTEGER DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    verification_token VARCHAR(255),
    reset_token VARCHAR(255),
    reset_token_expires TIMESTAMPTZ,
    mfa_enabled BOOLEAN NOT NULL DEFAULT false,
    mfa_secret VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_email CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

-- Organizations Table
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    website_url VARCHAR(500),
    logo_url TEXT,
    billing_email VARCHAR(255),
    billing_address JSONB,
    subscription_tier VARCHAR(50) NOT NULL DEFAULT 'free',
    subscription_status VARCHAR(50) NOT NULL DEFAULT 'inactive',
    subscription_current_period_end TIMESTAMPTZ,
    trial_ends_at TIMESTAMPTZ,
    settings JSONB DEFAULT '{}',
    limits JSONB DEFAULT '{
        "max_websites": 1,
        "max_scans_per_month": 10,
        "max_users": 1,
        "max_urls_per_scan": 50,
        "enable_ai_fixes": false,
        "enable_scheduled_scans": false
    }',
    is_active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_website_url CHECK (website_url IS NULL OR website_url ~ '^https?://[^\s/$.?#].[^\s]*$')
);

-- Organization Members Table
CREATE TABLE organization_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    permissions JSONB DEFAULT '[]',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    invited_by UUID REFERENCES users(id),
    invitation_token VARCHAR(255),
    invitation_status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (invitation_status IN ('pending', 'accepted', 'declined', 'expired')),
    invitation_expires_at TIMESTAMPTZ DEFAULT NOW() + INTERVAL '7 days',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, user_id)
);

-- Websites Table (User/Organization tracked websites)
CREATE TABLE websites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    website_url VARCHAR(500) NOT NULL,
    website_name VARCHAR(200),
    verified BOOLEAN NOT NULL DEFAULT false,
    verification_method VARCHAR(50) CHECK (verification_method IN ('meta_tag', 'dns', 'file_upload', 'google_search_console')),
    verification_token VARCHAR(255),
    verification_metadata JSONB,
    crawl_settings JSONB DEFAULT '{}',
    scan_settings JSONB DEFAULT '{}',
    notification_settings JSONB DEFAULT '{
        "email_weekly_report": true,
        "email_critical_issues": true,
        "slack_daily_summary": false,
        "webhook_on_scan_complete": false
    }',
    integration_settings JSONB DEFAULT '{
        "google_search_console": null,
        "google_analytics": null,
        "wordpress": null,
        "shopify": null
    }',
    last_scan_id UUID,
    last_scan_at TIMESTAMPTZ,
    seo_score DECIMAL(5,2),
    seo_score_updated_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_monitored BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_website_url CHECK (website_url ~ '^https?://[^\s/$.?#].[^\s]*$'),
    UNIQUE(user_id, website_url)
);

-- Website Scans Table
CREATE TABLE website_scans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    website_id UUID REFERENCES websites(id) ON DELETE CASCADE,
    website_url VARCHAR(500) NOT NULL,
    website_name VARCHAR(200),
    scan_type VARCHAR(50) NOT NULL CHECK (scan_type IN ('quick', 'deep', 'technical', 'content', 'performance', 'mobile', 'security')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'crawling', 'analyzing', 'completed', 'failed', 'cancelled')),
    total_urls INTEGER NOT NULL DEFAULT 0,
    urls_crawled INTEGER NOT NULL DEFAULT 0,
    scan_config JSONB NOT NULL DEFAULT '{}',
    scan_results JSONB,
    issues_found INTEGER DEFAULT 0,
    critical_issues INTEGER DEFAULT 0,
    warning_issues INTEGER DEFAULT 0,
    info_issues INTEGER DEFAULT 0,
    scan_duration INTERVAL,
    ai_analysis JSONB,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ DEFAULT NOW() + INTERVAL '30 days',
    CONSTRAINT valid_url CHECK (website_url ~ '^https?://[^\s/$.?#].[^\s]*$'),
    CONSTRAINT valid_urls_count CHECK (total_urls >= 0 AND total_urls <= 1000),
    CONSTRAINT valid_crawled_count CHECK (urls_crawled >= 0 AND urls_crawled <= total_urls)
);

-- SEO Issues Table
CREATE TABLE seo_issues (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scan_id UUID NOT NULL REFERENCES website_scans(id) ON DELETE CASCADE,
    issue_code VARCHAR(50) NOT NULL,
    issue_type VARCHAR(50) NOT NULL CHECK (issue_type IN ('technical', 'content', 'performance', 'mobile', 'security', 'on-page', 'off-page')),
    category VARCHAR(50) NOT NULL CHECK (category IN ('meta_tags', 'headings', 'images', 'links', 'speed', 'mobile', 'security', 'schema', 'content', 'crawlability')),
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    priority_score DECIMAL(5,4) NOT NULL CHECK (priority_score >= 0 AND priority_score <= 1),
    impact_score DECIMAL(5,4) CHECK (impact_score >= 0 AND impact_score <= 1),
    difficulty_score DECIMAL(5,4) CHECK (difficulty_score >= 0 AND difficulty_score <= 1),
    url VARCHAR(500) NOT NULL,
    element_path TEXT,
    element_selector TEXT,
    current_value TEXT,
    recommended_value TEXT,
    description TEXT NOT NULL,
    ai_explanation TEXT,
    fix_complexity VARCHAR(20) CHECK (fix_complexity IN ('trivial', 'simple', 'moderate', 'complex', 'very_complex')),
    estimated_fix_time INTERVAL,
    auto_fix_available BOOLEAN NOT NULL DEFAULT false,
    auto_fix_type VARCHAR(50) CHECK (auto_fix_type IN ('direct', 'ai_generated', 'manual', 'semi_auto')),
    status VARCHAR(20) NOT NULL DEFAULT 'detected' CHECK (status IN ('detected', 'acknowledged', 'fixing', 'fixed', 'won_fix', 'false_positive')),
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ,
    fixed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_url_format CHECK (url ~ '^https?://[^\s/$.?#].[^\s]*$'),
    CONSTRAINT valid_scores CHECK (
        priority_score >= 0 AND priority_score <= 1 AND
        (impact_score IS NULL OR (impact_score >= 0 AND impact_score <= 1)) AND
        (difficulty_score IS NULL OR (difficulty_score >= 0 AND difficulty_score <= 1))
    )
);

-- Applied Fixes Table
CREATE TABLE applied_fixes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    issue_id UUID NOT NULL REFERENCES seo_issues(id) ON DELETE CASCADE,
    fix_code VARCHAR(50) NOT NULL,
    fix_type VARCHAR(50) NOT NULL CHECK (fix_type IN ('meta_tag', 'image', 'link', 'speed', 'schema', 'content', 'redirect', 'header', 'mobile')),
    method VARCHAR(50) NOT NULL CHECK (method IN ('auto', 'semi_auto', 'manual', 'ai_generated')),
    original_value TEXT,
    new_value TEXT,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_by UUID REFERENCES users(id) ON DELETE SET NULL,
    applied_by_type VARCHAR(20) NOT NULL CHECK (applied_by_type IN ('system', 'user', 'ai', 'integration')),
    status VARCHAR(20) NOT NULL DEFAULT 'applied' CHECK (status IN ('applied', 'pending', 'failed', 'rolled_back', 'verified')),
    rollback_available BOOLEAN NOT NULL DEFAULT true,
    rollback_data JSONB,
    verification_result JSONB,
    verification_status VARCHAR(20) CHECK (verification_status IN ('not_verified', 'verified', 'failed', 'partial')),
    verification_attempts INTEGER DEFAULT 0,
    ai_model_used VARCHAR(100),
    tokens_used INTEGER,
    cost_usd DECIMAL(10,6),
    execution_duration INTERVAL,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_cost CHECK (cost_usd >= 0),
    CONSTRAINT valid_tokens CHECK (tokens_used >= 0)
);

-- Weekly Reports Table
CREATE TABLE weekly_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    website_id UUID REFERENCES websites(id) ON DELETE CASCADE,
    website_url VARCHAR(500) NOT NULL,
    website_name VARCHAR(200),
    report_date DATE NOT NULL,
    report_period_start DATE NOT NULL,
    report_period_end DATE NOT NULL,
    report_type VARCHAR(20) NOT NULL DEFAULT 'weekly' CHECK (report_type IN ('daily', 'weekly', 'monthly', 'quarterly')),
    report_data JSONB NOT NULL,
    ai_insights JSONB,
    recommendations JSONB,
    performance_trend JSONB,
    comparison_data JSONB,
    email_sent BOOLEAN NOT NULL DEFAULT false,
    email_sent_at TIMESTAMPTZ,
    email_recipients JSONB,
    dashboard_data JSONB,
    generated_by VARCHAR(50) DEFAULT 'system',
    generation_duration INTERVAL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_report_date CHECK (report_date >= report_period_start AND report_date <= report_period_end),
    CONSTRAINT valid_period CHECK (report_period_start < report_period_end),
    CONSTRAINT unique_user_website_report UNIQUE (user_id, website_url, report_date, report_type)
);

-- SEO Health Scores (TimescaleDB hypertable)
CREATE TABLE seo_health_scores (
    time TIMESTAMPTZ NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    website_id UUID REFERENCES websites(id) ON DELETE CASCADE,
    website_url VARCHAR(500) NOT NULL,
    scan_id UUID REFERENCES website_scans(id) ON DELETE SET NULL,
    overall_score DECIMAL(5,2) NOT NULL CHECK (overall_score >= 0 AND overall_score <= 100),
    technical_score DECIMAL(5,2) CHECK (technical_score >= 0 AND technical_score <= 100),
    content_score DECIMAL(5,2) CHECK (content_score >= 0 AND content_score <= 100),
    performance_score DECIMAL(5,2) CHECK (performance_score >= 0 AND performance_score <= 100),
    mobile_score DECIMAL(5,2) CHECK (mobile_score >= 0 AND mobile_score <= 100),
    security_score DECIMAL(5,2) CHECK (security_score >= 0 AND security_score <= 100),
    accessibility_score DECIMAL(5,2) CHECK (accessibility_score >= 0 AND accessibility_score <= 100),
    week_over_week_change DECIMAL(5,2),
    month_over_month_change DECIMAL(5,2),
    issues_resolved INTEGER DEFAULT 0,
    issues_introduced INTEGER DEFAULT 0,
    total_issues INTEGER DEFAULT 0,
    critical_issues INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (time, user_id, website_url)
);

-- Issue Categories Reference Table
CREATE TABLE seo_issue_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    issue_code VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    subcategory VARCHAR(50),
    default_severity VARCHAR(20) NOT NULL,
    default_priority_score DECIMAL(5,4) NOT NULL,
    description TEXT NOT NULL,
    ai_prompt_template TEXT,
    auto_fix_template TEXT,
    validation_rules JSONB,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Scan Templates Table
CREATE TABLE scan_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    scan_config JSONB NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- AI Integration Settings
CREATE TABLE ai_integration_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('openai', 'anthropic', 'gemini', 'azure_openai', 'custom')),
    model_name VARCHAR(100) NOT NULL,
    api_key_encrypted TEXT,
    base_url VARCHAR(500),
    default_parameters JSONB DEFAULT '{
        "temperature": 0.2,
        "max_tokens": 4000,
        "top_p": 0.95
    }',
    is_active BOOLEAN NOT NULL DEFAULT true,
    usage_stats JSONB DEFAULT '{
        "total_tokens": 0,
        "total_cost": 0,
        "total_requests": 0,
        "last_used": null
    }',
    rate_limits JSONB DEFAULT '{
        "requests_per_minute": 60,
        "tokens_per_minute": 60000
    }',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, organization_id, provider)
);

-- Automation Rules
CREATE TABLE automation_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    trigger_type VARCHAR(50) NOT NULL CHECK (trigger_type IN ('schedule', 'scan_complete', 'issue_detected', 'score_change')),
    trigger_config JSONB NOT NULL,
    conditions JSONB DEFAULT '[]',
    actions JSONB NOT NULL DEFAULT '[]',
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_triggered_at TIMESTAMPTZ,
    next_trigger_at TIMESTAMPTZ,
    execution_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Audit Log Table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    action_type VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    changes JSONB,
    ip_address INET,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- API Keys Table
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    key_prefix VARCHAR(8) NOT NULL,
    scopes JSONB NOT NULL DEFAULT '[]',
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create TimescaleDB hypertable
SELECT create_hypertable(
    'seo_health_scores', 
    'time',
    chunk_time_interval => INTERVAL '7 days',
    if_not_exists => TRUE
);

-- Add compression to hypertable
ALTER TABLE seo_health_scores SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'user_id, website_url',
    timescaledb.compress_orderby = 'time DESC'
);

-- Create indexes for website_scans
CREATE INDEX idx_website_scans_user_id ON website_scans(user_id);
CREATE INDEX idx_website_scans_website_url ON website_scans(website_url);
CREATE INDEX idx_website_scans_status ON website_scans(status);
CREATE INDEX idx_website_scans_created_at ON website_scans(created_at);
CREATE INDEX idx_website_scans_completed_at ON website_scans(completed_at);
CREATE INDEX idx_website_scans_scan_type ON website_scans(scan_type);
CREATE INDEX idx_website_scans_organization_id ON website_scans(organization_id);
CREATE INDEX idx_website_scans_website_id ON website_scans(website_id);
CREATE INDEX idx_website_scans_user_website ON website_scans(user_id, website_url, created_at DESC);
CREATE INDEX idx_website_scans_website_status ON website_scans(website_id, status, created_at DESC);

-- Create indexes for seo_issues
CREATE INDEX idx_seo_issues_scan_id ON seo_issues(scan_id);
CREATE INDEX idx_seo_issues_issue_type ON seo_issues(issue_type);
CREATE INDEX idx_seo_issues_severity ON seo_issues(severity);
CREATE INDEX idx_seo_issues_priority_score ON seo_issues(priority_score DESC);
CREATE INDEX idx_seo_issues_auto_fix_available ON seo_issues(auto_fix_available);
CREATE INDEX idx_seo_issues_status ON seo_issues(status);
CREATE INDEX idx_seo_issues_url ON seo_issues(url);
CREATE INDEX idx_seo_issues_category ON seo_issues(category);
CREATE INDEX idx_seo_issues_detected_at ON seo_issues(detected_at);
CREATE INDEX idx_seo_issues_scan_priority ON seo_issues(scan_id, priority_score DESC);
CREATE INDEX idx_seo_issues_website_url ON seo_issues((scan_id::text), url);
CREATE INDEX idx_seo_issues_fix_status ON seo_issues(scan_id, status, auto_fix_available);

-- Create indexes for applied_fixes
CREATE INDEX idx_applied_fixes_issue_id ON applied_fixes(issue_id);
CREATE INDEX idx_applied_fixes_fix_type ON applied_fixes(fix_type);
CREATE INDEX idx_applied_fixes_method ON applied_fixes(method);
CREATE INDEX idx_applied_fixes_status ON applied_fixes(status);
CREATE INDEX idx_applied_fixes_applied_at ON applied_fixes(applied_at);
CREATE INDEX idx_applied_fixes_applied_by_type ON applied_fixes(applied_by_type);
CREATE INDEX idx_applied_fixes_verification_status ON applied_fixes(verification_status);
CREATE INDEX idx_applied_fixes_created_at ON applied_fixes(created_at DESC);

-- Create indexes for weekly_reports
CREATE INDEX idx_weekly_reports_user_id ON weekly_reports(user_id);
CREATE INDEX idx_weekly_reports_website_url ON weekly_reports(website_url);
CREATE INDEX idx_weekly_reports_report_date ON weekly_reports(report_date DESC);
CREATE INDEX idx_weekly_reports_report_type ON weekly_reports(report_type);
CREATE INDEX idx_weekly_reports_email_sent ON weekly_reports(email_sent);
CREATE INDEX idx_weekly_reports_organization_id ON weekly_reports(organization_id);
CREATE INDEX idx_weekly_reports_period ON weekly_reports(report_period_start, report_period_end);
CREATE INDEX idx_weekly_reports_website_id ON weekly_reports(website_id);

-- Create indexes for websites
CREATE INDEX idx_websites_user_id ON websites(user_id);
CREATE INDEX idx_websites_organization_id ON websites(organization_id);
CREATE INDEX idx_websites_website_url ON websites(website_url);
CREATE INDEX idx_websites_is_active ON websites(is_active);
CREATE INDEX idx_websites_is_monitored ON websites(is_monitored);
CREATE INDEX idx_websites_seo_score ON websites(seo_score DESC);

-- Create indexes for users
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_subscription_status ON users(subscription_status);
CREATE INDEX idx_users_is_active ON users(is_active);
CREATE INDEX idx_users_created_at ON users(created_at DESC);

-- Create indexes for organizations
CREATE INDEX idx_organizations_slug ON organizations(slug);
CREATE INDEX idx_organizations_is_active ON organizations(is_active);

-- Create indexes for organization_members
CREATE INDEX idx_org_members_org_id ON organization_members(organization_id);
CREATE INDEX idx_org_members_user_id ON organization_members(user_id);
CREATE INDEX idx_org_members_status ON organization_members(invitation_status);

-- Create indexes for audit_logs
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_org_id ON audit_logs(organization_id);
CREATE INDEX idx_audit_logs_action_type ON audit_logs(action_type);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);

-- Create indexes for api_keys
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX idx_api_keys_is_active ON api_keys(is_active);

-- Create indexes for automation_rules
CREATE INDEX idx_automation_rules_user_id ON automation_rules(user_id);
CREATE INDEX idx_automation_rules_is_active ON automation_rules(is_active);
CREATE INDEX idx_automation_rules_next_trigger_at ON automation_rules(next_trigger_at);

-- Create indexes for seo_health_scores (hypertable specific)
CREATE INDEX idx_health_scores_user_time ON seo_health_scores(user_id, time DESC);
CREATE INDEX idx_health_scores_website_time ON seo_health_scores(website_url, time DESC);
CREATE INDEX idx_health_scores_organization ON seo_health_scores(organization_id, time DESC);
CREATE INDEX idx_health_scores_overall ON seo_health_scores(overall_score, time DESC);
CREATE INDEX idx_health_scores_website_id ON seo_health_scores(website_id, time DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_org_members_updated_at BEFORE UPDATE ON organization_members FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_websites_updated_at BEFORE UPDATE ON websites FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_website_scans_updated_at BEFORE UPDATE ON website_scans FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_seo_issues_updated_at BEFORE UPDATE ON seo_issues FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_applied_fixes_updated_at BEFORE UPDATE ON applied_fixes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_weekly_reports_updated_at BEFORE UPDATE ON weekly_reports FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_seo_issue_categories_updated_at BEFORE UPDATE ON seo_issue_categories FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_scan_templates_updated_at BEFORE UPDATE ON scan_templates FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_ai_integration_settings_updated_at BEFORE UPDATE ON ai_integration_settings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_automation_rules_updated_at BEFORE UPDATE ON automation_rules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update website's last scan info
CREATE OR REPLACE FUNCTION update_website_last_scan()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'completed' AND NEW.completed_at IS NOT NULL THEN
        UPDATE websites 
        SET last_scan_id = NEW.id, 
            last_scan_at = NEW.completed_at,
            seo_score = (NEW.scan_results->>'overall_score')::DECIMAL,
            seo_score_updated_at = NEW.completed_at
        WHERE id = NEW.website_id OR website_url = NEW.website_url;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_website_after_scan 
AFTER UPDATE ON website_scans 
FOR EACH ROW 
WHEN (OLD.status != 'completed' AND NEW.status = 'completed')
EXECUTE FUNCTION update_website_last_scan();

-- Function to track issue counts
CREATE OR REPLACE FUNCTION update_scan_issue_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE website_scans 
        SET issues_found = issues_found + 1,
            critical_issues = critical_issues + (CASE WHEN NEW.severity = 'critical' THEN 1 ELSE 0 END),
            warning_issues = warning_issues + (CASE WHEN NEW.severity IN ('high', 'medium') THEN 1 ELSE 0 END),
            info_issues = info_issues + (CASE WHEN NEW.severity IN ('low', 'info') THEN 1 ELSE 0 END)
        WHERE id = NEW.scan_id;
    ELSIF TG_OP = 'UPDATE' AND OLD.severity != NEW.severity THEN
        -- Remove from old severity count
        UPDATE website_scans 
        SET critical_issues = critical_issues - (CASE WHEN OLD.severity = 'critical' THEN 1 ELSE 0 END),
            warning_issues = warning_issues - (CASE WHEN OLD.severity IN ('high', 'medium') THEN 1 ELSE 0 END),
            info_issues = info_issues - (CASE WHEN OLD.severity IN ('low', 'info') THEN 1 ELSE 0 END)
        WHERE id = NEW.scan_id;
        
        -- Add to new severity count
        UPDATE website_scans 
        SET critical_issues = critical_issues + (CASE WHEN NEW.severity = 'critical' THEN 1 ELSE 0 END),
            warning_issues = warning_issues + (CASE WHEN NEW.severity IN ('high', 'medium') THEN 1 ELSE 0 END),
            info_issues = info_issues + (CASE WHEN NEW.severity IN ('low', 'info') THEN 1 ELSE 0 END)
        WHERE id = NEW.scan_id;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_scan_counts_on_issue 
AFTER INSERT OR UPDATE OF severity ON seo_issues 
FOR EACH ROW 
EXECUTE FUNCTION update_scan_issue_counts();

CREATE TRIGGER remove_scan_counts_on_issue_delete
AFTER DELETE ON seo_issues
FOR EACH ROW
EXECUTE FUNCTION update_scan_issue_counts();

-- Function to update issue status when fix is applied
CREATE OR REPLACE FUNCTION update_issue_on_fix()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'applied' OR NEW.status = 'verified' THEN
        UPDATE seo_issues 
        SET status = 'fixed',
            fixed_at = NEW.applied_at,
            updated_at = NOW()
        WHERE id = NEW.issue_id;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_issue_after_fix 
AFTER INSERT OR UPDATE OF status ON applied_fixes 
FOR EACH ROW 
WHEN (NEW.status IN ('applied', 'verified') AND (OLD.status IS NULL OR OLD.status NOT IN ('applied', 'verified')))
EXECUTE FUNCTION update_issue_on_fix();

-- Function to prevent duplicate active automation rules
CREATE OR REPLACE FUNCTION prevent_duplicate_automation_rules()
RETURNS TRIGGER AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM automation_rules 
        WHERE user_id = NEW.user_id 
        AND organization_id IS NOT DISTINCT FROM NEW.organization_id
        AND name = NEW.name 
        AND is_active = true
        AND id != COALESCE(NEW.id, '00000000-0000-0000-0000-000000000000'::uuid)
    ) THEN
        RAISE EXCEPTION 'Duplicate active automation rule with the same name';
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER check_duplicate_automation_rules
BEFORE INSERT OR UPDATE ON automation_rules
FOR EACH ROW
EXECUTE FUNCTION prevent_duplicate_automation_rules();

-- Function to enforce subscription limits
CREATE OR REPLACE FUNCTION check_subscription_limits()
RETURNS TRIGGER AS $$
DECLARE
    user_limits JSONB;
    org_limits JSONB;
    current_count INTEGER;
    max_limit INTEGER;
BEGIN
    -- Get user limits
    SELECT limits INTO user_limits
    FROM organizations 
    WHERE id = NEW.organization_id AND is_active = true;
    
    -- If no organization, use user's own limits (from default org)
    IF user_limits IS NULL THEN
        SELECT '{"max_websites": 1, "max_scans_per_month": 10, "max_users": 1, "max_urls_per_scan": 50, "enable_ai_fixes": false, "enable_scheduled_scans": false}'::JSONB 
        INTO user_limits;
    END IF;
    
    -- Check website limits
    IF TG_TABLE_NAME = 'websites' AND TG_OP = 'INSERT' THEN
        SELECT COUNT(*) INTO current_count
        FROM websites 
        WHERE user_id = NEW.user_id 
        AND organization_id IS NOT DISTINCT FROM NEW.organization_id
        AND is_active = true;
        
        max_limit := (user_limits->>'max_websites')::INTEGER;
        
        IF current_count >= max_limit THEN
            RAISE EXCEPTION 'Website limit reached. Maximum allowed: %', max_limit;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER enforce_website_limits
BEFORE INSERT ON websites
FOR EACH ROW
EXECUTE FUNCTION check_subscription_limits();

-- Function to clean up expired data
CREATE OR REPLACE FUNCTION cleanup_expired_data()
RETURNS void AS $$
BEGIN
    -- Delete expired scans (older than 30 days)
    DELETE FROM website_scans 
    WHERE expires_at < NOW() - INTERVAL '30 days';
    
    -- Archive old audit logs (older than 90 days)
    DELETE FROM audit_logs 
    WHERE created_at < NOW() - INTERVAL '90 days';
    
    -- Deactivate expired API keys
    UPDATE api_keys 
    SET is_active = false 
    WHERE expires_at < NOW() AND is_active = true;
END;
$$ language 'plpgsql';

-- Insert default scan templates
INSERT INTO scan_templates (id, name, description, scan_config, is_default, created_at, updated_at) VALUES
(
    uuid_generate_v4(),
    'Quick Scan',
    'Basic scan for common SEO issues',
    '{
        "check_metadata": true,
        "check_performance": true,
        "check_mobile": true,
        "check_security": false,
        "respect_robots_txt": true,
        "max_urls": 20,
        "parallel_requests": 3,
        "timeout_seconds": 30
    }',
    true,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'Deep Technical Scan',
    'Comprehensive technical SEO analysis',
    '{
        "check_metadata": true,
        "check_performance": true,
        "check_mobile": true,
        "check_security": true,
        "respect_robots_txt": true,
        "max_urls": 100,
        "parallel_requests": 5,
        "timeout_seconds": 60,
        "crawl_sitemap": true,
        "check_headers": true,
        "check_redirects": true,
        "check_canonical": true
    }',
    false,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'Content Analysis',
    'Focus on content optimization',
    '{
        "check_metadata": true,
        "check_performance": false,
        "check_mobile": false,
        "check_security": false,
        "respect_robots_txt": true,
        "max_urls": 50,
        "parallel_requests": 2,
        "timeout_seconds": 45,
        "analyze_content": true,
        "check_keywords": true,
        "check_readability": true,
        "check_duplicate": true
    }',
    false,
    NOW(),
    NOW()
);

-- Insert default SEO issue categories
INSERT INTO seo_issue_categories (id, issue_code, display_name, category, subcategory, default_severity, default_priority_score, description, ai_prompt_template, auto_fix_template, is_active, created_at, updated_at) VALUES
-- Meta Tags Issues
(
    uuid_generate_v4(),
    'MISSING_META_TITLE',
    'Missing Meta Title',
    'meta_tags',
    'title',
    'critical',
    0.95,
    'Page is missing the meta title tag. Titles are critical for SEO and user experience.',
    'Generate an SEO-optimized meta title for the page with content: {content}. Include primary keyword: {keyword}. Max 60 characters.',
    '<title>{generated_title}</title>',
    true,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'MISSING_META_DESCRIPTION',
    'Missing Meta Description',
    'meta_tags',
    'description',
    'high',
    0.85,
    'Page is missing the meta description. Descriptions improve click-through rates in search results.',
    'Generate an SEO-optimized meta description for the page with content: {content}. Include primary keyword: {keyword}. Max 160 characters.',
    '<meta name="description" content="{generated_description}">',
    true,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'DUPLICATE_META_TITLE',
    'Duplicate Meta Title',
    'meta_tags',
    'title',
    'medium',
    0.75,
    'Multiple pages share the same meta title. This can cause SEO cannibalization.',
    'Generate unique meta titles for similar pages. Original: {original_title}. Page content: {content}.',
    '<title>{unique_title}</title>',
    true,
    NOW(),
    NOW()
),
-- Heading Issues
(
    uuid_generate_v4(),
    'MISSING_H1',
    'Missing H1 Heading',
    'headings',
    'structure',
    'critical',
    0.90,
    'Page is missing the H1 heading tag. H1 is the most important heading for SEO.',
    'Generate an appropriate H1 heading for the page with content: {content}. Include primary keyword.',
    '<h1>{generated_h1}</h1>',
    true,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'MULTIPLE_H1',
    'Multiple H1 Headings',
    'headings',
    'structure',
    'medium',
    0.65,
    'Page has multiple H1 headings. Each page should have only one H1.',
    'Consolidate multiple H1 headings or demote secondary H1s to H2. Headings: {headings}.',
    '<h2>{demoted_heading}</h2>',
    true,
    NOW(),
    NOW()
),
-- Image Issues
(
    uuid_generate_v4(),
    'MISSING_ALT_TEXT',
    'Missing Image Alt Text',
    'images',
    'accessibility',
    'medium',
    0.70,
    'Image is missing alt text. Alt text is important for accessibility and image SEO.',
    'Generate descriptive alt text for the image. Context: {context}. Image filename: {filename}.',
    'alt="{generated_alt}"',
    true,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'LARGE_IMAGE_SIZE',
    'Large Image File Size',
    'images',
    'performance',
    'high',
    0.80,
    'Image file size is too large, affecting page load speed.',
    'Suggest optimized image dimensions and format. Current size: {size}. Dimensions: {dimensions}.',
    'data-optimize="true" data-suggested-format="{format}" data-suggested-dimensions="{dimensions}"',
    true,
    NOW(),
    NOW()
),
-- Link Issues
(
    uuid_generate_v4(),
    'BROKEN_LINK',
    'Broken Link',
    'links',
    'functionality',
    'critical',
    0.92,
    'Link points to a non-existent page (404 error).',
    'Suggest alternative URLs or removal. Broken URL: {url}. Context: {context}.',
    'href="{suggested_url}"',
    false,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'NON_HTTPS_LINK',
    'Non-HTTPS Link',
    'links',
    'security',
    'medium',
    0.60,
    'Link uses HTTP instead of HTTPS. HTTPS is important for security and SEO.',
    'Convert HTTP URLs to HTTPS. Original: {url}.',
    'href="{https_url}"',
    true,
    NOW(),
    NOW()
),
-- Performance Issues
(
    uuid_generate_v4(),
    'SLOW_LOAD_TIME',
    'Slow Page Load Time',
    'performance',
    'speed',
    'high',
    0.85,
    'Page takes too long to load, affecting user experience and SEO.',
    'Suggest performance optimizations. Current load time: {load_time}. Issues: {issues}.',
    NULL,
    false,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'UNMINIFIED_CSS_JS',
    'Unminified CSS/JavaScript',
    'performance',
    'optimization',
    'medium',
    0.65,
    'CSS or JavaScript files are not minified, increasing page size.',
    'Suggest minification tools and techniques. Files: {files}.',
    NULL,
    false,
    NOW(),
    NOW()
),
-- Mobile Issues
(
    uuid_generate_v4(),
    'NOT_MOBILE_FRIENDLY',
    'Not Mobile Friendly',
    'mobile',
    'responsiveness',
    'critical',
    0.88,
    'Page is not optimized for mobile devices.',
    'Suggest mobile optimization improvements. Issues: {issues}.',
    '<meta name="viewport" content="width=device-width, initial-scale=1.0">',
    true,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'SMALL_TOUCH_TARGETS',
    'Small Touch Targets',
    'mobile',
    'usability',
    'medium',
    0.55,
    'Touch targets (buttons, links) are too small for mobile users.',
    'Suggest minimum touch target sizes. Elements: {elements}.',
    'style="min-width: 44px; min-height: 44px;"',
    true,
    NOW(),
    NOW()
),
-- Security Issues
(
    uuid_generate_v4(),
    'MISSING_SECURITY_HEADERS',
    'Missing Security Headers',
    'security',
    'headers',
    'high',
    0.78,
    'Important security headers are missing, making the site vulnerable.',
    'Suggest appropriate security headers. Current headers: {headers}.',
    'Header: {header_name} Value: {header_value}',
    false,
    NOW(),
    NOW()
),
-- Schema Issues
(
    uuid_generate_v4(),
    'MISSING_SCHEMA_MARKUP',
    'Missing Schema Markup',
    'schema',
    'structured_data',
    'medium',
    0.68,
    'Page is missing structured data (schema.org markup).',
    'Generate appropriate JSON-LD schema markup for content: {content}. Type: {page_type}.',
    '<script type="application/ld+json">{generated_schema}</script>',
    true,
    NOW(),
    NOW()
),
-- Content Issues
(
    uuid_generate_v4(),
    'THIN_CONTENT',
    'Thin Content',
    'content',
    'quality',
    'medium',
    0.72,
    'Page has very little content, which may not provide value to users.',
    'Suggest content expansion. Current content: {content}. Word count: {word_count}.',
    NULL,
    false,
    NOW(),
    NOW()
),
(
    uuid_generate_v4(),
    'DUPLICATE_CONTENT',
    'Duplicate Content',
    'content',
    'originality',
    'high',
    0.82,
    'Content is duplicated from other pages or websites.',
    'Suggest content differentiation. Duplicate sources: {sources}.',
    NULL,
    false,
    NOW(),
    NOW()
);

-- Insert default organization for users without one
INSERT INTO organizations (id, name, slug, description, settings, limits, created_at, updated_at)
SELECT 
    uuid_generate_v4(),
    CONCAT(u.full_name, '''s Organization'),
    CONCAT('user-', REPLACE(LOWER(u.email), '@', '-'), '-', EXTRACT(EPOCH FROM NOW())::INT::TEXT),
    'Personal organization for ' || u.email,
    '{"theme": "light", "notifications": true}',
    '{"max_websites": 1, "max_scans_per_month": 10, "max_users": 1, "max_urls_per_scan": 50, "enable_ai_fixes": false, "enable_scheduled_scans": false}',
    NOW(),
    NOW()
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM organizations o 
    JOIN organization_members om ON o.id = om.organization_id 
    WHERE om.user_id = u.id AND om.role = 'owner'
);

-- Add users to their default organizations as owners
INSERT INTO organization_members (id, organization_id, user_id, role, invitation_status, joined_at, created_at, updated_at)
SELECT 
    uuid_generate_v4(),
    o.id,
    u.id,
    'owner',
    'accepted',
    NOW(),
    NOW(),
    NOW()
FROM users u
CROSS JOIN LATERAL (
    SELECT id FROM organizations 
    WHERE slug LIKE CONCAT('user-', REPLACE(LOWER(u.email), '@', '-'), '-%')
    LIMIT 1
) o
WHERE NOT EXISTS (
    SELECT 1 FROM organization_members om 
    WHERE om.user_id = u.id AND om.role = 'owner'
);

-- Create comments for documentation
COMMENT ON TABLE users IS 'System users with authentication and subscription information';
COMMENT ON TABLE organizations IS 'Organizations or teams that users belong to';
COMMENT ON TABLE organization_members IS 'Membership relationship between users and organizations';
COMMENT ON TABLE websites IS 'Websites tracked by users or organizations';
COMMENT ON TABLE website_scans IS 'SEO scans performed on websites';
COMMENT ON TABLE seo_issues IS 'SEO issues detected during scans';
COMMENT ON TABLE applied_fixes IS 'Fixes applied to SEO issues';
COMMENT ON TABLE weekly_reports IS 'Weekly SEO performance reports';
COMMENT ON TABLE seo_health_scores IS 'Time-series SEO health scores (TimescaleDB hypertable)';
COMMENT ON TABLE seo_issue_categories IS 'Reference table for SEO issue types and templates';
COMMENT ON TABLE scan_templates IS 'Predefined scan configurations';
COMMENT ON TABLE ai_integration_settings IS 'AI provider settings and API keys';
COMMENT ON TABLE automation_rules IS 'Automation rules for scheduled actions';
COMMENT ON TABLE audit_logs IS 'Audit trail for system actions';
COMMENT ON TABLE api_keys IS 'API keys for programmatic access';

COMMENT ON COLUMN users.api_key_hash IS 'Hashed API key for programmatic access';
COMMENT ON COLUMN users.metadata IS 'Flexible field for user-specific data';
COMMENT ON COLUMN organizations.limits IS 'Subscription limits in JSON format';
COMMENT ON COLUMN websites.crawl_settings IS 'Custom crawl settings per website';
COMMENT ON COLUMN website_scans.scan_results IS 'Full scan results in JSON format';
COMMENT ON COLUMN seo_issues.priority_score IS 'Calculated priority score (0-1) for fixing';
COMMENT ON COLUMN applied_fixes.rollback_data IS 'Data needed to rollback the fix if needed';
COMMENT ON COLUMN weekly_reports.ai_insights IS 'AI-generated insights about SEO performance';
COMMENT ON COLUMN seo_health_scores.overall_score IS 'Overall SEO health score (0-100)';
COMMENT ON COLUMN automation_rules.trigger_config IS 'Configuration for when the rule triggers';
COMMENT ON COLUMN audit_logs.changes IS 'Before/after changes for update actions';

-- Create views for common queries
CREATE OR REPLACE VIEW website_overview AS
SELECT 
    w.id,
    w.user_id,
    w.organization_id,
    w.website_url,
    w.website_name,
    w.verified,
    w.seo_score,
    w.last_scan_at,
    ws.status as last_scan_status,
    ws.issues_found as last_scan_issues,
    ws.critical_issues as last_scan_critical,
    COUNT(DISTINCT s.id) as total_scans,
    COUNT(DISTINCT i.id) as total_issues,
    COUNT(DISTINCT CASE WHEN i.status = 'fixed' THEN i.id END) as fixed_issues,
    w.created_at,
    w.updated_at
FROM websites w
LEFT JOIN website_scans ws ON w.last_scan_id = ws.id
LEFT JOIN website_scans s ON w.id = s.website_id OR w.website_url = s.website_url
LEFT JOIN seo_issues i ON s.id = i.scan_id
WHERE w.is_active = true
GROUP BY w.id, w.user_id, w.organization_id, w.website_url, w.website_name, w.verified, 
         w.seo_score, w.last_scan_at, ws.status, ws.issues_found, ws.critical_issues, w.created_at, w.updated_at;

CREATE OR REPLACE VIEW user_dashboard_stats AS
SELECT 
    u.id as user_id,
    u.email,
    u.full_name,
    u.subscription_tier,
    COUNT(DISTINCT w.id) as total_websites,
    COUNT(DISTINCT ws.id) as total_scans,
    COUNT(DISTINCT i.id) as total_issues,
    COUNT(DISTINCT CASE WHEN i.status = 'fixed' THEN i.id END) as fixed_issues,
    COUNT(DISTINCT af.id) as total_fixes_applied,
    AVG(shs.overall_score) as avg_seo_score,
    MAX(ws.created_at) as last_scan_date,
    u.created_at
FROM users u
LEFT JOIN websites w ON u.id = w.user_id AND w.is_active = true
LEFT JOIN website_scans ws ON u.id = ws.user_id
LEFT JOIN seo_issues i ON ws.id = i.scan_id
LEFT JOIN applied_fixes af ON i.id = af.issue_id
LEFT JOIN seo_health_scores shs ON u.id = shs.user_id
WHERE u.is_active = true
GROUP BY u.id, u.email, u.full_name, u.subscription_tier, u.created_at;

CREATE OR REPLACE VIEW organization_stats AS
SELECT 
    o.id as organization_id,
    o.name,
    o.slug,
    o.subscription_tier,
    COUNT(DISTINCT om.user_id) as total_members,
    COUNT(DISTINCT w.id) as total_websites,
    COUNT(DISTINCT ws.id) as total_scans,
    COUNT(DISTINCT i.id) as total_issues,
    COUNT(DISTINCT af.id) as total_fixes_applied,
    AVG(shs.overall_score) as avg_seo_score,
    o.created_at
FROM organizations o
LEFT JOIN organization_members om ON o.id = om.organization_id AND om.invitation_status = 'accepted'
LEFT JOIN websites w ON o.id = w.organization_id AND w.is_active = true
LEFT JOIN website_scans ws ON o.id = ws.organization_id
LEFT JOIN seo_issues i ON ws.id = i.scan_id
LEFT JOIN applied_fixes af ON i.id = af.issue_id
LEFT JOIN seo_health_scores shs ON o.id = shs.organization_id
WHERE o.is_active = true
GROUP BY o.id, o.name, o.slug, o.subscription_tier, o.created_at;

-- Create materialized views for performance
CREATE MATERIALIZED VIEW IF NOT EXISTS daily_seo_metrics AS
SELECT 
    DATE_TRUNC('day', time) as day,
    website_id,
    website_url,
    AVG(overall_score) as avg_score,
    MIN(overall_score) as min_score,
    MAX(overall_score) as max_score,
    SUM(issues_resolved) as total_issues_resolved,
    SUM(issues_introduced) as total_issues_introduced,
    COUNT(*) as data_points
FROM seo_health_scores
WHERE time >= NOW() - INTERVAL '30 days'
GROUP BY DATE_TRUNC('day', time), website_id, website_url
WITH DATA;

CREATE UNIQUE INDEX idx_daily_seo_metrics ON daily_seo_metrics (day, website_id);
REFRESH MATERIALIZED VIEW CONCURRENTLY daily_seo_metrics;

-- Create indexes on materialized views
CREATE INDEX idx_daily_metrics_website ON daily_seo_metrics(website_id, day DESC);
CREATE INDEX idx_daily_metrics_day ON daily_seo_metrics(day DESC);

-- Create function to refresh materialized views
CREATE OR REPLACE FUNCTION refresh_materialized_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY daily_seo_metrics;
END;
$$ language 'plpgsql';

-- Schedule materialized view refresh (would be called by a cron job)
-- SELECT cron.schedule('refresh-daily-metrics', '0 2 * * *', 'SELECT refresh_materialized_views();');

-- Create function to get website health trend
CREATE OR REPLACE FUNCTION get_website_health_trend(
    p_website_id UUID,
    p_days INTEGER DEFAULT 30
)
RETURNS TABLE (
    day DATE,
    overall_score DECIMAL(5,2),
    technical_score DECIMAL(5,2),
    content_score DECIMAL(5,2),
    performance_score DECIMAL(5,2),
    total_issues INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        DATE_TRUNC('day', shs.time)::DATE as day,
        AVG(shs.overall_score) as overall_score,
        AVG(shs.technical_score) as technical_score,
        AVG(shs.content_score) as content_score,
        AVG(shs.performance_score) as performance_score,
        SUM(shs.total_issues) as total_issues
    FROM seo_health_scores shs
    WHERE shs.website_id = p_website_id
    AND shs.time >= NOW() - (p_days || ' days')::INTERVAL
    GROUP BY DATE_TRUNC('day', shs.time)::DATE
    ORDER BY day DESC;
END;
$$ language 'plpgsql';

-- Create function to get critical issues summary
CREATE OR REPLACE FUNCTION get_critical_issues_summary(
    p_user_id UUID,
    p_days INTEGER DEFAULT 7
)
RETURNS TABLE (
    website_url VARCHAR(500),
    website_name VARCHAR(200),
    total_critical INTEGER,
    total_high INTEGER,
    unfixed_critical INTEGER,
    unfixed_high INTEGER,
    last_scan_date TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        w.website_url,
        w.website_name,
        COUNT(CASE WHEN i.severity = 'critical' THEN i.id END) as total_critical,
        COUNT(CASE WHEN i.severity = 'high' THEN i.id END) as total_high,
        COUNT(CASE WHEN i.severity = 'critical' AND i.status != 'fixed' THEN i.id END) as unfixed_critical,
        COUNT(CASE WHEN i.severity = 'high' AND i.status != 'fixed' THEN i.id END) as unfixed_high,
        MAX(ws.completed_at) as last_scan_date
    FROM websites w
    LEFT JOIN website_scans ws ON w.id = ws.website_id AND ws.status = 'completed'
    LEFT JOIN seo_issues i ON ws.id = i.scan_id AND i.severity IN ('critical', 'high')
    WHERE w.user_id = p_user_id
    AND w.is_active = true
    AND ws.completed_at >= NOW() - (p_days || ' days')::INTERVAL
    GROUP BY w.id, w.website_url, w.website_name
    HAVING COUNT(CASE WHEN i.severity IN ('critical', 'high') AND i.status != 'fixed' THEN i.id END) > 0
    ORDER BY unfixed_critical DESC, unfixed_high DESC;
END;
$$ language 'plpgsql';

-- Grant appropriate permissions (in production, these would be more restrictive)
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO seo_diagnostics_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO seo_diagnostics_app;
GRANT USAGE ON SCHEMA public TO seo_diagnostics_app;

-- Create application user (run this separately with admin privileges)
-- CREATE USER seo_diagnostics_app WITH PASSWORD 'secure_password_here';
-- GRANT CONNECT ON DATABASE seo_diagnostics TO seo_diagnostics_app;