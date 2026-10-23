package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Impervguin/ds-lab2/library/internal/auth"
	"github.com/Impervguin/ds-lab2/library/internal/logger"
)

const (
	issuer   = "https://idp.test/"
	audience = "https://library.test/api"
	keyID    = "test-key"
	subject  = "auth0|test-user"
)

func TestMain(m *testing.M) {
	_ = logger.Init(logger.Settings{Level: "error", Format: logger.FormatJSON, Service: "test", Output: io.Discard})
	os.Exit(m.Run())
}

var signingKey = mustKey()

func mustKey() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return key
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"iss":   issuer,
		"sub":   subject,
		"aud":   []string{audience, issuer + "userinfo"},
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"email": "test@test.com",
	}
}

func sign(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID
	raw, err := token.SignedString(signingKey)
	require.NoError(t, err)
	return raw
}

func config() auth.Config {
	return auth.Config{Issuer: issuer, Audience: audience}
}

func newVerifier(t *testing.T, config auth.Config) *auth.Verifier {
	t.Helper()

	verifier, err := auth.NewVerifierWithKeys(func(*jwt.Token) (any, error) {
		return &signingKey.PublicKey, nil
	}, config)
	require.NoError(t, err)
	return verifier
}

// serve passes the request through the middleware and reports what reached
// the protected handler.
func serve(verifier *auth.Verifier, authorization string) (*httptest.ResponseRecorder, *auth.Identity) {
	var seen *auth.Identity
	protected := verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := auth.FromContext(r.Context())
		if ok {
			seen = &identity
		}
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/rating", nil)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	recorder := httptest.NewRecorder()
	protected.ServeHTTP(recorder, request)
	return recorder, seen
}

func TestMiddlewarePutsTheIdentityIntoTheContext(t *testing.T) {
	raw := sign(t, validClaims())

	recorder, identity := serve(newVerifier(t, config()), "Bearer "+raw)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, identity)
	assert.Equal(t, auth.Identity{Username: subject, Token: raw}, *identity)
}

func TestMiddlewareTakesTheUsernameFromTheConfiguredClaim(t *testing.T) {
	withEmail := config()
	withEmail.UsernameClaim = "email"

	recorder, identity := serve(newVerifier(t, withEmail), "Bearer "+sign(t, validClaims()))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, identity)
	assert.Equal(t, "test@test.com", identity.Username)
}

func TestMiddlewareAcceptsASingleAudience(t *testing.T) {
	claims := validClaims()
	claims["aud"] = audience

	recorder, _ := serve(newVerifier(t, config()), "Bearer "+sign(t, claims))

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestMiddlewareRejectsABadAuthorizationHeader(t *testing.T) {
	raw := sign(t, validClaims())

	for name, header := range map[string]string{
		"missing":      "",
		"empty bearer": "Bearer ",
		"basic scheme": "Basic " + raw,
		"no scheme":    raw,
		"garbage":      "Bearer not-a-jwt",
	} {
		t.Run(name, func(t *testing.T) {
			recorder, identity := serve(newVerifier(t, config()), header)

			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
			assert.Equal(t, "Bearer", recorder.Header().Get("WWW-Authenticate"))
			assert.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
			assert.Nil(t, identity)
		})
	}
}

func TestMiddlewareRejectsAnInvalidToken(t *testing.T) {
	foreignKey := mustKey()

	for name, raw := range map[string]func(t *testing.T) string{
		"expired": func(t *testing.T) string {
			claims := validClaims()
			claims["exp"] = time.Now().Add(-time.Minute).Unix()
			return sign(t, claims)
		},
		"without exp": func(t *testing.T) string {
			claims := validClaims()
			delete(claims, "exp")
			return sign(t, claims)
		},
		"foreign audience": func(t *testing.T) string {
			claims := validClaims()
			claims["aud"] = "https://another.test/api"
			return sign(t, claims)
		},
		"foreign issuer": func(t *testing.T) string {
			claims := validClaims()
			claims["iss"] = "https://evil.test/"
			return sign(t, claims)
		},
		"without username": func(t *testing.T) string {
			claims := validClaims()
			delete(claims, "sub")
			return sign(t, claims)
		},
		"foreign signature": func(t *testing.T) string {
			raw, err := jwt.NewWithClaims(jwt.SigningMethodRS256, validClaims()).SignedString(foreignKey)
			require.NoError(t, err)
			return raw
		},
		"symmetric algorithm": func(t *testing.T) string {
			raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims()).SignedString([]byte("secret"))
			require.NoError(t, err)
			return raw
		},
	} {
		t.Run(name, func(t *testing.T) {
			recorder, identity := serve(newVerifier(t, config()), "Bearer "+raw(t))

			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
			assert.Nil(t, identity)
		})
	}
}

func TestVerifierChecksTokensAgainstTheRemoteJWKs(t *testing.T) {
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA",
			"use": "sig",
			"alg": "RS256",
			"kid": keyID,
			"n":   base64.RawURLEncoding.EncodeToString(signingKey.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(signingKey.E)).Bytes()),
		}}})
	}))
	t.Cleanup(jwks.Close)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	remote := config()
	remote.JWKsURL = jwks.URL
	verifier, err := auth.NewVerifier(ctx, remote)
	require.NoError(t, err)

	identity, err := verifier.Verify(sign(t, validClaims()))

	require.NoError(t, err)
	assert.Equal(t, subject, identity.Username)
}

func TestNewVerifierDemandsTheProviderSettings(t *testing.T) {
	keys := func(*jwt.Token) (any, error) { return &signingKey.PublicKey, nil }

	_, err := auth.NewVerifierWithKeys(keys, auth.Config{Audience: audience})
	assert.Error(t, err)

	_, err = auth.NewVerifierWithKeys(keys, auth.Config{Issuer: issuer})
	assert.Error(t, err)

	_, err = auth.NewVerifier(context.Background(), config())
	assert.Error(t, err)
}

func TestContextHelpersAreEmptyOutsideAnAuthenticatedRequest(t *testing.T) {
	_, ok := auth.FromContext(context.Background())

	assert.False(t, ok)
	assert.Empty(t, auth.Username(context.Background()))
	assert.Empty(t, auth.Token(context.Background()))
}
