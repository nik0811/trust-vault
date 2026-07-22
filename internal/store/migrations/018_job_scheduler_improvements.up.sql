-- Add retry support and distributed locking columns to jobs table
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS max_retries INTEGER DEFAULT 3;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS timeout_seconds INTEGER DEFAULT 3600;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS locked_by VARCHAR(255);
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS locked_at TIMESTAMPTZ;

-- Add retry tracking to job_executions
ALTER TABLE job_executions ADD COLUMN IF NOT EXISTS attempt INTEGER DEFAULT 1;
ALTER TABLE job_executions ADD COLUMN IF NOT EXISTS worker_id VARCHAR(255);

-- Index for finding due jobs efficiently
CREATE INDEX IF NOT EXISTS idx_jobs_scheduler ON jobs(tenant_id, status, next_run) 
    WHERE status IN ('scheduled', 'failed');

-- Index for finding stuck jobs (locked but not updated)
CREATE INDEX IF NOT EXISTS idx_jobs_locked ON jobs(locked_at) WHERE locked_by IS NOT NULL;
