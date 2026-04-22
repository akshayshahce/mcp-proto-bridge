// Package mapper decodes normalized JSON-like payloads into Go and protobuf
// response models.
package mapper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

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
	outValue := reflect.ValueOf(out)
	if outValue.Kind() != reflect.Pointer {
		return fmt.Errorf("%w: output must be a pointer", bridgeerrors.ErrFieldMappingFailed)
	}
	if outValue.IsNil() {
		return fmt.Errorf("%w: output pointer must be non-nil", bridgeerrors.ErrFieldMappingFailed)
	}

	mapped := ApplyAliases(payload, cfg.FieldAliases)
	if err := DecodeMapped(mapped, out, cfg); err != nil {
		return err
	}

	if err := ValidateDecoded(out); err != nil {
		return err
	}
	return nil
}

// DecodeProto maps payload into a protobuf message.
func DecodeProto(payload any, out proto.Message, cfg bridgeconfig.Config) error {
	if out == nil {
		return fmt.Errorf("%w: proto output must be non-nil", bridgeerrors.ErrFieldMappingFailed)
	}
	outValue := reflect.ValueOf(out)
	if outValue.Kind() == reflect.Pointer && outValue.IsNil() {
		return fmt.Errorf("%w: proto output pointer must be non-nil", bridgeerrors.ErrFieldMappingFailed)
	}

	mapped := ApplyAliases(payload, cfg.FieldAliases)
	if err := DecodeProtoMapped(mapped, out, cfg); err != nil {
		return err
	}
	return nil
}

// DecodeMapped decodes already-normalized payload into out.
func DecodeMapped(mapped any, out any, cfg bridgeconfig.Config) error {
	if out == nil {
		return fmt.Errorf("%w: output must be non-nil", bridgeerrors.ErrFieldMappingFailed)
	}
	outValue := reflect.ValueOf(out)
	if outValue.Kind() != reflect.Pointer {
		return fmt.Errorf("%w: output must be a pointer", bridgeerrors.ErrFieldMappingFailed)
	}
	if outValue.IsNil() {
		return fmt.Errorf("%w: output pointer must be non-nil", bridgeerrors.ErrFieldMappingFailed)
	}

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
	return nil
}

// DecodeProtoMapped decodes already-normalized payload into a protobuf target.
func DecodeProtoMapped(mapped any, out proto.Message, cfg bridgeconfig.Config) error {
	if out == nil {
		return fmt.Errorf("%w: proto output must be non-nil", bridgeerrors.ErrFieldMappingFailed)
	}
	outValue := reflect.ValueOf(out)
	if outValue.Kind() == reflect.Pointer && outValue.IsNil() {
		return fmt.Errorf("%w: proto output pointer must be non-nil", bridgeerrors.ErrFieldMappingFailed)
	}

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

// ValidateDecoded validates required tags on decoded structs.
func ValidateDecoded(out any) error {
	return validator.ValidateRequired(out)
}

// ApplyAliases recursively renames map keys using source -> target aliases.
func ApplyAliases(payload any, aliases map[string]string) any {
	if len(aliases) == 0 {
		return payload
	}
	switch v := payload.(type) {
	case map[string]any:
		return applyAliasesMap(v, aliases)
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

func applyAliasesMap(in map[string]any, aliases map[string]string) map[string]any {
	out := make(map[string]any, len(in))

	// First apply aliases recursively to values while preserving original keys.
	for key, value := range in {
		out[key] = ApplyAliases(value, aliases)
	}

	// Then remap keys in deterministic order.
	aliasedKeys := make([]string, 0, len(in))
	for key := range in {
		if _, ok := aliases[key]; ok {
			aliasedKeys = append(aliasedKeys, key)
		}
	}
	sort.Strings(aliasedKeys)

	for _, source := range aliasedKeys {
		target := aliases[source]
		if target == "" || target == source {
			continue
		}

		value, ok := out[source]
		if !ok {
			continue
		}

		// Explicit target keys from input always win over aliased sources.
		if _, explicitTargetExists := in[target]; explicitTargetExists {
			delete(out, source)
			continue
		}

		// For multiple aliases targeting the same key, keep the first assignment
		// in sorted source-key order for deterministic behavior.
		if _, targetAlreadySet := out[target]; targetAlreadySet {
			delete(out, source)
			continue
		}

		out[target] = value
		delete(out, source)
	}

	return out
}

func targetName(cfg bridgeconfig.Config) string {
	if cfg.TargetName != "" {
		return cfg.TargetName
	}
	return "target"
}
