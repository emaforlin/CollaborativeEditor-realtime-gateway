package middleware

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/emaforlin/ce-realtime-gateway/config"
	"github.com/golang-jwt/jwt/v5"
)

// Context keys for storing user information
type contextKey string

const (
	UserIDKey contextKey = "userID"
	IssuerKey contextKey = "issuer"
)

// GetUserID extracts the user ID from the request context
func GetUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	return userID, ok
}

// GetIssuer extracts the issuer from the request context
func GetIssuer(r *http.Request) (string, bool) {
	issuer, ok := r.Context().Value(IssuerKey).(string)
	return issuer, ok
}

// AuthJWT is a middleware to authenticate request via validating JWT tokens
func AuthJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.URL.Query().Get("token")

		// Check if token is provided
		if tokenStr == "" {
			http.Error(w, "Missing token parameter", http.StatusUnauthorized)
			return
		}

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(config.Load().JWT.SecretKey), nil
		})

		if err != nil {
			log.Printf("JWT validation error: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Check if token is valid and extract claims
		if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
			sub, err := claims.GetSubject()
			if err != nil {
				log.Printf("Failed to get subject from token: %v", err)
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			// Store user info in request context for downstream handlers
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, sub)
			if claims.Issuer != "" {
				ctx = context.WithValue(ctx, IssuerKey, claims.Issuer)
			}
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		} else {
			log.Printf("Invalid token or claims")
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
	}
}

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
