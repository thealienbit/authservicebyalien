package auth

import (
	"context"

	"github.com/thealienbit/authservicebyalien/internal/store"
)

type contextKey string

const (
	userIDKey   contextKey = "userID"
	emailKey    contextKey = "email"
	verifiedKey contextKey = "emailVerified"
	orgKey      contextKey = "org"
	roleKey     contextKey = "role"
	appKey      contextKey = "app"
)

func WithApp(ctx context.Context, app *store.App) context.Context {
	return context.WithValue(ctx, appKey, app)
}

func AppFromContext(ctx context.Context) (*store.App, bool) {
	app, ok := ctx.Value(appKey).(*store.App)
	return app, ok
}

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

func OrgFromContext(ctx context.Context) (string, bool) {
	org, ok := ctx.Value(orgKey).(string)
	return org, ok
}

func RoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(roleKey).(string)
	return role, ok
}

func WithAuthContext(
	ctx context.Context,
	userID string,
	email string,
	verified bool,
	orgID string,
	role string,
) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, emailKey, email)
	ctx = context.WithValue(ctx, verifiedKey, verified)
	ctx = context.WithValue(ctx, orgKey, orgID)
	ctx = context.WithValue(ctx, roleKey, role)
	return ctx
}
