-- Add action_type to remediation_actions for richer classification
ALTER TABLE remediation_actions ADD COLUMN IF NOT EXISTS action_type VARCHAR(50) DEFAULT '';
