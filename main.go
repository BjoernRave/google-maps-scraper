package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gosom/google-maps-scraper/runner"
	"github.com/gosom/google-maps-scraper/runner/databaserunner"
	"github.com/gosom/google-maps-scraper/runner/filerunner"
	"github.com/gosom/google-maps-scraper/runner/installplaywright"
	"github.com/gosom/google-maps-scraper/runner/lambdaaws"
	"github.com/gosom/google-maps-scraper/runner/webrunner"
	"github.com/gosom/google-maps-scraper/web/handlers"
	"github.com/gosom/google-maps-scraper/postgres"
	"github.com/gosom/google-maps-scraper/web/server"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	runner.Banner()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan

		log.Println("Received signal, shutting down...")

		cancel()
	}()

	cfg := runner.ParseConfig()

	// Initialize the web server first
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Initialize database connection
	db, err := sql.Open("pgx", cfg.Dsn)
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize provider
	provider := postgres.NewProvider(db)

	// Initialize job handler
	jobHandler := handlers.NewJobHandler(provider, logger)

	// Start web server in a goroutine
	go func() {
		srv := server.New(jobHandler, logger)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
			cancel()
		}
	}()

	// Start the scraper runner
	runnerInstance, err := runnerFactory(cfg)
	if err != nil {
		cancel()
		os.Stderr.WriteString(err.Error() + "\n")
		runner.Telemetry().Close()
		os.Exit(1)
	}

	if err := runnerInstance.Run(ctx); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		_ = runnerInstance.Close(ctx)
		runner.Telemetry().Close()
		cancel()
		os.Exit(1)
	}

	_ = runnerInstance.Close(ctx)
	runner.Telemetry().Close()
	cancel()
	os.Exit(0)
}

func runnerFactory(cfg *runner.Config) (runner.Runner, error) {
	switch cfg.RunMode {
	case runner.RunModeFile:
		return filerunner.New(cfg)
	case runner.RunModeDatabase, runner.RunModeDatabaseProduce:
		return databaserunner.New(cfg)
	case runner.RunModeInstallPlaywright:
		return installplaywright.New(cfg)
	case runner.RunModeWeb:
		return webrunner.New(cfg)
	case runner.RunModeAwsLambda:
		return lambdaaws.New(cfg)
	case runner.RunModeAwsLambdaInvoker:
		return lambdaaws.NewInvoker(cfg)
	default:
		return nil, fmt.Errorf("%w: %d", runner.ErrInvalidRunMode, cfg.RunMode)
	}
}
