package middleware

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// Logger is a middleware that logs HTTP requests
func Logger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a wrapper to capture status code
		wrapper := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)
		log.Printf("[%s] %s %s - %d - %v",
			r.Method,
			r.RequestURI,
			r.RemoteAddr,
			wrapper.statusCode,
			duration,
		)
	}
}

// WebSocketLogger is a middleware for WebSocket endpoints that doesn't wrap the response writer
func WebSocketLogger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		log.Printf("[%s] %s %s - WebSocket request started",
			r.Method,
			r.RequestURI,
			r.RemoteAddr,
		)

		next.ServeHTTP(w, r)

		duration := time.Since(start)
		log.Printf("[%s] %s %s - WebSocket request completed - %v",
			r.Method,
			r.RequestURI,
			r.RemoteAddr,
			duration,
		)
	}
}

// CORS adds CORS headers to responses
func CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// Recovery recovers from panics and logs them
func Recovery(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	}
}

// RateLimiter creates a simple rate limiting middleware
func RateLimiter(requests int, window time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	type client struct {
		count    int
		lastSeen time.Time
	}

	clients := make(map[string]*client)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			now := time.Now()

			if c, exists := clients[ip]; exists {
				if now.Sub(c.lastSeen) > window {
					c.count = 1
					c.lastSeen = now
				} else {
					c.count++
				}

				if c.count > requests {
					http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
					return
				}
			} else {
				clients[ip] = &client{
					count:    1,
					lastSeen: now,
				}
			}

			next.ServeHTTP(w, r)
		}
	}
}

// Chain combines multiple middlewares
func Chain(middlewares ...func(http.HandlerFunc) http.HandlerFunc) func(http.HandlerFunc) http.HandlerFunc {
	return func(final http.HandlerFunc) http.HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// responseWrapper wraps http.ResponseWriter to capture status code
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Hijack implements http.Hijacker interface for WebSocket support
func (w *responseWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("http.Hijacker interface not supported")
}
