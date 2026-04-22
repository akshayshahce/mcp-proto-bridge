// Package bridge is the public API for decoding MCP tool responses into typed
// Go and protobuf response models.
package bridge

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	bridgeconfig "github.com/akshay/mcp-proto-bridge/pkg/config"
	bridgeerrors "github.com/akshay/mcp-proto-bridge/pkg/errors"
	"github.com/akshay/mcp-proto-bridge/pkg/extractor"
	"github.com/akshay/mcp-proto-bridge/pkg/mapper"
	"github.com/akshay/mcp-proto-bridge/pkg/observe"
	"github.com/akshay/mcp-proto-bridge/pkg/safety"
	"github.com/akshay/mcp-proto-bridge/pkg/types"
	"google.golang.org/protobuf/proto"
)

// Option configures decoding.
type Option = bridgeconfig.Option

var (
	WithFieldAliases            = bridgeconfig.WithFieldAliases
	WithPreferStructuredContent = bridgeconfig.WithPreferStructuredContent
	WithStrictMode              = bridgeconfig.WithStrictMode
	WithAllowUnknownFields      = bridgeconfig.WithAllowUnknownFields
	WithCustomExtractor         = bridgeconfig.WithCustomExtractor
	WithJSONIndentDetection     = bridgeconfig.WithJSONIndentDetection
	WithTargetName              = bridgeconfig.WithTargetName
	WithHooks                   = bridgeconfig.WithHooks
	WithSafetyLimits            = bridgeconfig.WithSafetyLimits
	WithProfile                 = bridgeconfig.WithProfile
	WithDecodePolicy            = bridgeconfig.WithDecodePolicy
	WithVersionRules            = bridgeconfig.WithVersionRules
	WithDriftRules              = bridgeconfig.WithDriftRules
	WithAdaptiveRouting         = bridgeconfig.WithAdaptiveRouting
	WithAutoRepair              = bridgeconfig.WithAutoRepair
	WithRuntimeCounters         = bridgeconfig.WithRuntimeCounters
)

type (
	SafetyLimits    = bridgeconfig.SafetyLimits
	Profile         = bridgeconfig.Profile
	DecodePolicy    = bridgeconfig.DecodePolicy
	VersionRules    = bridgeconfig.VersionRules
	DriftRules      = bridgeconfig.DriftRules
	AdaptiveRouting = bridgeconfig.AdaptiveRouting
	AutoRepair      = bridgeconfig.AutoRepair
	RuntimeCounters = bridgeconfig.RuntimeCounters

	ErrorPolicyAction = bridgeconfig.ErrorPolicyAction
	ValidationPolicy  = bridgeconfig.ValidationPolicy
)

const (
	ErrorPolicyFail   = bridgeconfig.ErrorPolicyFail
	ErrorPolicyIgnore = bridgeconfig.ErrorPolicyIgnore

	ValidationEnforce = bridgeconfig.ValidationEnforce
	ValidationSkip    = bridgeconfig.ValidationSkip
)

