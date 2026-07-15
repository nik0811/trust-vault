DROP TABLE IF EXISTS consent_widget_configs;
ALTER TABLE datasources DROP COLUMN IF EXISTS country;
ALTER TABLE datasources DROP COLUMN IF EXISTS region;
DROP TABLE IF EXISTS residency_rules;
DROP TABLE IF EXISTS endpoint_agents;
