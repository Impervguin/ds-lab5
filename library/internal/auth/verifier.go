package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"

	"github.com/Impervguin/ds-lab2/library/internal/logger"
)

const DefaultUsernameClaim = "sub"

var ErrInvalidToken = errors.New("invalid token")

type Config struct {
	Issuer        string
	Audience      string
	JWKsURL       string
	UsernameClaim string
}

func (c Config) validate() error {
	switch {
	case c.Issuer == "":
		return errors.New("issuer is required")
	case c.Audience == "":
		return errors.New("audience is required")
	}
	return nil
}

type Verifier struct {
	keys          jwt.Keyfunc
	parser        *jwt.Parser
	usernameClaim string
}

func NewVerifier(ctx context.Context, config Config) (*Verifier, error) {
	if config.JWKsURL == "" {
		return nil, errors.New("jwks url is required")
	}

	log := logger.Named("auth.jwks")
	keys, err := keyfunc.NewDefaultOverrideCtx(ctx, []string{config.JWKsURL}, keyfunc.Override{
		RefreshErrorHandlerFunc: func(url string) func(context.Context, error) {
			return func(ctx context.Context, err error) {
				log.ErrorContext(ctx, "refresh jwks", slog.String("url", url), slog.String("error", err.Error()))
			}
		},
	})
	if err != nil {
		return nil, fmt.Errorf("load jwks: %w", err)
	}
	return NewVerifierWithKeys(keys.Keyfunc, config)
}

func NewVerifierWithKeys(keys jwt.Keyfunc, config Config) (*Verifier, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	usernameClaim := config.UsernameClaim
	if usernameClaim == "" {
		usernameClaim = DefaultUsernameClaim
	}

	return &Verifier{
		keys: keys,
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{"RS256", "RS384", "RS512", "PS256", "PS384", "PS512", "ES256", "ES384", "ES512"}),
			jwt.WithIssuer(config.Issuer),
			jwt.WithAudience(config.Audience),
			jwt.WithExpirationRequired(),
		),
		usernameClaim: usernameClaim,
	}, nil
}

func (v *Verifier) Verify(raw string) (Identity, error) {
	claims := jwt.MapClaims{}
	if _, err := v.parser.ParseWithClaims(raw, claims, v.keys); err != nil {
		return Identity{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	username, _ := claims[v.usernameClaim].(string)
	if username == "" {
		return Identity{}, fmt.Errorf("%w: claim %q is missing", ErrInvalidToken, v.usernameClaim)
	}
	return Identity{Username: username, Token: raw}, nil
}
