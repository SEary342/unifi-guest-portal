// Package config provides functionality for loading and managing environment variables for configuration.
// It loads values from a .env file (if available) and environment variables, returning them in a structured Config object.
package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config represents the application configuration loaded from environment variables.
// It includes the Unifi credentials, server URL, site, session duration, TLS setting, and application port.
type Config struct {
	Username   string // Username for Unifi authentication.
	Password   string // Password for Unifi authentication.
	URL        string // URL of the Unifi controller.
	Site       string // Site for Unifi controller access.
	Duration   int    // Session duration for guest authorization in minutes.
	DisableTLS bool   // Flag to disable TLS verification for Unifi connection.
	Port       string // Port to serve the application on.
}

// LoadEnv loads configuration values from the environment variables and returns a Config struct.
// It attempts to load a .env file if present, then retrieves values from the environment.
// The function returns the populated Config struct and an error (if any) encountered during the process.
//
// The function handles the following environment variables:
// - UNIFI_USERNAME: Unifi controller username
// - UNIFI_PASSWORD: Unifi controller password
// - UNIFI_URL: Unifi controller URL
// - UNIFI_SITE: Unifi site to use
// - UNIFI_DURATION: Duration of guest session in minutes
// - DISABLE_TLS: Flag to disable TLS verification (default: false)
// - PORT: Port to run the application on
//
// If the .env file is not found, a warning is logged, and the application continues without it.
// If any of the variables cannot be parsed, an error is returned.
func LoadEnv() (Config, error) {
	var cfg Config

	// Attempt to load environment variables from a .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found. Continuing without it.")
	}

	// Load the environment variables into the Config struct fields
	cfg.Username = os.Getenv("UNIFI_USERNAME")
	cfg.Password = os.Getenv("UNIFI_PASSWORD")
	cfg.URL = os.Getenv("UNIFI_URL")
	cfg.Site = os.Getenv("UNIFI_SITE")
	cfg.Port = os.Getenv("PORT")

	// Parse the UNIFI_DURATION environment variable into an integer
	duration, err := strconv.Atoi(os.Getenv("UNIFI_DURATION"))
	if err != nil {
		return cfg, fmt.Errorf("error loading duration from env file")
	}
	cfg.Duration = duration

	// Parse the DISABLE_TLS environment variable into a boolean
	disableTLS, err := strconv.ParseBool(os.Getenv("DISABLE_TLS"))
	if err != nil {
		cfg.DisableTLS = false // Default to false if the value is invalid
	} else {
		cfg.DisableTLS = disableTLS
	}

	return cfg, nil
}
