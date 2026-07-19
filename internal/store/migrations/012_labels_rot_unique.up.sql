-- Add unique constraint to labels table so ON CONFLICT (tenant_id, dataset_id) works
CREATE UNIQUE INDEX IF NOT EXISTS idx_labels_dataset_unique ON labels(tenant_id, dataset_id);

-- Add unique constraint to rot_data so ON CONFLICT DO NOTHING works
CREATE UNIQUE INDEX IF NOT EXISTS idx_rot_data_unique ON rot_data(tenant_id, category, dataset_id);
