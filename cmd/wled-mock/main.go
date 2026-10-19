package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/stuttgart-things/homerun2-light-catcher/internal/config"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/mock"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	// "/json" itself is the WLED API, which stays at the root as well.
	basePath, err := config.LoadBasePath()
	if err == nil && basePath == "/json" {
		err = errors.New(`invalid BASE_PATH "/json": the WLED API is served there`)
	}
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	s := mock.NewServer(version, commit, date)
	s.SetBasePath(basePath)
	s.Run(port)
}
