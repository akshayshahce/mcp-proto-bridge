// Package mapper decodes normalized JSON-like payloads into Go and protobuf
// response models.
package mapper

import (
	"bytes"
	"encoding/json"
	"fmt"

	bridgeconfig "github.com/akshay/mcp-proto-bridge/pkg/config"
	bridgeerrors "github.com/akshay/mcp-proto-bridge/pkg/errors"
	"github.com/akshay/mcp-proto-bridge/pkg/validator"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Decode maps payload into out, which must be a non-nil pointer.
func Decode(payload any, out any, cfg bridgeconfig.Config) error {
	if out == nil {
		return fmt.Errorf("%w: output must be non-nil", bridgeerrors.ErrFieldMappingFailed)
	}

	mapped := ApplyAliases(payload, cfg.FieldAliases)
	data, err := json.Marshal(mapped)
	if err != nil {
		return fmt.Errorf("%w: marshal normalized payload: %v", bridgeerrors.ErrFieldMappingFailed, err)
	}

	if cfg.AllowUnknownFields {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("%w: decode %s: %v", bridgeerrors.ErrFieldMappingFailed, targetName(cfg), err)
		}
	} else {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(out); err != nil {
			return fmt.Errorf("%w: decode %s: %v", bridgeerrors.ErrFieldMappingFailed, targetName(cfg), err)
		}
	}

	if err := validator.ValidateRequired(out); err != nil {
		return err
	}
	return nil
}

// DecodeProto maps payload into a protobuf message.
func DecodeProto(payload any, out proto.Message, cfg bridgeconfig.Config) error {
	if out == nil {
		return fmt.Errorf("%w: proto output must be non-nil", bridgeerrors.ErrFieldMappingFailed)
	}

	mapped := ApplyAliases(payload, cfg.FieldAliases)
	data, err := json.Marshal(mapped)
	if err != nil {
		return fmt.Errorf("%w: marshal normalized payload: %v", bridgeerrors.ErrFieldMappingFailed, err)
	}

	opts := protojson.UnmarshalOptions{
		DiscardUnknown: cfg.AllowUnknownFields,
	}
	if err := opts.Unmarshal(data, out); err != nil {
		return fmt.Errorf("%w: decode proto %s: %v", bridgeerrors.ErrFieldMappingFailed, targetName(cfg), err)
	}
	return nil
}

// ApplyAliases recursively renames map keys using source -> target aliases.
func ApplyAliases(payload any, aliases map[string]string) any {
	if len(aliases) == 0 {
		return payload
	}
	switch v := payload.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, value := range v {
			target := key
			if alias, ok := aliases[key]; ok {
				target = alias
			}
			out[target] = ApplyAliases(value, aliases)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, value := range v {
			out[i] = ApplyAliases(value, aliases)
		}
		return out
	default:
		return payload
	}
}

func targetName(cfg bridgeconfig.Config) string {
	if cfg.TargetName != "" {
		return cfg.TargetName
	}
	return "target"
}

