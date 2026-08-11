package internal

import (
	"fmt"
	"strings"
	"time"
)

// This file replaces the Pydantic schemas from app/schemas.py: BookCreate,
// BookUpdate and BookResponse. In Go we express these as request/response DTOs
// with explicit validation methods instead of declarative Pydantic validators.
//
// Pydantic's ConfigDict(str_strip_whitespace=True) is reproduced by trimming
// whitespace in the validation logic. from_attributes (ORM mode) is reproduced
// by NewBookResponse, which maps a domain Book into a BookResponse.
//
// MIGRATION_NOTE: Pydantic distinguishes a *missing* field ("Field required")
// from a field that is present but blank/invalid ("value_error"). BookUpdate
// makes every field optional, so to preserve the create-vs-update distinction
// we track field presence during JSON unmarshalling using pointer fields on
// UpdateRequest. A nil pointer means the client omitted the field entirely; a
// non-nil pointer means the field was supplied (and is subject to validation).

// maxTitleAuthorLen is the maximum allowed length for a title or author.
const maxTitleAuthorLen = 255

// maxSummaryLen is the maximum allowed length for a summary.
const maxSummaryLen = 2000

// minPublishedYear is the earliest year a book may be published.
const minPublishedYear = 1000

// ValidationError represents a single field validation failure. It mirrors the
// shape of a Pydantic/FastAPI validation error closely enough for callers to
// build an equivalent HTTP 422 response.
type ValidationError struct {
	// Field is the name of the offending field (source JSON name).
	Field string
	// Message is a human-readable description of the failure.
	Message string
	// Type categorises the failure: "missing" for an absent required field,
	// "value_error" for a present-but-invalid value.
	Type string
}

// Error implements the error interface for ValidationError.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// validateTitle reproduces the Pydantic validate_title rule: reject empty or
// whitespace-only values, cap length at 255, and return the trimmed value.
func validateTitle(v string) (string, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "", &ValidationError{Field: "title", Message: "Title cannot be empty", Type: "value_error"}
	}
	if len(v) > maxTitleAuthorLen {
		return "", &ValidationError{Field: "title", Message: "ensure this value has at most 255 characters", Type: "value_error"}
	}
	return trimmed, nil
}

// validateAuthor reproduces the Pydantic validate_author rule.
func validateAuthor(v string) (string, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "", &ValidationError{Field: "author", Message: "Author cannot be empty", Type: "value_error"}
	}
	if len(v) > maxTitleAuthorLen {
		return "", &ValidationError{Field: "author", Message: "ensure this value has at most 255 characters", Type: "value_error"}
	}
	return trimmed, nil
}

// validatePublishedYear reproduces the Pydantic validate_published_year rule:
// the year must be after 1000 and not in the future relative to now.
func validatePublishedYear(v int) error {
	if v < minPublishedYear {
		return &ValidationError{Field: "published_year", Message: "Published year must be after year 1000", Type: "value_error"}
	}
	currentYear := time.Now().Year()
	if v > currentYear {
		return &ValidationError{
			Field:   "published_year",
			Message: fmt.Sprintf("Published year cannot be in the future (current year: %d)", currentYear),
			Type:    "value_error",
		}
	}
	return nil
}

// validateSummary reproduces the Pydantic validate_summary rule: nil stays nil,
// whitespace-only collapses to nil, and length is capped at 2000. The returned
// pointer is the normalised summary (may be nil).
func validateSummary(v *string) (*string, error) {
	if v == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil, nil
	}
	if len(trimmed) > maxSummaryLen {
		return nil, &ValidationError{Field: "summary", Message: "ensure this value has at most 2000 characters", Type: "value_error"}
	}
	return &trimmed, nil
}