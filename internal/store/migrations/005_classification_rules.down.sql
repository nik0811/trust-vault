-- Revert classification rules migration
ALTER TABLE datasources DROP COLUMN IF EXISTS sensitivity_label;
ALTER TABLE classifications DROP COLUMN IF EXISTS classification_source;
ALTER TABLE classifications DROP COLUMN IF EXISTS rule_id;
ALTER TABLE classifications DROP COLUMN IF EXISTS label_id;

DROP INDEX IF EXISTS idx_classification_rules_type;
DROP INDEX IF EXISTS idx_classification_rules_active;
DROP INDEX IF EXISTS idx_classification_rules_tenant;
DROP TABLE IF EXISTS classification_rules;
