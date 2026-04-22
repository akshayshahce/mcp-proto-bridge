# API Behavior Matrix

This document defines the expected runtime behavior for mcp-proto-bridge decode APIs.

## Public decode APIs

- Decode(result, out, opts...)
- DecodeAs[T](result, opts...)
- DecodeProto(result, out, opts...)

## Output argument contract

### Decode

- out must be a non-nil pointer
- non-pointer or nil-pointer outputs fail with ErrFieldMappingFailed

### DecodeProto

- out must be a non-nil protobuf message
- typed nil protobuf pointers fail with ErrFieldMappingFailed

### DecodeAs[T]

- for non-pointer T, decode populates an internal value and returns it
- for pointer T, decode allocates pointed value and returns non-nil pointer on success

## Extraction behavior

Default extractor order:

- PreferStructuredContent=true: structuredContent first, then text JSON
- PreferStructuredContent=false: text JSON first, then structuredContent

Fallback rules in CompositeExtractor:

- soft-stop errors continue fallback:
  - ErrNoStructuredPayload
  - ErrUnsupportedContentType
  - ErrInvalidJSONTextContent
- all other errors stop immediately

Text extraction behavior:

- scans content blocks in order
- accepts direct JSON object/array text
- optionally scans embedded JSON when JSONIndentDetection is enabled
- malformed JSON-like text can be skipped if a later block contains valid JSON

## Strict vs lenient decode behavior

### Struct decode

- StrictMode=true: unknown fields rejected
- StrictMode=false (default): unknown fields allowed

### Proto decode

- StrictMode=true: unknown fields rejected by protojson
- StrictMode=false (default): unknown fields discarded

## Policy-driven behavior

DecodePolicy supports:

- OnToolError: fail or ignore
- OnNoPayload: fail or ignore
- RequiredValidation: enforce or skip

Policy effects:

- OnToolError=ignore: isError marker does not terminate extraction
- OnNoPayload=ignore: decode continues with empty payload map
- RequiredValidation=skip: required-field validation for struct decode is skipped

## Validation rules

Required tags recognized:

- bridge:"required"
- validate:"required"

Validation traversal:

- pointers and interfaces
- nested structs
- slices and arrays
- maps

Path format examples:

- customer.id
- items[0].sku
- metadata[key]

## Alias mapping behavior

Aliases map source key to target key recursively.

Conflict policy:

- explicit target key in input wins over alias source
- if multiple sources map to same target, first source in sorted-key order wins

## Safety limits behavior

Safety checks are opt-in. Zero-value limits disable checks.

When configured, checks apply to:

- inbound result size (MaxPayloadBytes)
- extracted and normalized payload complexity

Violation returns ErrPayloadSafetyViolation (wrapped in DecodeError by bridge API).

## Structured error model

Bridge wraps stage failures in DecodeError with:

- Stage (extract, normalize, map, validate, final_decode)
- Category
- Recoverability
- SuggestedAction
- Cause (preserves errors.Is compatibility)

## Version rules behavior

When VersionRules are configured:

- version is read from result _meta using VersionMetaKey (default schema_version)
- matching version profile is applied on top of base options
- non-matching versions keep base options

## Drift signal behavior

Drift signals are emitted only when enabled by DriftRules.

Current drift types:

- unknown_version
- ignored_tool_error
- ignored_no_payload

## Adaptive routing behavior

Adaptive routing is opt-in.

When enabled:

- routing can switch extractor order based on runtime shape
- mode is reflected in provenance extractor_mode

## Auto-repair behavior

Auto-repair is opt-in and bounded.

Current repair passes support:

- JSON string payload unwrap into object/array
- single-key envelope unwrap for payload, data, result

MaxRepairPasses bounds repair loop.

## Runtime counter behavior

Runtime counters are opt-in via WithRuntimeCounters.

Bridge emits low-cardinality counters for:

- decode/decode_proto calls, success, failure
- extractor mode selections
- auto-repair applications and pass totals
- policy ignores
- drift signal counts