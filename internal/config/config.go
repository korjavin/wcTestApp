package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	ServerHost string
	ServerPort int
	ServerURL  string // External URL for the server (for QR codes)

	// Relay configuration
	RelayHost string
	RelayPort int

	// Web configuration
	StaticDir   string
	TemplateDir string

	// TLS configuration
	EnableTLS bool
	CertFile  string
	KeyFile   string

	// Debug mode
	Debug bool
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		ServerHost:  "0.0.0.0", // Listen on all interfaces by default
		ServerPort:  8080,
		ServerURL:   "", // Will be auto-generated if not provided
		RelayHost:   "0.0.0.0",
		RelayPort:   8081,
		StaticDir:   "web/static",
		TemplateDir: "web/templates",
		EnableTLS:   false,
		CertFile:    "certs/server.crt",
		KeyFile:     "certs/server.key",
		Debug:       true,
	}
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	config := DefaultConfig()

	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.ServerHost = host
	}

	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.ServerPort = p
		}
	}

	if url := os.Getenv("SERVER_URL"); url != "" {
		config.ServerURL = url
	}

	if host := os.Getenv("RELAY_HOST"); host != "" {
		config.RelayHost = host
	}

	if port := os.Getenv("RELAY_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.RelayPort = p
		}
	}

	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		config.StaticDir = dir
	}

	if dir := os.Getenv("TEMPLATE_DIR"); dir != "" {
		config.TemplateDir = dir
	}

	if enableTLS := os.Getenv("ENABLE_TLS"); enableTLS != "" {
		if t, err := strconv.ParseBool(enableTLS); err == nil {
			config.EnableTLS = t
		}
	}

	if certFile := os.Getenv("CERT_FILE"); certFile != "" {
		config.CertFile = certFile
	}

	if keyFile := os.Getenv("KEY_FILE"); keyFile != "" {
		config.KeyFile = keyFile
	}

	if debug := os.Getenv("DEBUG"); debug != "" {
		if d, err := strconv.ParseBool(debug); err == nil {
			config.Debug = d
		}
	}

	// If SERVER_URL is not provided, generate it based on host and port
	if config.ServerURL == "" {
		protocol := "http"
		if config.EnableTLS {
			protocol = "https"
		}

		host := config.ServerHost
		if host == "0.0.0.0" {
			// Try to get the machine's hostname or IP
			hostname, err := os.Hostname()
			if err == nil {
				host = hostname
			} else {
				host = "localhost" // Fallback
			}
		}

		config.ServerURL = fmt.Sprintf("%s://%s:%d", protocol, host, config.ServerPort)
	}

	return config
}

// ServerAddress returns the full server address
func (c *Config) ServerAddress() string {
	return fmt.Sprintf("%s:%d", c.ServerHost, c.ServerPort)
}

// RelayAddress returns the full relay address
func (c *Config) RelayAddress() string {
	return fmt.Sprintf("%s:%d", c.RelayHost, c.RelayPort)
}

// ExternalURL returns the external URL for the server
func (c *Config) ExternalURL() string {
	return c.ServerURL
}

// RelayWebSocketURL returns the WebSocket URL for the relay server
func (c *Config) RelayWebSocketURL() string {
	protocol := "wss" //dirty for caddy
	if c.EnableTLS {
		protocol = "wss"
	}

	// Extract the host and port from the server URL
	serverURL := c.ServerURL
	if strings.HasPrefix(serverURL, "http://") {
		serverURL = strings.TrimPrefix(serverURL, "http://")
	} else if strings.HasPrefix(serverURL, "https://") {
		serverURL = strings.TrimPrefix(serverURL, "https://")
	}

	// Remove any path
	if idx := strings.Index(serverURL, "/"); idx != -1 {
		serverURL = serverURL[:idx]
	}

	return fmt.Sprintf("%s://%s/relay", protocol, serverURL)
}
