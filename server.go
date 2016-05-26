package main

import (
	"log"
	"net/http"
	"os"

	_ "net/http/pprof"

	"github.com/fukata/golang-stats-api-handler"
)

const (
	// DefaultPort is default port to listen
	DefaultPort = "8080"

	// EnvPort is environmental variable to change port to listen
	EnvPort = "PORT"
)

// To run nozzle as PaaS application, we need server.
// It would be better to have varz endpoint than just returning
// hello-world thing.
type VarzServer struct {
	Logger *log.Logger
}

// Start starts listening.
func (s *VarzServer) Start() {

	http.HandleFunc("/varz", stats_api.Handler)

	port := DefaultPort
	if p := os.Getenv(EnvPort); p != "" {
		port = p
	}

	s.Logger.Printf("[INFO] Start varz-server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		s.Logger.Printf("[ERROR] Failed to start varz-server: %s", err)
	}
}
