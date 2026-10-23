package auth

import "context"

// Identity is the authenticated caller of a request: who they are and the
// token they proved it with, which is forwarded to the upstream services.
type Identity struct {
	Username string
	Token    string
}

type identityKey struct{}

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, identity)
}

func FromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(Identity)
	return identity, ok
}

// Username returns the authenticated user, or "" outside an authenticated request.
func Username(ctx context.Context) string {
	identity, _ := FromContext(ctx)
	return identity.Username
}

// Token returns the caller's raw JWT, or "" outside an authenticated request.
func Token(ctx context.Context) string {
	identity, _ := FromContext(ctx)
	return identity.Token
}
