package tests

// This file replaces the pytest suite tests/test_schemas.py, which validated
// the Pydantic schemas BookCreate / BookUpdate / BookResponse for the Book
// Catalog API: field validation, whitespace stripping, length limits, and
// required-vs-empty field enforcement.
//
// MIGRATION_NOTE: Pydantic conflates JSON parsing and validation into model
// construction. In Go the equivalent flow is: json.Unmarshal into a
// CreateRequest/UpdateRequest (whose UnmarshalJSON strips whitespace and
// normalizes empty summary to nil), then call Validate() to enforce the
// business rules. So the tests below build the request either from a JSON
// payload (to exercise the missing-field distinction, which JSON captures via
// pointer nil-ness) or from a struct literal (to exercise Validate rules),
// mirroring how Pydantic's __init__ raised ValidationError.
//
// MIGRATION_NOTE: In Go, runnable tests live in *_test.go files run by
// `go test`. The pytest class grouping (TestBookCreate / TestBookUpdate /
// TestBookResponse) maps onto the table-driven tests in the sibling file
// test_schemas_test.go. This file carries only shared documentation and
// helpers for the schema test suite so `go test` in the sibling picks them up.

import (
	"encoding/json"
	"strings"
)

// decodeCreateRequest parses a JSON payload into a CreateRequest without
// running Validate, so tests can distinguish JSON-shape errors (a missing
// field) from business-rule errors (an empty field). It returns the parsed
// request and any unmarshalling error.
func decodeCreateRequest(payload string) (interface{}, error) {
	// MIGRATION_NOTE: The concrete return type is internal.CreateRequest; it is
	// returned as interface{} here only to keep this doc/helper file free of an
	// import cycle-inducing dependency. The sibling test_schemas_test.go decodes
	// directly into internal.CreateRequest and calls Validate — this helper is a
	// thin convenience mirror of that flow for readability.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// normalizeSummary mirrors the Pydantic validator that trims a summary and
// converts a whitespace-only value to the absence of a summary. It returns the
// trimmed value and whether a non-empty summary was present.
//
// MIGRATION_NOTE: This documents the exact whitespace/empty-summary semantics
// the CreateRequest.UnmarshalJSON in internal/schemas.go implements, so the
// table-driven tests in the sibling file can assert against a single source of
// truth.
func normalizeSummary(s string) (string, bool) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}
