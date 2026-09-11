package main

import (
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	homerun "github.com/stuttgart-things/homerun-library/v4"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/banner"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/catcher"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/config"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/dashboard"
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
		mockServer := mock.NewServer(version, commit, date)
		go mockServer.Run(mockPort)
	}

	// Event tracker for dashboard
	tracker := dashboard.NewEventTracker()
	dash := dashboard.NewHandler(tracker, version, commit, date)

	// HTTP server with dashboard + health endpoints
	go func() {
		mux := http.NewServeMux()
		healthHandler := handlers.NewHealthHandler(handlers.BuildInfo{
			Version: version, Commit: commit, Date: date,
		})
		dash.RegisterRoutes(mux)
		mux.HandleFunc("/health", healthHandler)
		mux.HandleFunc("/healthz", healthHandler)

		slog.Info("http server starting", "port", healthPort)
		if err := http.ListenAndServe(":"+healthPort, mux); err != nil {
			slog.Error("http server error", "error", err)
		}
	}()

	maxMessageAge := mustLoadDuration(config.LoadMaxMessageAge)

	// Build message handlers
	msgHandlers := []catcher.MessageHandler{
		catcher.LogHandler(),
		catcher.LightHandler(profilePath, maxMessageAge, tracker),
	}

	// Create Redis catcher
	redisConfig := config.LoadRedisConfig()
	streams := config.LoadStreams()
	consumerGroup := homerun.GetEnv("CONSUMER_GROUP", "homerun2-light-catcher")
	consumerName := homerun.GetEnv("CONSUMER_NAME", "")
	consumerStartID := homerun.GetEnv("CONSUMER_START_ID", catcher.DefaultStartID)

	waitForRedis(redisConfig)

	c, err := catcher.NewRedisCatcher(redisConfig, streams, consumerGroup, consumerName, consumerStartID, msgHandlers...)
	if err != nil {
		slog.Error("failed to create catcher", "error", err)
		os.Exit(1)
	}

	slog.Info("catcher configured",
		"redis_addr", redisConfig.Addr,
		"redis_port", redisConfig.Port,
		"streams", streams,
		"consumer_group", consumerGroup,
		"consumer_start_id", consumerStartID,
		"profile_path", profilePath,
		"max_message_age", maxMessageAge.String(),
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

// mustLoadDuration returns a duration setting, or exits on an invalid value.
func mustLoadDuration(load func() (time.Duration, error)) time.Duration {
	d, err := load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	return d
}

// waitForRedis blocks until Redis answers, or exits after
// REDIS_STARTUP_TIMEOUT. The consumer's preflight dials Redis exactly once, so
// without this a Redis that is still starting makes the catcher exit (#59).
func waitForRedis(rc homerun.RedisConfig) {
	timeout := mustLoadDuration(config.LoadRedisStartupTimeout)
	if err := catcher.WaitForRedis(rc, timeout); err != nil {
		slog.Error("redis not reachable",
			"error", err,
			"addr", rc.Addr,
			"port", rc.Port,
			"startup_timeout", timeout.String(),
		)
		os.Exit(1)
	}
}
