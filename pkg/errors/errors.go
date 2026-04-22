// Package errors exposes sentinel errors returned by the bridge.
package errors

import stderrors "errors"

var (
	// ErrToolReturnedError means the MCP tool result had isError set.
	ErrToolReturnedError = stderrors.New("mcp tool returned error")

	// ErrNoStructuredPayload means no structuredContent or JSON text payload
	// could be extracted.
	ErrNoStructuredPayload = stderrors.New("no structured payload found")

	// ErrInvalidJSONTextContent means a JSON-looking text content block could
	// not be parsed.
	ErrInvalidJSONTextContent = stderrors.New("invalid JSON text content")

	// ErrUnsupportedContentType means content blocks were present but none were
	// usable and at least one unsupported non-text block was encountered.
	ErrUnsupportedContentType = stderrors.New("unsupported content type")

	// ErrValidationFailed means the decoded payload failed validation.
	ErrValidationFailed = stderrors.New("validation failed")

	// ErrFieldMappingFailed means aliases or payload mapping failed.
	ErrFieldMappingFailed = stderrors.New("field mapping failed")

	// ErrPayloadSafetyViolation means a payload exceeded configured safety limits.
	ErrPayloadSafetyViolation = stderrors.New("payload safety violation")
)

// Stage identifies where a decode failure occurred.
type Stage string

const (
	StageExtract     Stage = "extract"
	StageNormalize   Stage = "normalize"
	StageMap         Stage = "map"
	StageValidate    Stage = "validate"
	StageFinalDecode Stage = "final_decode"
)

// Category classifies decode failures into stable groups.
type Category string

const (
	CategoryToolError   Category = "tool_error"
	CategoryNoPayload   Category = "no_payload"
	CategoryInvalidJSON Category = "invalid_json"
	CategoryUnsupported Category = "unsupported_content"
	CategoryValidation  Category = "validation"
	CategoryMapping     Category = "mapping"
	CategorySafety      Category = "safety"
	CategoryUnknown     Category = "unknown"
)

// Recoverability classifies whether a caller should retry, fallback, or fail.
type Recoverability string

const (
	RecoverabilityRetryable Recoverability = "retryable"
	RecoverabilityFallback  Recoverability = "fallback"
	RecoverabilityFatal     Recoverability = "fatal"
)

// DecodeError is a structured error envelope for decode failures.
// It preserves the wrapped cause for sentinel compatibility via errors.Is.
type DecodeError struct {
	Category        Category
	Stage           Stage
	Recoverability  Recoverability
	SuggestedAction string
	Cause           error
}

// Error returns a human-readable structured decode error.
func (e *DecodeError) Error() string {
	if e == nil {
		return "decode error"
	}
	if e.Cause != nil {
		return string(e.Stage) + ": " + string(e.Category) + ": " + e.Cause.Error()
	}
	return string(e.Stage) + ": " + string(e.Category)
}

// Unwrap returns the underlying cause so errors.Is/errors.As continue to work.
func (e *DecodeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// WrapDecodeError builds a structured decode error preserving the wrapped cause.
func WrapDecodeError(stage Stage, cause error) error {
	if cause == nil {
		return nil
	}

	category, rec, action := classifyDecodeError(cause)
	return &DecodeError{
		Category:        category,
		Stage:           stage,
		Recoverability:  rec,
		SuggestedAction: action,
		Cause:           cause,
	}
}

func classifyDecodeError(err error) (Category, Recoverability, string) {
	switch {
	case stderrors.Is(err, ErrToolReturnedError):
		return CategoryToolError, RecoverabilityFatal, "inspect upstream tool error payload"
	case stderrors.Is(err, ErrNoStructuredPayload):
		return CategoryNoPayload, RecoverabilityFallback, "try alternate extractor or adjust tool output"
	case stderrors.Is(err, ErrInvalidJSONTextContent):
		return CategoryInvalidJSON, RecoverabilityFallback, "fix text JSON formatting or enable alternate payload path"
	case stderrors.Is(err, ErrUnsupportedContentType):
		return CategoryUnsupported, RecoverabilityFallback, "provide structuredContent or text JSON content blocks"
	case stderrors.Is(err, ErrValidationFailed):
		return CategoryValidation, RecoverabilityFatal, "update required fields or relax validation policy"
	case stderrors.Is(err, ErrFieldMappingFailed):
		return CategoryMapping, RecoverabilityFatal, "verify aliases, target contract, and strictness settings"
	case stderrors.Is(err, ErrPayloadSafetyViolation):
		return CategorySafety, RecoverabilityFatal, "reduce payload complexity or raise configured safety limits"
	default:
		return CategoryUnknown, RecoverabilityFatal, "inspect wrapped cause and decoder stage"
	}
}

// FieldError describes one validation or mapping failure.
type FieldError struct {
	Field   string
	Message string
}

// Error returns a human-readable field error.
func (e FieldError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}
