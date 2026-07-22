package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/securelens/securelens/internal/store"
)

// Scheduler handles job scheduling with distributed locking and retry support
type Scheduler struct {
	db       *store.DB
	workerID string
	producer JobProducer
	interval time.Duration
	mu       sync.Mutex
	running  bool
	stopCh   chan struct{}
}

// JobProducer interface for queueing jobs (implemented by Kafka)
type JobProducer interface {
	Produce(ctx context.Context, topic, key string, value any) error
}

// New creates a new job scheduler
func New(db *store.DB, producer JobProducer) *Scheduler {
	hostname, _ := os.Hostname()
	workerID := fmt.Sprintf("%s-%s", hostname, uuid.New().String()[:8])

	return &Scheduler{
		db:       db,
		workerID: workerID,
		producer: producer,
		interval: 30 * time.Second,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the scheduler loop
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	log.Info().
		Str("worker_id", s.workerID).
		Dur("interval", s.interval).
		Msg("Job scheduler started")

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run immediately on start
	s.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Job scheduler stopping (context cancelled)")
			return
		case <-s.stopCh:
			log.Info().Msg("Job scheduler stopping (stop signal)")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		close(s.stopCh)
		s.running = false
	}
}

// tick processes one scheduler cycle
func (s *Scheduler) tick(ctx context.Context) {
	// Release stale locks (jobs locked > 10 minutes without progress)
	s.releaseStaleJobs(ctx)

	// Process due jobs
	s.processDueJobs(ctx)

	// Retry failed jobs
	s.retryFailedJobs(ctx)
}

// releaseStaleJobs unlocks jobs that have been locked too long (worker crashed)
func (s *Scheduler) releaseStaleJobs(ctx context.Context) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE jobs 
		SET locked_by = NULL, 
		    locked_at = NULL,
		    status = CASE 
		        WHEN status = 'running' THEN 'failed'
		        ELSE status 
		    END,
		    updated_at = NOW()
		WHERE locked_by IS NOT NULL 
		  AND locked_at < NOW() - INTERVAL '10 minutes'`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to release stale job locks")
		return
	}
	if n, _ := result.RowsAffected(); n > 0 {
		log.Warn().Int64("count", n).Msg("Released stale job locks")
	}
}

// processDueJobs finds and queues jobs that are due to run
func (s *Scheduler) processDueJobs(ctx context.Context) {
	// Use SELECT FOR UPDATE SKIP LOCKED for distributed locking
	// This ensures only one worker picks up each job
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, tenant_id, name, type, schedule, config, max_retries, timeout_seconds
		FROM jobs 
		WHERE status = 'scheduled' 
		  AND (next_run IS NULL OR next_run <= NOW())
		  AND locked_by IS NULL
		ORDER BY next_run NULLS FIRST
		LIMIT 10
		FOR UPDATE SKIP LOCKED`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query due jobs")
		return
	}
	defer rows.Close()

	var jobsToProcess []struct {
		ID             string
		TenantID       string
		Name           string
		Type           string
		Schedule       string
		Config         json.RawMessage
		MaxRetries     int
		TimeoutSeconds int
	}

	for rows.Next() {
		var j struct {
			ID             string
			TenantID       string
			Name           string
			Type           string
			Schedule       string
			Config         json.RawMessage
			MaxRetries     int
			TimeoutSeconds int
		}
		if err := rows.Scan(&j.ID, &j.TenantID, &j.Name, &j.Type, &j.Schedule, &j.Config, &j.MaxRetries, &j.TimeoutSeconds); err != nil {
			log.Error().Err(err).Msg("Failed to scan job row")
			continue
		}
		jobsToProcess = append(jobsToProcess, j)
	}

	for _, job := range jobsToProcess {
		s.queueJob(ctx, job.ID, job.TenantID, job.Name, job.Type, job.Schedule, job.Config, 1, job.TimeoutSeconds)
	}
}

