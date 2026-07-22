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
	"github.com/securelens/securelens/internal/api"
	"github.com/securelens/securelens/internal/events"
	"github.com/securelens/securelens/internal/external"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/scheduler"
	"github.com/securelens/securelens/internal/store"
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

	log.Info().Str("mode", *mode).Msg("Starting SecureLens")

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

	// Start scan watchdog: marks scans stuck in 'running' for >30 minutes as failed.
	go runScanWatchdog(ctx, db)

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
	log.Info().Msg("Starting SecureLens worker")

	// Create cancellable context for worker
	workerCtx, workerCancel := context.WithCancel(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start all consumers in parallel
	go kafka.ConsumeClassificationJobs(workerCtx, db)
	go kafka.ConsumeScanJobs(workerCtx, db)
	go kafka.ConsumeJobExecutions(workerCtx, db)

	// Start production-grade job scheduler with distributed locking
	jobScheduler := scheduler.New(db, kafka)
	go jobScheduler.Start(workerCtx)

	log.Info().Msg("Worker started - consuming from: raw-data-chunks, scan-jobs, job-executions")

	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
	log.Info().Msg("Worker shutting down gracefully...")

	// Stop the job scheduler first
	jobScheduler.Stop()

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

// runScanWatchdog periodically marks scans stuck in 'running' for over 30 minutes as failed.
// This handles cases where the ingestion sidecar is unreachable or crashes without calling back.
func runScanWatchdog(ctx context.Context, db *store.DB) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := db.ExecContext(ctx,
				`UPDATE scan_logs
				 SET status = 'failed',
				     message = 'Timed out: no completion callback received within 30 minutes',
				     completed_at = NOW()
				 WHERE status = 'running'
				   AND started_at < NOW() - INTERVAL '30 minutes'`)
			if err != nil {
				log.Error().Err(err).Msg("scan watchdog: failed to mark timed-out scans")
				continue
			}
			if n, _ := result.RowsAffected(); n > 0 {
				log.Warn().Int64("count", n).Msg("scan watchdog: marked timed-out scans as failed")
			}
		}
	}
}
