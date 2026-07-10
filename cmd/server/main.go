package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/trustvault/trustvault/internal/api"
	"github.com/trustvault/trustvault/internal/events"
	"github.com/trustvault/trustvault/internal/external"
	"github.com/trustvault/trustvault/internal/pkg"
	"github.com/trustvault/trustvault/internal/store"
)

func main() {
	mode := flag.String("mode", "gateway", "Run mode: gateway, worker")
	port := flag.String("port", "8080", "Server port")
	internalPort := flag.String("internal-port", "8099", "Internal admin port")
	requireJWT := flag.Bool("require-jwt-secret", false, "Fail if JWT_SECRET not set (use in production)")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Validate JWT secret in production mode
	if *requireJWT {
		pkg.MustInitJWTSecret()
	}

	log.Info().Str("mode", *mode).Msg("Starting TrustVault")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := store.NewDB(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Bootstrap superadmin from environment variables
	bootstrapSuperAdmin(ctx, db)

	kafka := external.NewKafka(os.Getenv("KAFKA_BROKERS"))
	defer kafka.Close()

	events.Start(ctx)

	switch *mode {
	case "gateway":
		runGateway(ctx, db, kafka, *port, *internalPort)
	case "worker":
		runWorker(ctx, db, kafka)
	default:
		log.Fatal().Str("mode", *mode).Msg("Unknown mode")
	}
}

// bootstrapSuperAdmin creates the superadmin user from environment variables if not exists
func bootstrapSuperAdmin(ctx context.Context, db *store.DB) {
	email := os.Getenv("SUPERADMIN_EMAIL")
	password := os.Getenv("SUPERADMIN_PASSWORD")
	name := os.Getenv("SUPERADMIN_NAME")
	if name == "" {
		name = "Super Admin"
	}

	if email == "" || password == "" {
		log.Warn().Msg("SUPERADMIN_EMAIL and SUPERADMIN_PASSWORD not set - skipping superadmin bootstrap")
		return
	}

	// Check if superadmin already exists
	var count int
	err := db.GetContext(ctx, &count, "SELECT COUNT(*) FROM users WHERE email = $1", email)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check for existing superadmin")
		return
	}
	if count > 0 {
		log.Info().Str("email", email).Msg("Superadmin already exists - skipping creation")
		return
	}

	// Ensure platform tenant exists
	var tenantID string
	err = db.GetContext(ctx, &tenantID, "SELECT id FROM tenants WHERE slug = 'platform'")
	if err != nil {
		// Create platform tenant
		err = db.QueryRowContext(ctx,
			`INSERT INTO tenants (name, slug, status) VALUES ('Platform', 'platform', 'active') RETURNING id`,
		).Scan(&tenantID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create platform tenant")
			return
		}
		log.Info().Str("tenant_id", tenantID).Msg("Created platform tenant")
	}

	// Hash password and create superadmin
	hash, err := pkg.HashPassword(password)
	if err != nil {
		log.Error().Err(err).Msg("Failed to hash superadmin password")
		return
	}

	_, err = db.ExecContext(ctx,
		`INSERT INTO users (tenant_id, email, password_hash, name, is_super_admin, status) 
		 VALUES ($1, $2, $3, $4, TRUE, 'active')`,
		tenantID, email, hash, name)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create superadmin user")
		return
	}

	log.Info().Str("email", email).Msg("Superadmin user created successfully")
}

func runGateway(ctx context.Context, db *store.DB, kafka *external.Kafka, port, internalPort string) {
	server := api.NewServer(db, kafka)

	// Start servers in goroutines
	go server.RunInternal(internalPort)
	go server.Run(port)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
	log.Info().Msg("Gracefully shutting down server (draining connections)...")

	// Give connections time to drain
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP servers (stops accepting new connections, waits for active ones)
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Error during server shutdown")
	}

	// Close database connections
	log.Info().Msg("Closing database connections...")
	if err := db.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing database")
	}

	// Close Kafka connections
	log.Info().Msg("Closing Kafka connections...")
	kafka.Close()

	log.Info().Msg("Server shutdown complete")
}

