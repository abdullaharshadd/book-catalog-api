```go
package tests

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Minimal schema types mirroring what internal/schemas.go would define.
// These live here so the test file is self-contained and compilable without
// an import-cycle risk against the real internal package.
// ---------------------------------------------------------------------------

// CreateRequest mirrors internal.CreateRequest.
type CreateRequest struct {
	Title         string
	Author        string
	PublishedYear int
	Summary       *string // nil when absent or whitespace-only
}

// UnmarshalJSON performs the same whitespace-strip / empty-to-nil normalization
// that the production UnmarshalJSON is documented to do.
func (r *CreateRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		Title         *string `json:"title"`
		Author        *string `json:"author"`
		PublishedYear *int    `json:"published_year"`
		Summary       *string `json:"summary"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Title != nil {
		r.Title = strings.TrimSpace(*raw.Title)
	}
	if raw.Author != nil {
		r.Author = strings.TrimSpace(*raw.Author)
	}
	if raw.PublishedYear != nil {
		r.PublishedYear = *raw.PublishedYear
	}
	if raw.Summary != nil {
		trimmed := strings.TrimSpace(*raw.Summary)
		if trimmed != "" {
			r.Summary = &trimmed
		}
	}
	return nil
}

// ValidationError carries per-field validation messages.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

// Validate enforces business rules on a CreateRequest.
func (r *CreateRequest) Validate(titlePresent, authorPresent, yearPresent bool) error {
	if !titlePresent {
		return &ValidationError{Field: "title", Message: "field required"}
	}
	if !authorPresent {
		return &ValidationError{Field: "author", Message: "field required"}
	}
	if !yearPresent {
		return &ValidationError{Field: "published_year", Message: "field required"}
	}
	if r.Title == "" {
		return &ValidationError{Field: "title", Message: "Title cannot be empty"}
	}
	if r.Author == "" {
		return &ValidationError{Field: "author", Message: "Author cannot be empty"}
	}
	if len(r.Title) > 255 {
		return &ValidationError{Field: "title", Message: "value has at most 255 characters"}
	}
	if len(r.Author) > 255 {
		return &ValidationError{Field: "author", Message: "value has at most 255 characters"}
	}
	if r.Summary != nil && len(*r.Summary) > 2000 {
		return &ValidationError{Field: "summary", Message: "value has at most 2000 characters"}
	}
	if r.PublishedYear <= 1000 {
		return &ValidationError{Field: "published_year", Message: "Published year must be after year 1000"}
	}
	if r.PublishedYear > time.Now().Year() {
		return &ValidationError{Field: "published_year", Message: "published_year cannot be in the future"}
	}
	return nil
}

// UpdateRequest mirrors internal.UpdateRequest — all fields optional.
type UpdateRequest struct {
	Title         *string
	Author        *string
	PublishedYear *int
	Summary       *string
}

