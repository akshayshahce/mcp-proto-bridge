package replay

// Package replay captures deterministic decode failure artifacts and replays them.

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/akshay/mcp-proto-bridge/pkg/bridge"
	bridgeconfig "github.com/akshay/mcp-proto-bridge/pkg/config"
	bridgeerrors "github.com/akshay/mcp-proto-bridge/pkg/errors"
	"github.com/akshay/mcp-proto-bridge/pkg/types"
	"google.golang.org/protobuf/proto"
)

const artifactVersion = 1

// Artifact stores enough data to replay a decode failure deterministically.
type Artifact struct {
	Version      int             `json:"version"`
	Mode         string          `json:"mode"`
	Config       ArtifactConfig  `json:"config"`
	Result       json.RawMessage `json:"result"`
	ErrorMessage string          `json:"error_message,omitempty"`
	ErrorStage   string          `json:"error_stage,omitempty"`
	ErrorClass   string          `json:"error_class,omitempty"`
}

// ArtifactConfig is a serializable subset of bridge decode config.
type ArtifactConfig struct {
	FieldAliases            map[string]string            `json:"field_aliases,omitempty"`
	PreferStructuredContent bool                         `json:"prefer_structured_content"`
	StrictMode              bool                         `json:"strict_mode"`
	AllowUnknownFields      bool                         `json:"allow_unknown_fields"`
	TargetName              string                       `json:"target_name,omitempty"`
	JSONIndentDetection     bool                         `json:"json_indent_detection"`
	ProfileName             string                       `json:"profile_name,omitempty"`
	SafetyLimits            bridgeconfig.SafetyLimits    `json:"safety_limits,omitempty"`
	DecodePolicy            bridgeconfig.DecodePolicy    `json:"decode_policy,omitempty"`
	AdaptiveRouting         bridgeconfig.AdaptiveRouting `json:"adaptive_routing,omitempty"`
	AutoRepair              bridgeconfig.AutoRepair      `json:"auto_repair,omitempty"`
}

// CaptureDecodeFailure captures a deterministic artifact for Decode failures.
func CaptureDecodeFailure(result *types.CallToolResult, opts []bridge.Option, decodeErr error) ([]byte, error) {
	cfg := bridgeconfig.New(opts...)
	return capture("decode", result, cfg, decodeErr)
}

// CaptureDecodeProtoFailure captures a deterministic artifact for DecodeProto failures.
func CaptureDecodeProtoFailure(result *types.CallToolResult, opts []bridge.Option, decodeErr error) ([]byte, error) {
	cfg := bridgeconfig.New(opts...)
	return capture("decode_proto", result, cfg, decodeErr)
}

// ReplayDecode replays a captured decode artifact.
func ReplayDecode(artifactJSON []byte, out any) error {
	artifact, result, opts, err := parseArtifact(artifactJSON)
	if err != nil {
		return err
	}
	if artifact.Mode != "decode" {
		return fmt.Errorf("%w: artifact mode %q is not decode", bridgeerrors.ErrFieldMappingFailed, artifact.Mode)
	}
	return bridge.Decode(result, out, opts...)
}

// ReplayDecodeProto replays a captured proto decode artifact.
func ReplayDecodeProto(artifactJSON []byte, out proto.Message) error {
	artifact, result, opts, err := parseArtifact(artifactJSON)
	if err != nil {
		return err
	}
	if artifact.Mode != "decode_proto" {
		return fmt.Errorf("%w: artifact mode %q is not decode_proto", bridgeerrors.ErrFieldMappingFailed, artifact.Mode)
	}
	return bridge.DecodeProto(result, out, opts...)
}

func capture(mode string, result *types.CallToolResult, cfg bridgeconfig.Config, decodeErr error) ([]byte, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}

	artifact := Artifact{
		Version: artifactVersion,
		Mode:    mode,
		Config: ArtifactConfig{
			FieldAliases:            cfg.FieldAliases,
			PreferStructuredContent: cfg.PreferStructuredContent,
			StrictMode:              cfg.StrictMode,
			AllowUnknownFields:      cfg.AllowUnknownFields,
			TargetName:              cfg.TargetName,
			JSONIndentDetection:     cfg.JSONIndentDetection,
			ProfileName:             cfg.ProfileName,
			SafetyLimits:            cfg.SafetyLimits,
			DecodePolicy:            cfg.DecodePolicy,
			AdaptiveRouting:         cfg.AdaptiveRouting,
			AutoRepair:              cfg.AutoRepair,
		},
		Result: resultJSON,
	}
	if decodeErr != nil {
		artifact.ErrorMessage = decodeErr.Error()
		var de *bridgeerrors.DecodeError
		if ok := asDecodeError(decodeErr, &de); ok {
			artifact.ErrorStage = string(de.Stage)
			artifact.ErrorClass = string(de.Category)
		}
	}
	return json.MarshalIndent(artifact, "", "  ")
}

func parseArtifact(artifactJSON []byte) (Artifact, *types.CallToolResult, []bridge.Option, error) {
	var artifact Artifact
	if err := json.Unmarshal(artifactJSON, &artifact); err != nil {
		return Artifact{}, nil, nil, fmt.Errorf("unmarshal artifact: %w", err)
	}
	if artifact.Version != artifactVersion {
		return Artifact{}, nil, nil, fmt.Errorf("unsupported artifact version: %d", artifact.Version)
	}

	var result types.CallToolResult
	if err := json.Unmarshal(artifact.Result, &result); err != nil {
		return Artifact{}, nil, nil, fmt.Errorf("unmarshal artifact result: %w", err)
	}

	opts := []bridge.Option{
		bridge.WithPreferStructuredContent(artifact.Config.PreferStructuredContent),
		bridge.WithStrictMode(artifact.Config.StrictMode),
		bridge.WithAllowUnknownFields(artifact.Config.AllowUnknownFields),
		bridge.WithJSONIndentDetection(artifact.Config.JSONIndentDetection),
		bridge.WithTargetName(artifact.Config.TargetName),
		bridge.WithSafetyLimits(artifact.Config.SafetyLimits),
		bridge.WithDecodePolicy(artifact.Config.DecodePolicy),
		bridge.WithAdaptiveRouting(artifact.Config.AdaptiveRouting),
		bridge.WithAutoRepair(artifact.Config.AutoRepair),
	}
	if len(artifact.Config.FieldAliases) > 0 {
		opts = append(opts, bridge.WithFieldAliases(artifact.Config.FieldAliases))
	}
	if artifact.Config.ProfileName != "" {
		opts = append(opts, bridge.WithProfile(bridge.Profile{Name: artifact.Config.ProfileName}))
	}

	return artifact, &result, opts, nil
}

func asDecodeError(err error, out **bridgeerrors.DecodeError) bool {
	var de *bridgeerrors.DecodeError
	if !errors.As(err, &de) {
		return false
	}
	*out = de
	return true
}