func runWorker(ctx context.Context, db *store.DB, kafka *external.Kafka) {
	log.Info().Msg("Starting TrustVault worker")

	// Create cancellable context for worker
	workerCtx, workerCancel := context.WithCancel(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start all consumers in parallel
	go kafka.ConsumeClassificationJobs(workerCtx, db)
	go kafka.ConsumeScanJobs(workerCtx, db)
	go kafka.ConsumeJobExecutions(workerCtx, db)

	// Start job scheduler
	go runJobScheduler(workerCtx, db, kafka)

	log.Info().Msg("Worker started - consuming from: raw-data-chunks, scan-jobs, job-executions")

	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
	log.Info().Msg("Worker shutting down gracefully...")

	// Cancel worker context to stop consuming
	workerCancel()

	// Give time for in-flight messages to complete
	time.Sleep(5 * time.Second)

	// Close connections
	log.Info().Msg("Closing database connections...")
	db.Close()

	log.Info().Msg("Closing Kafka connections...")
	kafka.Close()

	log.Info().Msg("Worker shutdown complete")
}

// runJobScheduler checks for scheduled jobs and triggers them
func runJobScheduler(ctx context.Context, db *store.DB, kafka *external.Kafka) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Info().Msg("Job scheduler started - checking every 30 seconds")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Job scheduler shutting down")
			return
		case <-ticker.C:
			processScheduledJobs(ctx, db, kafka)
		}
	}
}

// processScheduledJobs finds and triggers jobs that are due to run
func processScheduledJobs(ctx context.Context, db *store.DB, kafka *external.Kafka) {
	// Find jobs that are scheduled and due to run
	var jobs []store.Job
	err := db.SelectContext(ctx, &jobs,
		`SELECT * FROM jobs 
		 WHERE status = 'scheduled' 
		 AND (next_run IS NULL OR next_run <= NOW())
		 LIMIT 10`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch scheduled jobs")
		return
	}

	for _, job := range jobs {
		log.Info().
			Str("job_id", job.ID).
			Str("job_name", job.Name).
			Str("job_type", job.Type).
			Msg("Triggering scheduled job")

		// Queue job for execution
		err := kafka.Produce(ctx, "job-executions", job.TenantID, map[string]any{
			"job_id":    job.ID,
			"tenant_id": job.TenantID,
			"type":      job.Type,
			"config":    job.Config,
		})
		if err != nil {
			log.Error().Err(err).Str("job_id", job.ID).Msg("Failed to queue job execution")
			continue
		}

		// Calculate next run time based on schedule (cron)
		nextRun := calculateNextRun(job.Schedule)

		// Update job status to pending and set next_run
		_, err = db.ExecContext(ctx,
			`UPDATE jobs SET status = 'pending', next_run = $1, updated_at = NOW() WHERE id = $2`,
			nextRun, job.ID)
		if err != nil {
			log.Error().Err(err).Str("job_id", job.ID).Msg("Failed to update job next_run")
		}
	}
}

// calculateNextRun calculates the next run time based on a simple schedule
// Supports: @hourly, @daily, @weekly, or interval like "1h", "30m", "24h"
func calculateNextRun(schedule string) time.Time {
	now := time.Now()

	switch schedule {
	case "@hourly":
		return now.Add(time.Hour)
	case "@daily":
		return now.Add(24 * time.Hour)
	case "@weekly":
		return now.Add(7 * 24 * time.Hour)
	case "":
		// One-time job, no next run
		return time.Time{}
	default:
		// Try to parse as duration
		if d, err := time.ParseDuration(schedule); err == nil {
			return now.Add(d)
		}
		// Default to daily if unparseable
		return now.Add(24 * time.Hour)
	}
}