// Decode extracts an MCP result payload and decodes it into out.
func Decode(result *types.CallToolResult, out any, opts ...Option) error {
	cfg := bridgeconfig.New(opts...)
	cfg = resolveVersionRules(result, cfg)
	incCounter(cfg, "decode.calls")
	success := false
	defer func() {
		if success {
			incCounter(cfg, "decode.success")
		} else {
			incCounter(cfg, "decode.failure")
		}
	}()

	endFinal := startStage(cfg, bridgeerrors.StageFinalDecode)

	var payload any
	err := runStage(cfg, bridgeerrors.StageExtract, func() error {
		if err := safety.ValidateResult(result, cfg.SafetyLimits); err != nil {
			return err
		}
		extracted, mode, err := extractPayload(result, cfg)
		if err != nil {
			if shouldIgnoreErrorByPolicy(cfg, err) {
				if stderrors.Is(err, bridgeerrors.ErrNoStructuredPayload) && cfg.DriftRules.EmitIgnoredNoPayload {
					incCounter(cfg, "decode.policy.ignored_no_payload")
					emitDrift(cfg, bridgeerrors.StageExtract, "ignored_no_payload", "no payload was ignored by policy", map[string]string{"policy": string(cfg.DecodePolicy.OnNoPayload)})
				}
				payload = map[string]any{}
				cfg.ExtractorMode = "policy_empty"
				return nil
			}
			return err
		}
		if err := safety.ValidatePayload(extracted, cfg.SafetyLimits); err != nil {
			return err
		}
		repaired, repairPasses, repairErr := maybeAutoRepairPayload(extracted, cfg)
		if repairErr != nil {
			return repairErr
		}
		payload = repaired
		cfg.ExtractorMode = mode
		incCounter(cfg, "decode.extractor_mode."+mode)
		cfg.AutoRepairPasses = repairPasses
		if repairPasses > 0 {
			incCounter(cfg, "decode.auto_repair.applied")
			addCounter(cfg, "decode.auto_repair.passes", int64(repairPasses))
		}
		return nil
	})
	if err != nil {
		err = bridgeerrors.WrapDecodeError(bridgeerrors.StageExtract, err)
		endFinal(err)
		return err
	}

	var normalized any
	err = runStage(cfg, bridgeerrors.StageNormalize, func() error {
		normalized = mapper.ApplyAliases(payload, cfg.FieldAliases)
		return safety.ValidatePayload(normalized, cfg.SafetyLimits)
	})
	if err != nil {
		err = bridgeerrors.WrapDecodeError(bridgeerrors.StageNormalize, err)
		endFinal(err)
		return err
	}

	err = runStage(cfg, bridgeerrors.StageMap, func() error {
		return mapper.DecodeMapped(normalized, out, cfg)
	})
	if err != nil {
		err = bridgeerrors.WrapDecodeError(bridgeerrors.StageMap, err)
		endFinal(err)
		return err
	}

	if cfg.DecodePolicy.RequiredValidation != bridgeconfig.ValidationSkip {
		err = runStage(cfg, bridgeerrors.StageValidate, func() error {
			return mapper.ValidateDecoded(out)
		})
		if err != nil {
			err = bridgeerrors.WrapDecodeError(bridgeerrors.StageValidate, err)
			endFinal(err)
			return err
		}
	}

	endFinal(nil)
	success = true
	return nil
}

// DecodeAs extracts an MCP result payload and returns a decoded value.
func DecodeAs[T any](result *types.CallToolResult, opts ...Option) (T, error) {
	targetType := reflect.TypeOf((*T)(nil)).Elem()
	if targetType.Kind() == reflect.Pointer {
		target := reflect.New(targetType.Elem())
		if err := Decode(result, target.Interface(), opts...); err != nil {
			var zero T
			return zero, err
		}
		return target.Interface().(T), nil
	}

	var out T
	err := Decode(result, &out, opts...)
	return out, err
}

