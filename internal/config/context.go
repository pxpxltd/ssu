package config

import "context"

type ctxKey struct{}
type annotatedCtxKey struct{}

// WithConfig stores a Config in the context.
func WithConfig(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, ctxKey{}, cfg)
}

// FromContext retrieves the Config from the context.
// Returns nil if no Config is stored.
func FromContext(ctx context.Context) *Config {
	cfg, _ := ctx.Value(ctxKey{}).(*Config)
	return cfg
}

// WithAnnotated stores an AnnotatedConfig in the context.
func WithAnnotated(ctx context.Context, ac *AnnotatedConfig) context.Context {
	return context.WithValue(ctx, annotatedCtxKey{}, ac)
}

// AnnotatedFromContext retrieves the AnnotatedConfig from the context.
// Returns nil if no AnnotatedConfig is stored.
func AnnotatedFromContext(ctx context.Context) *AnnotatedConfig {
	ac, _ := ctx.Value(annotatedCtxKey{}).(*AnnotatedConfig)
	return ac
}
