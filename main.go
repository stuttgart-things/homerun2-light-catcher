package main

import (
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	homerun "github.com/stuttgart-things/homerun-library/v2"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/banner"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/catcher"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/config"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/handlers"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/mock"
)

// Build-time variables set via ldflags.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	banner.Show()
	config.SetupLogging()

	slog.Info("starting homerun2-light-catcher",
		"version", version,
		"commit", commit,
		"date", date,
		"go", runtime.Version(),
	)

	profilePath := homerun.GetEnv("PROFILE_PATH", "profile.yaml")
	healthPort := homerun.GetEnv("HEALTH_PORT", "8080")
	mockWLED := homerun.GetEnv("MOCK_WLED", "")

	// Start embedded WLED mock if requested
	if mockWLED != "" {
		mockPort := homerun.GetEnv("MOCK_WLED_PORT", "9090")
		slog.Info("starting embedded WLED mock", "port", mockPort)
		mockServer := mock.NewServer()
		go mockServer.Run(mockPort)
	}

	// Health endpoint
	buildInfo := handlers.BuildInfo{Version: version, Commit: commit, Date: date}
	go func() {
		mux := http.NewServeMux()
		healthHandler := handlers.NewHealthHandler(buildInfo)
		mux.HandleFunc("/health", healthHandler)
		mux.HandleFunc("/healthz", healthHandler)

		slog.Info("health server starting", "port", healthPort)
		if err := http.ListenAndServe(":"+healthPort, mux); err != nil {
			slog.Error("health server error", "error", err)
		}
	}()

	// Build message handlers
	msgHandlers := []catcher.MessageHandler{
		catcher.LogHandler(),
		catcher.LightHandler(profilePath),
	}

	// Create Redis catcher
	redisConfig := config.LoadRedisConfig()
	consumerGroup := homerun.GetEnv("CONSUMER_GROUP", "homerun2-light-catcher")
	consumerName := homerun.GetEnv("CONSUMER_NAME", "")

	c, err := catcher.NewRedisCatcher(redisConfig, consumerGroup, consumerName, msgHandlers...)
	if err != nil {
		slog.Error("failed to create catcher", "error", err)
		os.Exit(1)
	}

	slog.Info("catcher configured",
		"redis_addr", redisConfig.Addr,
		"redis_port", redisConfig.Port,
		"stream", redisConfig.Stream,
		"consumer_group", consumerGroup,
		"profile_path", profilePath,
	)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	if errCh := c.Errors(); errCh != nil {
		go func() {
			for err := range errCh {
				slog.Error("consumer error", "error", err)
			}
		}()
	}

	go func() {
		<-quit
		slog.Info("shutting down catcher")
		c.Shutdown()
	}()

	slog.Info("catcher running, waiting for messages...")
	c.Run()

	slog.Info("catcher exited gracefully")
}
