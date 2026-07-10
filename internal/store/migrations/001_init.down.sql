-- TrustVault Database Schema Rollback
-- Migration: 001_init.down.sql

DROP TABLE IF EXISTS rot_data;
DROP TABLE IF EXISTS integrations;
DROP TABLE IF EXISTS feedback;
DROP TABLE IF EXISTS labels;
DROP TABLE IF EXISTS webhooks;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS dsars;
DROP TABLE IF EXISTS quality_scores;
DROP TABLE IF EXISTS gate_queries;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS classifications;
DROP TABLE IF EXISTS policies;
DROP TABLE IF EXISTS datasources;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
