// Package config contains decoding options shared by bridge packages.
package config

import (
	"github.com/akshay/mcp-proto-bridge/pkg/extractor"
	"github.com/akshay/mcp-proto-bridge/pkg/observe"
)

// ErrorPolicyAction controls how specific decode errors should be handled.
type ErrorPolicyAction string

const (
	ErrorPolicyFail   ErrorPolicyAction = "fail"
	ErrorPolicyIgnore ErrorPolicyAction = "ignore"
)

// ValidationPolicy controls required-field validation behavior.
type ValidationPolicy string

const (
	ValidationEnforce ValidationPolicy = "enforce"
	ValidationSkip    ValidationPolicy = "skip"
)

// DecodePolicy defines policy-driven decode behavior.
type DecodePolicy struct {
	OnToolError        ErrorPolicyAction
	OnNoPayload        ErrorPolicyAction
	RequiredValidation ValidationPolicy
}

// VersionRules configures version-aware profile overrides.
// When VersionMetaKey is present in result _meta and matches an entry in
// ProfilesByVersion, that profile is applied on top of base options.
type VersionRules struct {
	VersionMetaKey    string
	ProfilesByVersion map[string]Profile
}

// DriftRules controls optional emission of basic drift signals.
type DriftRules struct {
	EmitUnknownVersion   bool
	EmitIgnoredToolError bool
	EmitIgnoredNoPayload bool
}

// AdaptiveRouting controls dynamic extractor routing decisions.
type AdaptiveRouting struct {
	Enabled                    bool
	PreferTextWhenNoStructured bool
	PreferTextWhenBothPresent  bool
}

// AutoRepair controls bounded payload repair attempts.
type AutoRepair struct {
	Enabled         bool
	MaxRepairPasses int
}

// RuntimeCounters receives low-cardinality operational counters.
type RuntimeCounters interface {
	Inc(name string)
	Add(name string, delta int64)
}

// SafetyLimits enforces optional payload protection constraints.
// A zero value disables all limits.
type SafetyLimits struct {
	MaxPayloadBytes     int
	MaxNestingDepth     int
	MaxStringLength     int
	MaxCollectionLength int
	MaxNodeCount        int
}

// Profile provides named preset decoder behavior for a tool family/use case.
// Individual options may still override these values by call order.
type Profile struct {
	Name                    string
	FieldAliases            map[string]string
	PreferStructuredContent *bool
	StrictMode              *bool
	AllowUnknownFields      *bool
	Extractor               extractor.Extractor
	JSONIndentDetection     *bool
	TargetName              string
	SafetyLimits            *SafetyLimits
	DecodePolicy            *DecodePolicy
}

// Config controls extraction, mapping, and validation.
type Config struct {
	FieldAliases            map[string]string
	PreferStructuredContent bool
	StrictMode              bool
	AllowUnknownFields      bool
	Extractor               extractor.Extractor
	TargetName              string
	JSONIndentDetection     bool
	Hooks                   observe.Hooks
	SafetyLimits            SafetyLimits
	ProfileName             string
	DecodePolicy            DecodePolicy
	VersionRules            VersionRules
	ResolvedVersion         string
	VersionRuleMatched      bool
	ExtractorMode           string
	DriftRules              DriftRules
	AdaptiveRouting         AdaptiveRouting
	AutoRepair              AutoRepair
	AutoRepairPasses        int
	RuntimeCounters         RuntimeCounters
}

// Option mutates Config.
type Option func(*Config)

// New returns a Config with production-friendly defaults.
func New(opts ...Option) Config {
	cfg := Config{
		PreferStructuredContent: true,
		AllowUnknownFields:      true,
		JSONIndentDetection:     true,
		DecodePolicy: DecodePolicy{
			OnToolError:        ErrorPolicyFail,
			OnNoPayload:        ErrorPolicyFail,
			RequiredValidation: ValidationEnforce,
		},
		VersionRules: VersionRules{
			VersionMetaKey: "schema_version",
		},
		ExtractorMode: "default",
		AdaptiveRouting: AdaptiveRouting{
			Enabled:                    false,
			PreferTextWhenNoStructured: true,
			PreferTextWhenBothPresent:  false,
		},
		AutoRepair: AutoRepair{
			Enabled:         false,
			MaxRepairPasses: 1,
		},
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

// WithProfile applies named profile defaults. Later options can override.
func WithProfile(profile Profile) Option {
	return func(cfg *Config) {
		cfg.ProfileName = profile.Name
		if profile.FieldAliases != nil {
			cfg.FieldAliases = profile.FieldAliases
		}
		if profile.PreferStructuredContent != nil {
			cfg.PreferStructuredContent = *profile.PreferStructuredContent
		}
		if profile.StrictMode != nil {
			cfg.StrictMode = *profile.StrictMode
			if cfg.StrictMode {
				cfg.AllowUnknownFields = false
			}
		}
		if profile.AllowUnknownFields != nil {
			cfg.AllowUnknownFields = *profile.AllowUnknownFields
		}
		if profile.Extractor != nil {
			cfg.Extractor = profile.Extractor
		}
		if profile.JSONIndentDetection != nil {
			cfg.JSONIndentDetection = *profile.JSONIndentDetection
		}
		if profile.TargetName != "" {
			cfg.TargetName = profile.TargetName
		}
		if profile.SafetyLimits != nil {
			cfg.SafetyLimits = *profile.SafetyLimits
		}
		if profile.DecodePolicy != nil {
			cfg.DecodePolicy = *profile.DecodePolicy
		}
	}
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

// WithHooks configures optional observability integrations.
func WithHooks(hooks observe.Hooks) Option {
	return func(cfg *Config) {
		cfg.Hooks = hooks
	}
}

// WithSafetyLimits configures payload safety constraints.
func WithSafetyLimits(limits SafetyLimits) Option {
	return func(cfg *Config) {
		cfg.SafetyLimits = limits
	}
}

// WithDecodePolicy configures policy-driven decode behavior.
func WithDecodePolicy(policy DecodePolicy) Option {
	return func(cfg *Config) {
		cfg.DecodePolicy = policy
	}
}

// WithVersionRules configures version-aware profile overrides.
func WithVersionRules(rules VersionRules) Option {
	return func(cfg *Config) {
		cfg.VersionRules = rules
		if cfg.VersionRules.VersionMetaKey == "" {
			cfg.VersionRules.VersionMetaKey = "schema_version"
		}
	}
}

// WithDriftRules configures optional basic drift signal emission.
func WithDriftRules(rules DriftRules) Option {
	return func(cfg *Config) {
		cfg.DriftRules = rules
	}
}

// WithAdaptiveRouting configures dynamic extractor routing.
func WithAdaptiveRouting(routing AdaptiveRouting) Option {
	return func(cfg *Config) {
		cfg.AdaptiveRouting = routing
	}
}

// WithAutoRepair configures bounded payload repair behavior.
func WithAutoRepair(repair AutoRepair) Option {
	return func(cfg *Config) {
		cfg.AutoRepair = repair
		if cfg.AutoRepair.MaxRepairPasses < 1 {
			cfg.AutoRepair.MaxRepairPasses = 1
		}
	}
}

// WithRuntimeCounters configures optional runtime counters.
func WithRuntimeCounters(counters RuntimeCounters) Option {
	return func(cfg *Config) {
		cfg.RuntimeCounters = counters
	}
}
