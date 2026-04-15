package watchdock

import "context"

type scopeKey struct{}

type Scope struct {
	Request *RequestData
	User    *UserData
	Server  *ServerData
}

func WithScope(ctx context.Context, scope Scope) context.Context {
	return context.WithValue(ctx, scopeKey{}, scope)
}

func ScopeFromContext(ctx context.Context) (Scope, bool) {
	scope, ok := ctx.Value(scopeKey{}).(Scope)
	return scope, ok
}
