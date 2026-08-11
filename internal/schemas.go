package internal

import (
	"encoding/json"
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

// schemaCreateRequest is the DTO for creating a book. It replaces the Pydantic
// BookCreate schema. Title, Author and PublishedYear are required; Summary is
// optional.
type schemaCreateRequest struct {
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear int     `json:"published_year"`
	Summary       *string `json:"summary"`

	// presence tracks which fields were present in the incoming JSON so that
	// Validate can emit "missing" errors for required fields that were omitted,
	// matching Pydantic's "Field required" behaviour.
	presence map[string]bool
}

// schemaUpdateRequest is the DTO for partially updating a book. It replaces the
// Pydantic BookUpdate schema, where every field is optional. Nil pointers mean
// the field was omitted by the client and must be left untouched.
type schemaUpdateRequest struct {
	Title         *string `json:"title"`
	Author        *string `json:"author"`
	PublishedYear *int    `json:"published_year"`
	Summary       *string `json:"summary"`

	// summaryPresent records whether the "summary" key was present in the
	// incoming JSON, distinguishing an explicit null from an omitted field.
	summaryPresent bool
}

// UnmarshalJSON decodes a CreateRequest while recording which top-level keys
// were present in the source document. This lets Validate distinguish a missing
// required field from a blank one.
func (r *CreateRequest) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	inner := schemaCreateRequest{}
	inner.presence = make(map[string]bool, len(raw))
	for k := range raw {
		inner.presence[k] = true
	}

	type alias struct {
		Title         string  `json:"title"`
		Author        string  `json:"author"`
		PublishedYear int     `json:"published_year"`
		Summary       *string `json:"summary"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	inner.Title = a.Title
	inner.Author = a.Author
	inner.PublishedYear = a.PublishedYear
	inner.Summary = a.Summary

	r.Title = inner.Title
	r.Author = inner.Author
	r.schemaPresence = inner.presence
	r.schemaPublishedYear = inner.PublishedYear
	r.schemaSummary = inner.Summary
	return nil
}

// UnmarshalJSON decodes an UpdateRequest, tracking presence of the optional
// summary field so an explicit null can be told apart from omission.
func (r *UpdateRequest) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	inner := schemaUpdateRequest{}
	_, inner.summaryPresent = raw["summary"]

	type alias struct {
		Title         *string `json:"title"`
		Author        *string `json:"author"`
		PublishedYear *int    `json:"published_year"`
		Summary       *string `json:"summary"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	inner.Title = a.Title
	inner.Author = a.Author
	inner.PublishedYear = a.PublishedYear
	inner.Summary = a.Summary

	r.Title = inner.Title
	r.Author = inner.Author
	r.schemaPublishedYear = inner.PublishedYear
	r.schemaSummary = inner.Summary
	r.schemaSummaryPresent = inner.summaryPresent
	return nil
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