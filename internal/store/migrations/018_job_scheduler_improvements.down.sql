-- Remove job scheduler improvement columns
ALTER TABLE jobs DROP COLUMN IF EXISTS max_retries;
ALTER TABLE jobs DROP COLUMN IF EXISTS timeout_seconds;
ALTER TABLE jobs DROP COLUMN IF EXISTS locked_by;
ALTER TABLE jobs DROP COLUMN IF EXISTS locked_at;

ALTER TABLE job_executions DROP COLUMN IF EXISTS attempt;
ALTER TABLE job_executions DROP COLUMN IF EXISTS worker_id;

DROP INDEX IF EXISTS idx_jobs_scheduler;
DROP INDEX IF EXISTS idx_jobs_locked;