// DecodeProto extracts an MCP result payload and decodes it into a protobuf
// message.
func DecodeProto(result *types.CallToolResult, out proto.Message, opts ...Option) error {
	cfg := bridgeconfig.New(opts...)
	cfg = resolveVersionRules(result, cfg)
	incCounter(cfg, "decode_proto.calls")
	success := false
	defer func() {
		if success {
			incCounter(cfg, "decode_proto.success")
		} else {
			incCounter(cfg, "decode_proto.failure")
		}
	}()

	endFinal := startStage(cfg, bridgeerrors.StageFinalDecode)

	var payload any
	err := runStage(cfg, bridgeerrors.StageExtract, func() error {
		if err := safety.ValidateResult(result, cfg.SafetyLimits); err != nil {
			return err
		}
		extracted, mode, err := extractPayload(result, cfg)
		if err != nil {
			if shouldIgnoreErrorByPolicy(cfg, err) {
				if stderrors.Is(err, bridgeerrors.ErrNoStructuredPayload) && cfg.DriftRules.EmitIgnoredNoPayload {
					incCounter(cfg, "decode_proto.policy.ignored_no_payload")
					emitDrift(cfg, bridgeerrors.StageExtract, "ignored_no_payload", "no payload was ignored by policy", map[string]string{"policy": string(cfg.DecodePolicy.OnNoPayload)})
				}
				payload = map[string]any{}
				cfg.ExtractorMode = "policy_empty"
				return nil
			}
			return err
		}
		if err := safety.ValidatePayload(extracted, cfg.SafetyLimits); err != nil {
			return err
		}
		repaired, repairPasses, repairErr := maybeAutoRepairPayload(extracted, cfg)
		if repairErr != nil {
			return repairErr
		}
		payload = repaired
		cfg.ExtractorMode = mode
		incCounter(cfg, "decode_proto.extractor_mode."+mode)
		cfg.AutoRepairPasses = repairPasses
		if repairPasses > 0 {
			incCounter(cfg, "decode_proto.auto_repair.applied")
			addCounter(cfg, "decode_proto.auto_repair.passes", int64(repairPasses))
		}
		return nil
	})
	if err != nil {
		err = bridgeerrors.WrapDecodeError(bridgeerrors.StageExtract, err)
		endFinal(err)
		return err
	}

	var normalized any
	err = runStage(cfg, bridgeerrors.StageNormalize, func() error {
		normalized = mapper.ApplyAliases(payload, cfg.FieldAliases)
		return safety.ValidatePayload(normalized, cfg.SafetyLimits)
	})
	if err != nil {
		err = bridgeerrors.WrapDecodeError(bridgeerrors.StageNormalize, err)
		endFinal(err)
		return err
	}

	err = runStage(cfg, bridgeerrors.StageMap, func() error {
		return mapper.DecodeProtoMapped(normalized, out, cfg)
	})
	if err != nil {
		err = bridgeerrors.WrapDecodeError(bridgeerrors.StageMap, err)
		endFinal(err)
		return err
	}

	endFinal(nil)
	success = true
	return nil
}

func extractPayload(result *types.CallToolResult, cfg bridgeconfig.Config) (any, string, error) {
	if result == nil {
		return nil, cfg.ExtractorMode, bridgeerrors.ErrNoStructuredPayload
	}
	if result.IsError {
		if cfg.DecodePolicy.OnToolError == bridgeconfig.ErrorPolicyIgnore {
			incCounter(cfg, "decode.policy.ignored_tool_error")
			// Ignore tool error marker and attempt payload extraction using normal flow.
			if cfg.DriftRules.EmitIgnoredToolError {
				emitDrift(cfg, bridgeerrors.StageExtract, "ignored_tool_error", "tool isError marker was ignored by policy", map[string]string{"policy": string(cfg.DecodePolicy.OnToolError)})
			}
		} else {
			return nil, cfg.ExtractorMode, fmt.Errorf("%w: %s", bridgeerrors.ErrToolReturnedError, toolErrorSummary(result))
		}
	}

	ext := cfg.Extractor
	mode := cfg.ExtractorMode
	if ext == nil {
		textExtractor := extractor.FirstJSONTextExtractor{EnableIndentDetection: cfg.JSONIndentDetection}
		preferStructured := cfg.PreferStructuredContent
		mode = "default"
		if cfg.AdaptiveRouting.Enabled {
			preferStructured = shouldPreferStructured(result, cfg)
			if preferStructured {
				mode = "adaptive_structured_first"
			} else {
				mode = "adaptive_text_first"
			}
		}
		if preferStructured {
			ext = extractor.CompositeExtractor{Extractors: []extractor.Extractor{
				extractor.PreferStructuredExtractor{},
				textExtractor,
			}}
		} else {
			ext = extractor.CompositeExtractor{Extractors: []extractor.Extractor{
				textExtractor,
				extractor.PreferStructuredExtractor{},
			}}
		}
	} else {
		mode = "custom"
	}

	payload, err := ext.Extract(result)
	if err != nil {
		return nil, mode, err
	}
	return payload, mode, nil
}

