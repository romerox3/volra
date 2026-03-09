package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const roleContextKey contextKey = "auth_role"

// KeyLookup is a function that returns all active API key hashes with their roles.
type KeyLookup func() ([]KeyEntry, error)

// KeyEntry represents a stored API key for authentication.
type KeyEntry struct {
	KeyHash   string
	Role      Role
	Namespace string
}

// Middleware returns an HTTP middleware that enforces Bearer token authentication.
// If hasKeys returns false, authentication is bypassed (RBAC disabled until first key).
func Middleware(hasKeys func() (bool, error), lookup KeyLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if RBAC is enabled (any keys exist).
			enabled, err := hasKeys()
			if err != nil {
				http.Error(w, `{"error":"internal auth error"}`, http.StatusInternalServerError)
				return
			}
			if !enabled {
				// No keys configured — allow all requests as admin.
				ctx := context.WithValue(r.Context(), roleContextKey, RoleAdmin)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Extract Bearer token.
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"missing or invalid Authorization header"}`))
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Verify token against stored keys.
			keys, err := lookup()
			if err != nil {
				http.Error(w, `{"error":"internal auth error"}`, http.StatusInternalServerError)
				return
			}

			for _, k := range keys {
				if VerifyKey(token, k.KeyHash) {
					ctx := context.WithValue(r.Context(), roleContextKey, k.Role)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid API key"}`))
		})
	}
}

// RoleFromContext extracts the authenticated role from the request context.
func RoleFromContext(ctx context.Context) Role {
	role, ok := ctx.Value(roleContextKey).(Role)
	if !ok {
		return ""
	}
	return role
}

// RequireAction returns an HTTP middleware that checks if the authenticated role
// has permission to perform the given action.
func RequireAction(action Action) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := RoleFromContext(r.Context())
			if role == "" || !CanPerform(role, action) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"insufficient permissions"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
