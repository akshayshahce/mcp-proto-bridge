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

	// ErrUnsupportedContentType means the extractor encountered a content type
	// it does not support.
	ErrUnsupportedContentType = stderrors.New("unsupported content type")

	// ErrValidationFailed means the decoded payload failed validation.
	ErrValidationFailed = stderrors.New("validation failed")

	// ErrFieldMappingFailed means aliases or payload mapping failed.
	ErrFieldMappingFailed = stderrors.New("field mapping failed")
)

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

