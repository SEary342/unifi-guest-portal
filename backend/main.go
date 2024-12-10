// Package main serves as the entry point for the Unifi Guest Portal application.
// It initializes the cache purge routine, loads environment configuration, and starts the HTTP server.
package main

import (
	"backend/cache"
	"backend/config"
	"backend/router"
	"log"
	"time"
)

func main() {
	// Start a goroutine to periodically purge expired cache entries.
	// The cache is purged every 30 seconds to maintain optimal performance.
	go cache.PurgeCacheEvery(30 * time.Second)

	// Load application configuration from environment variables.
	// The configuration includes server settings, Unifi credentials, and other runtime options.
	cfg, err := config.LoadEnv()
	if err != nil {
		// Log an error and terminate the application if the configuration fails to load.
		log.Fatal(err)
	}

	// Set up and start the HTTP server using the loaded configuration.
	router.SetupServer(cfg)
}
