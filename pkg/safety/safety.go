package safety

// Package safety enforces optional payload safety limits.

import (
	"encoding/json"
	"fmt"
	"reflect"

	bridgeconfig "github.com/akshay/mcp-proto-bridge/pkg/config"
	bridgeerrors "github.com/akshay/mcp-proto-bridge/pkg/errors"
)

// ValidateResult enforces size and complexity limits on the incoming result.
func ValidateResult(result any, limits bridgeconfig.SafetyLimits) error {
	if isLimitsDisabled(limits) {
		return nil
	}

	if limits.MaxPayloadBytes > 0 {
		encoded, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("%w: marshal result for size check: %v", bridgeerrors.ErrPayloadSafetyViolation, err)
		}
		if len(encoded) > limits.MaxPayloadBytes {
			return fmt.Errorf("%w: payload bytes %d exceed max %d", bridgeerrors.ErrPayloadSafetyViolation, len(encoded), limits.MaxPayloadBytes)
		}
	}

	nodeCount := 0
	return validateValue(reflect.ValueOf(result), limits, 0, &nodeCount)
}

// ValidatePayload enforces complexity limits on extracted/normalized payload data.
func ValidatePayload(payload any, limits bridgeconfig.SafetyLimits) error {
	if isLimitsDisabled(limits) {
		return nil
	}
	nodeCount := 0
	return validateValue(reflect.ValueOf(payload), limits, 0, &nodeCount)
}

func isLimitsDisabled(limits bridgeconfig.SafetyLimits) bool {
	return limits.MaxPayloadBytes <= 0 &&
		limits.MaxNestingDepth <= 0 &&
		limits.MaxStringLength <= 0 &&
		limits.MaxCollectionLength <= 0 &&
		limits.MaxNodeCount <= 0
}

func validateValue(value reflect.Value, limits bridgeconfig.SafetyLimits, depth int, nodeCount *int) error {
	if !value.IsValid() {
		return nil
	}

	(*nodeCount)++
	if limits.MaxNodeCount > 0 && *nodeCount > limits.MaxNodeCount {
		return fmt.Errorf("%w: node count exceeds max %d", bridgeerrors.ErrPayloadSafetyViolation, limits.MaxNodeCount)
	}
	if limits.MaxNestingDepth > 0 && depth > limits.MaxNestingDepth {
		return fmt.Errorf("%w: nesting depth %d exceeds max %d", bridgeerrors.ErrPayloadSafetyViolation, depth, limits.MaxNestingDepth)
	}

	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.String:
		if limits.MaxStringLength > 0 && len(value.String()) > limits.MaxStringLength {
			return fmt.Errorf("%w: string length %d exceeds max %d", bridgeerrors.ErrPayloadSafetyViolation, len(value.String()), limits.MaxStringLength)
		}
	case reflect.Slice, reflect.Array:
		if limits.MaxCollectionLength > 0 && value.Len() > limits.MaxCollectionLength {
			return fmt.Errorf("%w: collection length %d exceeds max %d", bridgeerrors.ErrPayloadSafetyViolation, value.Len(), limits.MaxCollectionLength)
		}
		for i := 0; i < value.Len(); i++ {
			if err := validateValue(value.Index(i), limits, depth+1, nodeCount); err != nil {
				return err
			}
		}
	case reflect.Map:
		if limits.MaxCollectionLength > 0 && value.Len() > limits.MaxCollectionLength {
			return fmt.Errorf("%w: map size %d exceeds max %d", bridgeerrors.ErrPayloadSafetyViolation, value.Len(), limits.MaxCollectionLength)
		}
		iter := value.MapRange()
		for iter.Next() {
			if err := validateValue(iter.Key(), limits, depth+1, nodeCount); err != nil {
				return err
			}
			if err := validateValue(iter.Value(), limits, depth+1, nodeCount); err != nil {
				return err
			}
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if err := validateValue(value.Field(i), limits, depth+1, nodeCount); err != nil {
				return err
			}
		}
	}
	return nil
}
