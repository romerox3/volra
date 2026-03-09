package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey_ProducesValidOutput(t *testing.T) {
	id, plaintext, hash, err := GenerateKey()
	require.NoError(t, err)

	assert.Len(t, id, 16)         // 8 bytes = 16 hex chars
	assert.Len(t, plaintext, 64)  // 32 bytes = 64 hex chars
	assert.NotEmpty(t, hash)

	// Verify the key matches.
	assert.True(t, VerifyKey(plaintext, hash))
}

func TestVerifyKey_WrongKey(t *testing.T) {
	_, _, hash, err := GenerateKey()
	require.NoError(t, err)

	assert.False(t, VerifyKey("wrong-key", hash))
}

func TestCanPerform_AdminCanDoEverything(t *testing.T) {
	for action := range permissions[RoleAdmin] {
		assert.True(t, CanPerform(RoleAdmin, action), "admin should be able to %s", action)
	}
}

func TestCanPerform_ViewerCannotDeploy(t *testing.T) {
	assert.False(t, CanPerform(RoleViewer, ActionDeployAgent))
	assert.False(t, CanPerform(RoleViewer, ActionStopAgent))
	assert.False(t, CanPerform(RoleViewer, ActionManageKeys))
}

func TestCanPerform_OperatorCanDeployButNotManageKeys(t *testing.T) {
	assert.True(t, CanPerform(RoleOperator, ActionDeployAgent))
	assert.True(t, CanPerform(RoleOperator, ActionStopAgent))
	assert.False(t, CanPerform(RoleOperator, ActionManageKeys))
}

func TestCanPerform_InvalidRole(t *testing.T) {
	assert.False(t, CanPerform(Role("superadmin"), ActionListAgents))
}

func TestValidRoles(t *testing.T) {
	assert.True(t, ValidRoles[RoleAdmin])
	assert.True(t, ValidRoles[RoleOperator])
	assert.True(t, ValidRoles[RoleViewer])
	assert.False(t, ValidRoles[Role("root")])
}

// --- Middleware tests ---

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := RoleFromContext(r.Context())
		w.Write([]byte("ok:" + string(role)))
	})
}

func TestMiddleware_NoKeysAllowsAll(t *testing.T) {
	hasKeys := func() (bool, error) { return false, nil }
	lookup := func() ([]KeyEntry, error) { return nil, nil }

	handler := Middleware(hasKeys, lookup)(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok:admin", w.Body.String())
}

func TestMiddleware_MissingTokenReturns401(t *testing.T) {
	hasKeys := func() (bool, error) { return true, nil }
	lookup := func() ([]KeyEntry, error) { return nil, nil }

	handler := Middleware(hasKeys, lookup)(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMiddleware_InvalidTokenReturns401(t *testing.T) {
	_, _, hash, _ := GenerateKey()
	hasKeys := func() (bool, error) { return true, nil }
	lookup := func() ([]KeyEntry, error) {
		return []KeyEntry{{KeyHash: hash, Role: RoleAdmin}}, nil
	}

	handler := Middleware(hasKeys, lookup)(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMiddleware_ValidTokenSetsRole(t *testing.T) {
	_, plaintext, hash, _ := GenerateKey()
	hasKeys := func() (bool, error) { return true, nil }
	lookup := func() ([]KeyEntry, error) {
		return []KeyEntry{{KeyHash: hash, Role: RoleOperator}}, nil
	}

	handler := Middleware(hasKeys, lookup)(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+plaintext)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok:operator", w.Body.String())
}

func TestRequireAction_Allowed(t *testing.T) {
	_, plaintext, hash, _ := GenerateKey()
	hasKeys := func() (bool, error) { return true, nil }
	lookup := func() ([]KeyEntry, error) {
		return []KeyEntry{{KeyHash: hash, Role: RoleOperator}}, nil
	}

	handler := Middleware(hasKeys, lookup)(RequireAction(ActionDeployAgent)(okHandler()))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+plaintext)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAction_Denied(t *testing.T) {
	_, plaintext, hash, _ := GenerateKey()
	hasKeys := func() (bool, error) { return true, nil }
	lookup := func() ([]KeyEntry, error) {
		return []KeyEntry{{KeyHash: hash, Role: RoleViewer}}, nil
	}

	handler := Middleware(hasKeys, lookup)(RequireAction(ActionDeployAgent)(okHandler()))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+plaintext)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