// retryFailedJobs finds failed jobs that should be retried
func (s *Scheduler) retryFailedJobs(ctx context.Context) {
	// Find jobs that failed but haven't exceeded max retries
	// Use exponential backoff: wait 2^attempt minutes before retry
	rows, err := s.db.QueryContext(ctx, `
		SELECT j.id, j.tenant_id, j.name, j.type, j.schedule, j.config, j.max_retries, j.timeout_seconds,
		       COALESCE(e.attempt, 0) as last_attempt
		FROM jobs j
		LEFT JOIN LATERAL (
			SELECT attempt FROM job_executions 
			WHERE job_id = j.id 
			ORDER BY created_at DESC 
			LIMIT 1
		) e ON true
		WHERE j.status = 'failed'
		  AND j.locked_by IS NULL
		  AND COALESCE(e.attempt, 0) < j.max_retries
		  AND j.updated_at < NOW() - (INTERVAL '1 minute' * POWER(2, COALESCE(e.attempt, 0)))
		ORDER BY j.updated_at
		LIMIT 5
		FOR UPDATE OF j SKIP LOCKED`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query failed jobs for retry")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var j struct {
			ID             string
			TenantID       string
			Name           string
			Type           string
			Schedule       string
			Config         json.RawMessage
			MaxRetries     int
			TimeoutSeconds int
			LastAttempt    int
		}
		if err := rows.Scan(&j.ID, &j.TenantID, &j.Name, &j.Type, &j.Schedule, &j.Config, &j.MaxRetries, &j.TimeoutSeconds, &j.LastAttempt); err != nil {
			log.Error().Err(err).Msg("Failed to scan retry job row")
			continue
		}

		log.Info().
			Str("job_id", j.ID).
			Str("job_name", j.Name).
			Int("attempt", j.LastAttempt+1).
			Int("max_retries", j.MaxRetries).
			Msg("Retrying failed job")

		s.queueJob(ctx, j.ID, j.TenantID, j.Name, j.Type, j.Schedule, j.Config, j.LastAttempt+1, j.TimeoutSeconds)
	}
}

// queueJob locks a job and queues it for execution
func (s *Scheduler) queueJob(ctx context.Context, jobID, tenantID, name, jobType, schedule string, config json.RawMessage, attempt, timeoutSeconds int) {
	// Lock the job
	result, err := s.db.ExecContext(ctx, `
		UPDATE jobs 
		SET locked_by = $1, 
		    locked_at = NOW(),
		    status = 'pending',
		    updated_at = NOW()
		WHERE id = $2 
		  AND locked_by IS NULL`,
		s.workerID, jobID)
	if err != nil {
		log.Error().Err(err).Str("job_id", jobID).Msg("Failed to lock job")
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		// Another worker grabbed it
		return
	}

	// Create execution record
	executionID := uuid.New().String()
	now := time.Now()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO job_executions (id, tenant_id, job_id, status, started_at, attempt, worker_id, created_at, updated_at)
		 VALUES ($1, $2, $3, 'pending', $4, $5, $6, $4, $4)`,
		executionID, tenantID, jobID, now, attempt, s.workerID)
	if err != nil {
		log.Error().Err(err).Str("job_id", jobID).Msg("Failed to create execution record")
		// Unlock the job
		s.db.ExecContext(ctx, `UPDATE jobs SET locked_by = NULL, locked_at = NULL, status = 'scheduled' WHERE id = $1`, jobID)
		return
	}

	// Queue for execution via Kafka
	err = s.producer.Produce(ctx, "job-executions", tenantID, map[string]any{
		"job_id":          jobID,
		"execution_id":    executionID,
		"tenant_id":       tenantID,
		"type":            jobType,
		"config":          config,
		"attempt":         attempt,
		"timeout_seconds": timeoutSeconds,
		"worker_id":       s.workerID,
	})
	if err != nil {
		log.Error().Err(err).Str("job_id", jobID).Msg("Failed to queue job execution")
		// Mark execution as failed
		s.db.ExecContext(ctx,
			`UPDATE job_executions SET status = 'failed', error = $1, completed_at = NOW() WHERE id = $2`,
			err.Error(), executionID)
		// Unlock and mark job as failed
		s.db.ExecContext(ctx, `UPDATE jobs SET locked_by = NULL, locked_at = NULL, status = 'failed' WHERE id = $1`, jobID)
		return
	}

	log.Info().
		Str("job_id", jobID).
		Str("job_name", name).
		Str("job_type", jobType).
		Str("execution_id", executionID).
		Int("attempt", attempt).
		Msg("Job queued for execution")
}

// CompleteJob marks a job as completed and schedules next run
func (s *Scheduler) CompleteJob(ctx context.Context, jobID, executionID, status, errorMsg string, result map[string]any, durationMs int) {
	now := time.Now()

	// Update execution record
	resultJSON, _ := json.Marshal(result)
	var errPtr *string
	if errorMsg != "" {
		errPtr = &errorMsg
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE job_executions 
		 SET status = $1, 
		     completed_at = $2, 
		     duration_ms = $3, 
		     result = $4, 
		     error = $5,
		     updated_at = $2
		 WHERE id = $6`,
		status, now, durationMs, resultJSON, errPtr, executionID)
	if err != nil {
		log.Error().Err(err).Str("execution_id", executionID).Msg("Failed to update execution record")
	}

	// Get job details for next run calculation
	var job struct {
		Schedule   string `db:"schedule"`
		MaxRetries int    `db:"max_retries"`
	}
	err = s.db.GetContext(ctx, &job, `SELECT schedule, max_retries FROM jobs WHERE id = $1`, jobID)
	if err != nil {
		log.Error().Err(err).Str("job_id", jobID).Msg("Failed to get job for completion")
		return
	}

	// Calculate next run time
	nextRun := CalculateNextRun(job.Schedule)

	// Determine final job status
	finalStatus := "scheduled"
	if status == "failed" {
		// Check if we've exceeded retries
		var attempt int
		s.db.GetContext(ctx, &attempt, `SELECT attempt FROM job_executions WHERE id = $1`, executionID)
		if attempt >= job.MaxRetries {
			finalStatus = "failed"
		} else {
			finalStatus = "failed" // Will be retried by retryFailedJobs
		}
	}

	// If no schedule, mark as completed (one-time job)
	if job.Schedule == "" && status == "completed" {
		finalStatus = "completed"
		nextRun = nil
	}

	// Update job status and unlock
	_, err = s.db.ExecContext(ctx, `
		UPDATE jobs 
		SET status = $1,
		    last_run = $2,
		    next_run = $3,
		    locked_by = NULL,
		    locked_at = NULL,
		    updated_at = $2
		WHERE id = $4`,
		finalStatus, now, nextRun, jobID)
	if err != nil {
		log.Error().Err(err).Str("job_id", jobID).Msg("Failed to update job status")
	}

	log.Info().
		Str("job_id", jobID).
		Str("execution_id", executionID).
		Str("status", status).
		Str("final_status", finalStatus).
		Int("duration_ms", durationMs).
		Msg("Job execution completed")
}

