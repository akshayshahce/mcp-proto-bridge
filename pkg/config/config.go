// Package config contains decoding options shared by bridge packages.
package config

import "github.com/akshay/mcp-proto-bridge/pkg/extractor"

// Config controls extraction, mapping, and validation.
type Config struct {
	FieldAliases            map[string]string
	PreferStructuredContent bool
	StrictMode              bool
	AllowUnknownFields      bool
	Extractor               extractor.Extractor
	TargetName              string
	JSONIndentDetection     bool
}

// Option mutates Config.
type Option func(*Config)

// New returns a Config with production-friendly defaults.
func New(opts ...Option) Config {
	cfg := Config{
		PreferStructuredContent: true,
		AllowUnknownFields:      true,
		JSONIndentDetection:     true,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.StrictMode {
		cfg.AllowUnknownFields = false
	}
	return cfg
}

// WithFieldAliases maps source field names to target field names.
func WithFieldAliases(aliases map[string]string) Option {
	return func(cfg *Config) {
		cfg.FieldAliases = aliases
	}
}

// WithPreferStructuredContent controls whether structuredContent is attempted
// before text content.
func WithPreferStructuredContent(prefer bool) Option {
	return func(cfg *Config) {
		cfg.PreferStructuredContent = prefer
	}
}

// WithStrictMode rejects unknown fields and enables stricter validation.
func WithStrictMode(strict bool) Option {
	return func(cfg *Config) {
		cfg.StrictMode = strict
		if strict {
			cfg.AllowUnknownFields = false
		}
	}
}

// WithAllowUnknownFields controls unknown field handling.
func WithAllowUnknownFields(allow bool) Option {
	return func(cfg *Config) {
		cfg.AllowUnknownFields = allow
	}
}

// WithCustomExtractor uses a caller-provided extractor.
//
// Pass an extractor.CompositeExtractor if you want ordered fallback behavior.
// In that case, extractor errors wrapped with these sentinels are treated as
// soft-stop signals by CompositeExtractor and allow the next extractor to run:
// - bridgeerrors.ErrNoStructuredPayload
// - bridgeerrors.ErrUnsupportedContentType
// - bridgeerrors.ErrInvalidJSONTextContent
func WithCustomExtractor(ext extractor.Extractor) Option {
	return func(cfg *Config) {
		cfg.Extractor = ext
	}
}

// WithJSONIndentDetection enables scanning text content for embedded JSON
// objects/arrays (for example, markdown fenced code blocks) when text does not
// start directly with '{' or '['.
func WithJSONIndentDetection(enabled bool) Option {
	return func(cfg *Config) {
		cfg.JSONIndentDetection = enabled
	}
}

// WithTargetName labels errors with the target model name.
func WithTargetName(name string) Option {
	return func(cfg *Config) {
		cfg.TargetName = name
	}
}
