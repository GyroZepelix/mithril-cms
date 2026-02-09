-- 000001_initial_schema.down.sql
-- Drops all system tables in reverse dependency order.

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS media;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS admins;
DROP TABLE IF EXISTS content_types;