// CalculateNextRun calculates the next run time based on schedule
// Supports: @hourly, @daily, @weekly, duration strings (1h, 30m), and basic cron (minute hour day month weekday)
func CalculateNextRun(schedule string) *time.Time {
	if schedule == "" {
		return nil
	}

	now := time.Now()
	var next time.Time

	switch schedule {
	case "@hourly":
		next = now.Add(time.Hour)
	case "@daily":
		next = now.Add(24 * time.Hour)
	case "@weekly":
		next = now.Add(7 * 24 * time.Hour)
	case "@monthly":
		next = now.AddDate(0, 1, 0)
	default:
		// Try duration format (e.g., "1h", "30m", "24h")
		if d, err := time.ParseDuration(schedule); err == nil {
			next = now.Add(d)
		} else if nextCron := parseCronSchedule(schedule, now); nextCron != nil {
			next = *nextCron
		} else {
			// Default to daily if unparseable
			next = now.Add(24 * time.Hour)
		}
	}

	return &next
}

// parseCronSchedule parses a basic cron expression (minute hour day month weekday)
// Returns the next run time or nil if invalid
func parseCronSchedule(schedule string, now time.Time) *time.Time {
	parts := strings.Fields(schedule)
	if len(parts) != 5 {
		return nil
	}

	minute := parseCronField(parts[0], 0, 59)
	hour := parseCronField(parts[1], 0, 23)
	day := parseCronField(parts[2], 1, 31)
	month := parseCronField(parts[3], 1, 12)
	weekday := parseCronField(parts[4], 0, 6)

	if minute == nil || hour == nil {
		return nil
	}

	// Simple implementation: find next occurrence
	// Start from next minute
	candidate := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
	candidate = candidate.Add(time.Minute)

	// Search up to 366 days ahead
	for i := 0; i < 366*24*60; i++ {
		if matchesCron(candidate, minute, hour, day, month, weekday) {
			return &candidate
		}
		candidate = candidate.Add(time.Minute)
	}

	return nil
}

func parseCronField(field string, min, max int) []int {
	if field == "*" {
		result := make([]int, max-min+1)
		for i := range result {
			result[i] = min + i
		}
		return result
	}

	// Handle */n (every n)
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err != nil || step <= 0 {
			return nil
		}
		var result []int
		for i := min; i <= max; i += step {
			result = append(result, i)
		}
		return result
	}

	// Handle single value
	val, err := strconv.Atoi(field)
	if err != nil {
		return nil
	}
	if val < min || val > max {
		return nil
	}
	return []int{val}
}

func matchesCron(t time.Time, minute, hour, day, month, weekday []int) bool {
	return contains(minute, t.Minute()) &&
		contains(hour, t.Hour()) &&
		(day == nil || contains(day, t.Day())) &&
		(month == nil || contains(month, int(t.Month()))) &&
		(weekday == nil || contains(weekday, int(t.Weekday())))
}

func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
