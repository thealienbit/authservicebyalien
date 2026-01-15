package auth

import "context"

type contextKey string

const (
	userIDKey  contextKey = "userID"
	emailKey   contextKey = "email"
	verifiedKey contextKey = "emailVerified"
)

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok
}

func EmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(emailKey).(string)
	return email, ok
}

func EmailVerifiedFromContext(ctx context.Context) (bool, bool) {
	v, ok := ctx.Value(verifiedKey).(bool)
	return v, ok
}

func WithAuthContext(
	ctx context.Context,
	userID string,
	email string,
	verified bool,
) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, emailKey, email)
	ctx = context.WithValue(ctx, verifiedKey, verified)
	return ctx
}


// TODO: JWT middleware will set this later.
