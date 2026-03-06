-- Initial schema for Sub2API Plugin Market (Phase 0 + Phase 1)
-- Generated from ent/schema. Covers: plugin name Match constraint, plugin_type enum,
-- capabilities/config_schema JSON, submission->version edge, sync_job table.
-- Apply with: psql $DATABASE_URL -f migrations/000001_initial_schema.up.sql

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- admin_users
CREATE TABLE IF NOT EXISTS "admin_users" (
    "id" uuid NOT NULL,
    "username" varchar NOT NULL,
    "email" varchar NOT NULL,
    "password_hash" varchar NOT NULL,
    "role" varchar NOT NULL DEFAULT 'reviewer',
    "is_active" boolean NOT NULL DEFAULT true,
    "last_login_at" timestamptz NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS "adminuser_username" ON "admin_users" ("username");
CREATE UNIQUE INDEX IF NOT EXISTS "adminuser_email" ON "admin_users" ("email");
CREATE INDEX IF NOT EXISTS "adminuser_is_active" ON "admin_users" ("is_active");

-- trust_keys
CREATE TABLE IF NOT EXISTS "trust_keys" (
    "id" uuid NOT NULL,
    "key_id" varchar NOT NULL,
    "public_key" varchar NOT NULL,
    "key_type" varchar NOT NULL DEFAULT 'community',
    "owner_name" varchar NOT NULL,
    "owner_email" varchar NOT NULL,
    "description" text NULL,
    "is_active" boolean NOT NULL DEFAULT true,
    "expires_at" timestamptz NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS "trustkey_key_id" ON "trust_keys" ("key_id");
CREATE INDEX IF NOT EXISTS "trustkey_key_type_is_active" ON "trust_keys" ("key_type", "is_active");
CREATE INDEX IF NOT EXISTS "trustkey_is_active_expires_at" ON "trust_keys" ("is_active", "expires_at");

-- plugins (Phase 0: name Match regex; Phase 1: plugin_type, capabilities, config_schema)
CREATE TABLE IF NOT EXISTS "plugins" (
    "id" uuid NOT NULL,
    "name" varchar NOT NULL,
    "display_name" varchar NOT NULL,
    "description" text NULL,
    "author" varchar NOT NULL,
    "repository_url" varchar NULL,
    "homepage_url" varchar NULL,
    "license" varchar NOT NULL DEFAULT 'MIT',
    "category" varchar NOT NULL DEFAULT 'other',
    "plugin_type" varchar NULL,
    "tags" jsonb NULL,
    "is_official" boolean NOT NULL DEFAULT false,
    "is_verified" boolean NOT NULL DEFAULT false,
    "download_count" integer NOT NULL DEFAULT 0,
    "rating" double precision NULL,
    "source_type" varchar NOT NULL DEFAULT 'upload',
    "github_repo_url" varchar NULL,
    "github_repo_normalized" varchar NULL,
    "auto_upgrade_enabled" boolean NOT NULL DEFAULT false,
    "status" varchar NOT NULL DEFAULT 'active',
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS "plugin_name" ON "plugins" ("name");
CREATE INDEX IF NOT EXISTS "plugin_is_official_status" ON "plugins" ("is_official", "status");
CREATE INDEX IF NOT EXISTS "plugin_category_status" ON "plugins" ("category", "status");
CREATE INDEX IF NOT EXISTS "plugin_is_official_status_download_count" ON "plugins" ("is_official", "status", "download_count");
CREATE INDEX IF NOT EXISTS "plugin_github_repo_normalized" ON "plugins" ("github_repo_normalized");
CREATE INDEX IF NOT EXISTS "plugin_plugin_type_status" ON "plugins" ("plugin_type", "status");
-- Phase 0: name CHECK constraint for regex ^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$
ALTER TABLE "plugins" ADD CONSTRAINT "plugins_name_format" CHECK ("name" ~ '^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$');

-- submissions
CREATE TABLE IF NOT EXISTS "submissions" (
    "id" uuid NOT NULL,
    "submission_type" varchar NOT NULL,
    "submitter_email" varchar NOT NULL,
    "submitter_name" varchar NOT NULL,
    "notes" text NULL,
    "source_type" varchar NOT NULL DEFAULT 'upload',
    "github_repo_url" varchar NULL,
    "auto_upgrade_enabled" boolean NOT NULL DEFAULT false,
    "status" varchar NOT NULL DEFAULT 'pending',
    "reviewer_notes" text NULL,
    "reviewed_by" varchar NULL,
    "reviewed_at" timestamptz NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    "plugin_id" uuid NOT NULL,
    PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "submission_status_created_at" ON "submissions" ("status", "created_at");
CREATE INDEX IF NOT EXISTS "submission_plugin_id_status" ON "submissions" ("plugin_id", "status");
ALTER TABLE "submissions" ADD CONSTRAINT "submissions_plugins_submissions" FOREIGN KEY ("plugin_id") REFERENCES "plugins" ("id") ON DELETE NO ACTION;

-- plugin_versions (Phase 1: capabilities, config_schema, submission_version FK)
CREATE TABLE IF NOT EXISTS "plugin_versions" (
    "id" uuid NOT NULL,
    "version" varchar NOT NULL,
    "changelog" text NULL,
    "wasm_url" varchar NOT NULL,
    "wasm_hash" varchar NOT NULL,
    "signature" varchar NULL,
    "sign_key_id" varchar NULL,
    "file_size" integer NOT NULL,
    "min_api_version" varchar NOT NULL,
    "plugin_api_version" varchar NOT NULL,
    "max_api_version" varchar NULL,
    "dependencies" jsonb NULL,
    "capabilities" jsonb NULL,
    "config_schema" jsonb NULL,
    "status" varchar NOT NULL DEFAULT 'draft',
    "published_at" timestamptz NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    "plugin_id" uuid NOT NULL,
    "submission_version" uuid NULL,
    PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX IF NOT EXISTS "pluginversion_plugin_id_version" ON "plugin_versions" ("plugin_id", "version");
CREATE INDEX IF NOT EXISTS "pluginversion_plugin_id_status_published_at" ON "plugin_versions" ("plugin_id", "status", "published_at");
CREATE INDEX IF NOT EXISTS "pluginversion_status_published_at" ON "plugin_versions" ("status", "published_at");
CREATE UNIQUE INDEX IF NOT EXISTS "pluginversion_submission_version" ON "plugin_versions" ("submission_version");
ALTER TABLE "plugin_versions" ADD CONSTRAINT "plugin_versions_plugins_versions" FOREIGN KEY ("plugin_id") REFERENCES "plugins" ("id") ON DELETE NO ACTION;
ALTER TABLE "plugin_versions" ADD CONSTRAINT "plugin_versions_submissions_version" FOREIGN KEY ("submission_version") REFERENCES "submissions" ("id") ON DELETE SET NULL;

-- download_logs
CREATE TABLE IF NOT EXISTS "download_logs" (
    "id" uuid NOT NULL,
    "version" varchar NOT NULL,
    "client_ip" varchar NOT NULL,
    "user_agent" varchar NULL,
    "country_code" varchar(2) NULL,
    "success" boolean NOT NULL DEFAULT true,
    "error_message" varchar NULL,
    "downloaded_at" timestamptz NOT NULL,
    "plugin_id" uuid NOT NULL,
    PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "downloadlog_plugin_id_downloaded_at" ON "download_logs" ("plugin_id", "downloaded_at");
CREATE INDEX IF NOT EXISTS "downloadlog_downloaded_at" ON "download_logs" ("downloaded_at");
CREATE INDEX IF NOT EXISTS "downloadlog_success_downloaded_at" ON "download_logs" ("success", "downloaded_at");
ALTER TABLE "download_logs" ADD CONSTRAINT "download_logs_plugins_download_logs" FOREIGN KEY ("plugin_id") REFERENCES "plugins" ("id") ON DELETE NO ACTION;

-- sync_jobs (Phase 1: new table)
CREATE TABLE IF NOT EXISTS "sync_jobs" (
    "id" uuid NOT NULL,
    "trigger_type" varchar NOT NULL DEFAULT 'manual',
    "status" varchar NOT NULL DEFAULT 'pending',
    "target_ref" varchar NULL,
    "error_message" text NULL,
    "started_at" timestamptz NULL,
    "finished_at" timestamptz NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    "plugin_id" uuid NOT NULL,
    PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "syncjob_plugin_id_created_at" ON "sync_jobs" ("plugin_id", "created_at");
CREATE INDEX IF NOT EXISTS "syncjob_status_created_at" ON "sync_jobs" ("status", "created_at");
CREATE INDEX IF NOT EXISTS "syncjob_trigger_type_created_at" ON "sync_jobs" ("trigger_type", "created_at");
CREATE INDEX IF NOT EXISTS "syncjob_plugin_id_status_trigger_type_created_at" ON "sync_jobs" ("plugin_id", "status", "trigger_type", "created_at");
ALTER TABLE "sync_jobs" ADD CONSTRAINT "sync_jobs_plugins_sync_jobs" FOREIGN KEY ("plugin_id") REFERENCES "plugins" ("id") ON DELETE NO ACTION;