func shouldPreferStructured(result *types.CallToolResult, cfg bridgeconfig.Config) bool {
	if result == nil {
		return cfg.PreferStructuredContent
	}
	hasStructured := len(result.StructuredContent) > 0
	hasText := resultHasTextContent(result)

	if !hasStructured && hasText && cfg.AdaptiveRouting.PreferTextWhenNoStructured {
		return false
	}
	if hasStructured && hasText && cfg.AdaptiveRouting.PreferTextWhenBothPresent {
		return false
	}
	return cfg.PreferStructuredContent
}

func resultHasTextContent(result *types.CallToolResult) bool {
	for _, block := range result.Content {
		if text, ok := block.(types.TextContent); ok && strings.TrimSpace(text.Text) != "" {
			return true
		}
		if text, ok := block.(*types.TextContent); ok && text != nil && strings.TrimSpace(text.Text) != "" {
			return true
		}
	}
	return false
}

func maybeAutoRepairPayload(payload any, cfg bridgeconfig.Config) (any, int, error) {
	if !cfg.AutoRepair.Enabled {
		return payload, 0, nil
	}

	maxPasses := cfg.AutoRepair.MaxRepairPasses
	if maxPasses < 1 {
		maxPasses = 1
	}

	current := payload
	passes := 0
	for passes < maxPasses {
		next, changed, err := autoRepairPass(current)
		if err != nil {
			return nil, passes, fmt.Errorf("%w: auto-repair pass failed: %v", bridgeerrors.ErrFieldMappingFailed, err)
		}
		if !changed {
			break
		}
		current = next
		passes++
	}
	return current, passes, nil
}

func autoRepairPass(payload any) (any, bool, error) {
	s, ok := payload.(string)
	if ok {
		trimmed := strings.TrimSpace(s)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			var out any
			dec := json.NewDecoder(strings.NewReader(trimmed))
			dec.UseNumber()
			if err := dec.Decode(&out); err != nil {
				return nil, false, err
			}
			return out, true, nil
		}
	}

	m, ok := payload.(map[string]any)
	if ok && len(m) == 1 {
		for key, value := range m {
			switch key {
			case "payload", "data", "result":
				return value, true, nil
			}
		}
	}

	return payload, false, nil
}

func shouldIgnoreErrorByPolicy(cfg bridgeconfig.Config, err error) bool {
	if stderrors.Is(err, bridgeerrors.ErrNoStructuredPayload) && cfg.DecodePolicy.OnNoPayload == bridgeconfig.ErrorPolicyIgnore {
		return true
	}
	if stderrors.Is(err, bridgeerrors.ErrToolReturnedError) && cfg.DecodePolicy.OnToolError == bridgeconfig.ErrorPolicyIgnore {
		return true
	}
	return false
}

func resolveVersionRules(result *types.CallToolResult, cfg bridgeconfig.Config) bridgeconfig.Config {
	if result == nil || result.Meta == nil {
		return cfg
	}
	if len(cfg.VersionRules.ProfilesByVersion) == 0 {
		return cfg
	}
	metaKey := cfg.VersionRules.VersionMetaKey
	if metaKey == "" {
		metaKey = "schema_version"
	}

	rawVersion, ok := result.Meta[metaKey]
	if !ok {
		return cfg
	}
	version, ok := rawVersion.(string)
	if !ok || version == "" {
		return cfg
	}

	profile, ok := cfg.VersionRules.ProfilesByVersion[version]
	if !ok {
		if cfg.DriftRules.EmitUnknownVersion {
			emitDrift(cfg, bridgeerrors.StageExtract, "unknown_version", "version meta did not match configured version rules", map[string]string{"version": version, "meta_key": metaKey})
		}
		return cfg
	}

	bridgeconfig.WithProfile(profile)(&cfg)
	cfg.ResolvedVersion = version
	cfg.VersionRuleMatched = true
	return cfg
}

