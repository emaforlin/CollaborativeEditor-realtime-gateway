package config

import (
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	singleConfig *Config
	once         sync.Once
)

// Config holds all configuration for the application
type Config struct {
	Server    ServerConfig
	WebSocket WebSocketConfig
	JWT       JWTConfig
	NATS      NATSConfig
}

type NATSConfig struct {
	URL     string
	Timeout time.Duration
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// WebSocketConfig holds WebSocket-specific configuration
type WebSocketConfig struct {
	CheckOrigin       bool
	ReadBufferSize    int
	WriteBufferSize   int
	HandshakeTimeout  time.Duration
	EnableCompression bool
}

// JWTConfig holds JWT-related configuration
type JWTConfig struct {
	SecretKey     string
	TokenDuration time.Duration
	Issuer        string
}

// Load loads configuration from environment variables with sensible defaults
func Load() *Config {
	once.Do(func() {
		singleConfig = &Config{
			Server: ServerConfig{
				Port:         getEnv("SERVER_PORT", "9001"),
				Host:         getEnv("SERVER_HOST", "localhost"),
				ReadTimeout:  getDuration("SERVER_READ_TIMEOUT", 5*time.Second),
				WriteTimeout: getDuration("SERVER_WRITE_TIMEOUT", 2*time.Second),
			},
			WebSocket: WebSocketConfig{
				CheckOrigin:       getBool("WS_CHECK_ORIGIN", false),
				ReadBufferSize:    getInt("WS_READ_BUFFER_SIZE", 1024),
				WriteBufferSize:   getInt("WS_WRITE_BUFFER_SIZE", 1024),
				HandshakeTimeout:  getDuration("WS_HANDSHAKE_TIMEOUT", 10*time.Second),
				EnableCompression: getBool("WS_ENABLE_COMPRESSION", false),
			},
			JWT: JWTConfig{
				SecretKey: getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production"),
				Issuer:    getEnv("JWT_ISSUER", "ce-realtime-gateway"),
			},
			NATS: NATSConfig{
				URL: getEnv("NATS_URL", "nats://localhost:4222"),
			},
		}
	})
	return singleConfig
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// GetServerAddress returns the full server address
func (c *Config) GetServerAddress() string {
	return ":" + c.Server.Port
}

// GetWebSocketURL returns the WebSocket URL for the given endpoint
func (c *Config) GetWebSocketURL(endpoint string) string {
	return "ws://" + c.Server.Host + ":" + c.Server.Port + endpoint
}

// GetHTTPURL returns the HTTP URL for the given endpoint
func (c *Config) GetHTTPURL(endpoint string) string {
	return "http://" + c.Server.Host + ":" + c.Server.Port + endpoint
}
