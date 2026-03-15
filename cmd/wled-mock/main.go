package main

import (
	"os"

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

	s := mock.NewServer(version, commit, date)
	s.Run(port)
}
