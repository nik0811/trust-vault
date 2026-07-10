-- Rollback enterprise features
-- Migration: 002_enterprise_features.down.sql

DROP TABLE IF EXISTS retention_violations;
DROP TABLE IF EXISTS retention_policies;
DROP TABLE IF EXISTS review_queue;
DROP TABLE IF EXISTS duplicate_groups;
DROP TABLE IF EXISTS data_flows;
DROP TABLE IF EXISTS integration_logs;
DROP TABLE IF EXISTS model_lineage;
DROP TABLE IF EXISTS playbooks;
DROP TABLE IF EXISTS ropa;
DROP TABLE IF EXISTS label_rules;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS remediation_actions;
DROP TABLE IF EXISTS classification_models;