func toolErrorSummary(result *types.CallToolResult) string {
	for _, block := range result.Content {
		if text, ok := block.(types.TextContent); ok && text.Text != "" {
			return text.Text
		}
		if text, ok := block.(*types.TextContent); ok && text != nil && text.Text != "" {
			return text.Text
		}
	}
	return "tool result marked as error"
}

func runStage(cfg bridgeconfig.Config, stage bridgeerrors.Stage, fn func() error) error {
	end := startStage(cfg, stage)
	err := fn()
	end(err)
	return err
}

func startStage(cfg bridgeconfig.Config, stage bridgeerrors.Stage) func(err error) {
	start := time.Now()
	if cfg.Hooks.EventLogger != nil {
		cfg.Hooks.EventLogger.LogEvent(observe.Event{
			Kind:       "start",
			Stage:      stage,
			Profile:    cfg.ProfileName,
			Target:     cfg.TargetName,
			Provenance: provenanceFromConfig(cfg),
			Timestamp:  start,
		})
	}

	var tracerEnd func(err error)
	if cfg.Hooks.Tracer != nil {
		tracerEnd = cfg.Hooks.Tracer.StartStage(stage)
	}

	return func(err error) {
		dur := time.Since(start)
		if tracerEnd != nil {
			tracerEnd(err)
		}
		if cfg.Hooks.Metrics != nil {
			cfg.Hooks.Metrics.ObserveStage(stage, dur, err == nil)
		}
		if cfg.Hooks.EventLogger != nil {
			cfg.Hooks.EventLogger.LogEvent(observe.Event{
				Kind:       "finish",
				Stage:      stage,
				Profile:    cfg.ProfileName,
				Target:     cfg.TargetName,
				Duration:   dur,
				Err:        err,
				Provenance: provenanceFromConfig(cfg),
				Timestamp:  time.Now(),
			})
		}
	}
}

func provenanceFromConfig(cfg bridgeconfig.Config) observe.Provenance {
	return observe.Provenance{
		ResolvedVersion:    cfg.ResolvedVersion,
		VersionRuleMatched: cfg.VersionRuleMatched,
		AppliedProfile:     cfg.ProfileName,
		ExtractorMode:      cfg.ExtractorMode,
		AutoRepairPasses:   cfg.AutoRepairPasses,
		PolicyOnToolError:  string(cfg.DecodePolicy.OnToolError),
		PolicyOnNoPayload:  string(cfg.DecodePolicy.OnNoPayload),
		ValidationPolicy:   string(cfg.DecodePolicy.RequiredValidation),
	}
}

func emitDrift(cfg bridgeconfig.Config, stage bridgeerrors.Stage, driftType string, message string, metadata map[string]string) {
	incCounter(cfg, "drift."+driftType)
	if cfg.Hooks.EventLogger == nil {
		return
	}
	cfg.Hooks.EventLogger.LogEvent(observe.Event{
		Kind:       "drift",
		Stage:      stage,
		Profile:    cfg.ProfileName,
		Target:     cfg.TargetName,
		Provenance: provenanceFromConfig(cfg),
		Drift: &observe.DriftSignal{
			Type:     driftType,
			Message:  message,
			Metadata: metadata,
		},
		Timestamp: time.Now(),
	})
}

func incCounter(cfg bridgeconfig.Config, name string) {
	if cfg.RuntimeCounters == nil || name == "" {
		return
	}
	cfg.RuntimeCounters.Inc(name)
}

func addCounter(cfg bridgeconfig.Config, name string, delta int64) {
	if cfg.RuntimeCounters == nil || name == "" || delta == 0 {
		return
	}
	cfg.RuntimeCounters.Add(name, delta)
}
