-- Rollback for 000001_initial_schema
-- Drops tables in reverse dependency order

DROP TABLE IF EXISTS "sync_jobs";
DROP TABLE IF EXISTS "download_logs";
DROP TABLE IF EXISTS "plugin_versions";
DROP TABLE IF EXISTS "submissions";
DROP TABLE IF EXISTS "plugins";
DROP TABLE IF EXISTS "trust_keys";
DROP TABLE IF EXISTS "admin_users";
