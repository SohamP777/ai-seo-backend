-- migrations/001_initial_schema.down.sql
-- Drop all tables in reverse order of creation

-- Drop materialized views first
DROP MATERIALIZED VIEW IF EXISTS daily_seo_metrics CASCADE;

-- Drop views
DROP VIEW IF EXISTS organization_stats CASCADE;
DROP VIEW IF EXISTS user_dashboard_stats CASCADE;
DROP VIEW IF EXISTS website_overview CASCADE;

-- Drop functions
DROP FUNCTION IF EXISTS get_critical_issues_summary(UUID, INTEGER) CASCADE;
DROP FUNCTION IF EXISTS get_website_health_trend(UUID, INTEGER) CASCADE;
DROP FUNCTION IF EXISTS refresh_materialized_views() CASCADE;
DROP FUNCTION IF EXISTS cleanup_expired_data() CASCADE;
DROP FUNCTION IF EXISTS check_subscription_limits() CASCADE;
DROP FUNCTION IF EXISTS prevent_duplicate_automation_rules() CASCADE;
DROP FUNCTION IF EXISTS update_issue_on_fix() CASCADE;
DROP FUNCTION IF EXISTS update_scan_issue_counts() CASCADE;
DROP FUNCTION IF EXISTS update_website_last_scan() CASCADE;
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;

-- Drop triggers (they should be automatically dropped with functions, but explicit for safety)
DROP TRIGGER IF EXISTS update_api_keys_updated_at ON api_keys;
DROP TRIGGER IF EXISTS update_automation_rules_updated_at ON automation_rules;
DROP TRIGGER IF EXISTS update_ai_integration_settings_updated_at ON ai_integration_settings;
DROP TRIGGER IF EXISTS update_scan_templates_updated_at ON scan_templates;
DROP TRIGGER IF EXISTS update_seo_issue_categories_updated_at ON seo_issue_categories;
DROP TRIGGER IF EXISTS update_weekly_reports_updated_at ON weekly_reports;
DROP TRIGGER IF EXISTS update_applied_fixes_updated_at ON applied_fixes;
DROP TRIGGER IF EXISTS update_seo_issues_updated_at ON seo_issues;
DROP TRIGGER IF EXISTS update_website_scans_updated_at ON website_scans;
DROP TRIGGER IF EXISTS update_websites_updated_at ON websites;
DROP TRIGGER IF EXISTS update_org_members_updated_at ON organization_members;
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS enforce_website_limits ON websites;
DROP TRIGGER IF EXISTS check_duplicate_automation_rules ON automation_rules;
DROP TRIGGER IF EXISTS update_issue_after_fix ON applied_fixes;
DROP TRIGGER IF EXISTS remove_scan_counts_on_issue_delete ON seo_issues;
DROP TRIGGER IF EXISTS update_scan_counts_on_issue ON seo_issues;
DROP TRIGGER IF EXISTS update_website_after_scan ON website_scans;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS api_keys CASCADE;
DROP TABLE IF EXISTS automation_rules CASCADE;
DROP TABLE IF EXISTS ai_integration_settings CASCADE;
DROP TABLE IF EXISTS scan_templates CASCADE;
DROP TABLE IF EXISTS seo_issue_categories CASCADE;
DROP TABLE IF EXISTS seo_health_scores CASCADE;
DROP TABLE IF EXISTS weekly_reports CASCADE;
DROP TABLE IF EXISTS applied_fixes CASCADE;
DROP TABLE IF EXISTS seo_issues CASCADE;
DROP TABLE IF EXISTS website_scans CASCADE;
DROP TABLE IF EXISTS websites CASCADE;
DROP TABLE IF EXISTS organization_members CASCADE;
DROP TABLE IF EXISTS organizations CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Drop extensions
DROP EXTENSION IF EXISTS "timescaledb" CASCADE;
DROP EXTENSION IF EXISTS "pgcrypto" CASCADE;
DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;