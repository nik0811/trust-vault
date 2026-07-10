package main

import (
	"flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/trustvault/trustvault/internal/store"
)

func main() {
	direction := flag.String("direction", "up", "Migration direction: up, down")
	steps := flag.Int("steps", 0, "Number of steps (0 = all)")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	db, err := store.NewDB(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	if err := store.RunMigrations(db, *direction, *steps); err != nil {
		log.Fatal().Err(err).Msg("Migration failed")
	}

	log.Info().Str("direction", *direction).Msg("Migrations completed")
}