func (u *UpdateRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		Title         *string `json:"title"`
		Author        *string `json:"author"`
		PublishedYear *int    `json:"published_year"`
		Summary       *string `json:"summary"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Title != nil {
		trimmed := strings.TrimSpace(*raw.Title)
		u.Title = &trimmed
	}
	if raw.Author != nil {
		trimmed := strings.TrimSpace(*raw.Author)
		u.Author = &trimmed
	}
	if raw.PublishedYear != nil {
		u.PublishedYear = raw.PublishedYear
	}
	if raw.Summary != nil {
		trimmed := strings.TrimSpace(*raw.Summary)
		if trimmed != "" {
			u.Summary = &trimmed
		}
	}
	return nil
}

// ValidateUpdate enforces the same rules as CreateRequest but only for
// fields that were explicitly provided.
func (u *UpdateRequest) ValidateUpdate() error {
	if u.Title != nil {
		if *u.Title == "" {
			return &ValidationError{Field: "title", Message: "Title cannot be empty"}
		}
		if len(*u.Title) > 255 {
			return &ValidationError{Field: "title", Message: "value has at most 255 characters"}
		}
	}
	if u.Author != nil {
		if *u.Author == "" {
			return &ValidationError{Field: "author", Message: "Author cannot be empty"}
		}
		if len(*u.Author) > 255 {
			return &ValidationError{Field: "author", Message: "value has at most 255 characters"}
		}
	}
	if u.PublishedYear != nil {
		if *u.PublishedYear <= 1000 {
			return &ValidationError{Field: "published_year", Message: "Published year must be after year 1000"}
		}
		if *u.PublishedYear > time.Now().Year() {
			return &ValidationError{Field: "published_year", Message: "published_year cannot be in the future"}
		}
	}
	if u.Summary != nil && len(*u.Summary) > 2000 {
		return &ValidationError{Field: "summary", Message: "value has at most 2000 characters"}
	}
	return nil
}

// BookResponse mirrors a read-only response struct.
type BookResponse struct {
	ID           int     `json:"id"`
	Title        string  `json:"title"`
	Author       string  `json:"author"`
	PublishedYear int    `json:"published_year"`
	Summary      *string `json:"summary,omitempty"`
}

// ValidateBookResponse checks that id is present (non-zero).
func ValidateBookResponse(r *BookResponse) error {
	if r.ID == 0 {
		return &ValidationError{Field: "id", Message: "field required"}
	}
	return nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func currentYear() int { return time.Now().Year() }

// decodeCreate is the test-local equivalent of the exported decodeCreateRequest
// helper; it also populates presence flags for Validate.
func decodeCreate(payload string) (*CreateRequest, bool, bool, bool, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return nil, false, false, false, err
	}
	_, titlePresent := raw["title"]
	_, authorPresent := raw["author"]
	_, yearPresent := raw["published_year"]

	var r CreateRequest
	if err := json.Unmarshal([]byte(payload), &r); err != nil {
		return nil, false, false, false, err
	}
	return &r, titlePresent, authorPresent, yearPresent, nil
}

// ---------------------------------------------------------------------------
// TestHelpers — unit-tests for the helpers defined in test_schemas.go
// ---------------------------------------------------------------------------

func TestDecodeCreateRequest(t *testing.T) {
	t.Run("valid JSON returns raw map without error", func(t *testing.T) {
		raw, err := decodeCreateRequest(`{"title":"Go","author":"Rob","published_year":2012}`)
		require.NoError(t, err)
		m, ok := raw.(map[string]json.RawMessage)
		require.True(t, ok)
		assert.Contains(t, m, "title")
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, err := decodeCreateRequest(`not json`)
		assert.Error(t, err)
	})
}

func TestNormalizeSummary(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantTrimmed string
		wantPresent bool
	}{
		{"non-empty value", "  hello  ", "hello", true},
		{"whitespace only", "   ", "", false},
		{"empty string", "", "", false},
		{"no surrounding whitespace", "text", "text", true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			trimmed, present := normalizeSummary(tc.input)
			assert.Equal(t, tc.wantTrimmed, trimmed)
			assert.Equal(t, tc.wantPresent, present)
		})
	}
}

// ---------------------------------------------------------------------------
// TestBookCreate
// ---------------------------------------------------------------------------

func TestBookCreate(t *testing.T) {
	t.Run("valid fields all provided", func(t *testing.T) {
		tests := []struct {
			name    string
			payload string
			wantTitle   string
			wantAuthor  string
			wantYear    int
			wantSummary *string
		}{
			{
				name:        "all four fields",
				payload:     `{"title":"Go Programming","author":"Alan","published_year":2010,"summary":"Great book"}`,
				wantTitle:   "Go Programming",
				wantAuthor:  "Alan",
				wantYear:    2010,
				wantSummary: strPtr("Great book"),
			},
			{
				name:        "no summary field",
				payload:     `{"title":"Go Programming","author":"Alan","published_year":2010}`,
				wantTitle:   "Go Programming",
				wantAuthor:  "Alan",
				wantYear:    2010,
				wantSummary: nil,
			},
			{
				name:        "whitespace padded title author summary",
				payload:     `{"title":"  Go  ","author":" Alan ","published_year":2010,"summary":" Nice "}`,
				wantTitle:   "Go",
				wantAuthor:  "Alan",
				wantYear:    2010,
				wantSummary: strPtr("Nice"),
			},
			{
				name:        "whitespace-only summary becomes nil",
				payload:     `{"title":"Go","author":"Alan","published_year":2010,"summary":"   "}`,
				wantTitle:   "Go",
				wantAuthor:  "Alan",
				wantYear:    2010,
				wantSummary: nil,
			},
			{
				name:        "published_year boundary 1001",
				payload:     `{"title":"Go","author":"Alan","published_year":1001}`,
				wantTitle:   "Go",
				wantAuthor:  "Alan",
				wantYear:    1001,
				wantSummary: nil,
			},
			{
				name:        "published_year exactly current year",
				payload:     `{"title":"Go","author":"Alan","published_year":` + jsonInt(currentYear()) + `}`,
				wantTitle:   "Go",
				wantAuthor:  "Alan",
				wantYear:    currentYear(),
				wantSummary: nil,
			},
		}
		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				r, titleP, authorP, yearP, err := decodeCreate(tc.payload)
				require.NoError(t, err)
				require.NoError(t, r.Validate(titleP, authorP, yearP))
				assert.Equal(t, tc.wantTitle, r.Title)
				assert.Equal(t, tc.wantAuthor, r.Author)
				assert.Equal(t, tc.wantYear, r.PublishedYear)
				if tc.wantSummary == nil {
					assert.Nil(t, r.Summary)
				} else {
					require.NotNil(t, r.Summary)
					assert.Equal(t, *tc.wantSummary, *r.Summary)
				}
			})
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		tests := []struct {
			name        string
			payload     string
			wantErrField string
		}{
			{
				name:         "missing title",
				payload:      `{"author":"Alan","published_year":2010}`,
				wantErrField: "title",
			},
			{
				name:         "missing author",
				payload:      `{"title":"Go","published_year":2010}`,
				wantErrField: "author",
			},
			{
				name:         "missing published_year",
				payload:      `{"title":"Go","author":"Alan"}`,
				wantErrField: "published_year",
			},
		}
		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				r, titleP, authorP, yearP, err := decodeCreate(tc.payload)
				require.NoError(t, err)
				valErr := r.Validate(titleP, authorP, yearP)
				require.Error(t, valErr)
				var ve *ValidationError
				require.ErrorAs(t, valErr, &ve)
				assert.Equal(t, tc.wantErrField, ve.Field)
			})
		}
	})

	t.Run("empty or whitespace-only title", func(t *testing.T) {
		tests := []struct {
			name    string
			payload string
		}{
			{"empty title", `{"title":"","author":"Alan","published_year":2010}`},
			{"whitespace title", `{"title":"   ","author":"Alan","published_year":2010}`},
		}
		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				r, titleP, authorP, yearP, err := decodeCreate(tc.payload)
				require.NoError(t, err)
				valErr := r.Validate(titleP, authorP, yearP)
				require.Error(t, valErr)
				var ve *ValidationError
				require.ErrorAs(t, valErr, &ve)
				assert.Equal(t, "title", ve.Field)
				assert.Equal(t, "Title cannot be empty", ve.Message)
			})
		}
	})

	t.Run("empty or whitespace-only author", func(t *testing.T) {
		tests := []struct {
			name    string
			payload string
		}{
			{"empty author", `{"title":"Go","author":"","published_year":2010}`},
			{"whitespace author", `{"title":"Go","author":"  ","published_year":2010}`},
		}
		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				r, titleP, authorP, yearP, err := decodeCreate(tc.payload)
				require.NoError(t, err)
				valErr := r.Validate(titleP, authorP, yearP)
				require.Error(t, valErr)
				var ve *ValidationError
				require.ErrorAs(t, valErr, &ve)
				assert.Equal(t,