```go
package internal

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpers

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func currentYear() int { return time.Now().Year() }

// unmarshalCreateRequest is a helper that JSON-decodes into a CreateRequest.
func unmarshalCreateRequest(t *testing.T, raw string) (*CreateRequest, error) {
	t.Helper()
	var r CreateRequest
	err := json.Unmarshal([]byte(raw), &r)
	return &r, err
}

func unmarshalUpdateRequest(t *testing.T, raw string) (*UpdateRequest, error) {
	t.Helper()
	var r UpdateRequest
	err := json.Unmarshal([]byte(raw), &r)
	return &r, err
}

// ─── ValidationError ─────────────────────────────────────────────────────────

func TestValidationError_Error(t *testing.T) {
	e := &ValidationError{Field: "title", Message: "Title cannot be empty", Type: "value_error"}
	assert.Equal(t, "title: Title cannot be empty", e.Error())
}

// ─── CreateRequest – JSON unmarshalling ──────────────────────────────────────

func TestCreateRequest_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantErr     bool
		checkFields func(t *testing.T, r *CreateRequest)
	}{
		{
			name: "all fields present",
			body: `{"title":"Go","author":"Rob","published_year":2009,"summary":"A book"}`,
			checkFields: func(t *testing.T, r *CreateRequest) {
				assert.Equal(t, "Go", r.Title)
				assert.Equal(t, "Rob", r.Author)
				assert.Equal(t, 2009, r.PublishedYear)
				require.NotNil(t, r.Summary)
				assert.Equal(t, "A book", *r.Summary)
				assert.True(t, r.presence["title"])
				assert.True(t, r.presence["author"])
				assert.True(t, r.presence["published_year"])
				assert.True(t, r.presence["summary"])
			},
		},
		{
			name: "summary omitted",
			body: `{"title":"Go","author":"Rob","published_year":2009}`,
			checkFields: func(t *testing.T, r *CreateRequest) {
				assert.Nil(t, r.Summary)
				assert.False(t, r.presence["summary"])
			},
		},
		{
			name:    "invalid JSON",
			body:    `{not valid}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := unmarshalCreateRequest(t, tc.body)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.checkFields != nil {
				tc.checkFields(t, r)
			}
		})
	}
}

// ─── CreateRequest.Validate ───────────────────────────────────────────────────

func TestCreateRequest_Validate(t *testing.T) {
	longStr := strings.Repeat("a", 256)
	longSummary := strings.Repeat("a", 2001)
	year := currentYear()

	tests := []struct {
		name        string
		body        string
		wantErr     bool
		errField    string
		errMsg      string
		errType     string
		checkResult func(t *testing.T, r *CreateRequest)
	}{
		// ── happy paths ───────────────────────────────────────────────────────
		{
			name: "valid all fields",
			body: fmt.Sprintf(`{"title":"Go","author":"Rob Pike","published_year":%d,"summary":"Great book"}`, year),
			checkResult: func(t *testing.T, r *CreateRequest) {
				assert.Equal(t, "Go", r.Title)
				assert.Equal(t, "Rob Pike", r.Author)
				assert.Equal(t, year, r.PublishedYear)
				require.NotNil(t, r.Summary)
				assert.Equal(t, "Great book", *r.Summary)
			},
		},
		{
			name: "valid without summary",
			body: fmt.Sprintf(`{"title":"Go","author":"Rob Pike","published_year":%d}`, year),
			checkResult: func(t *testing.T, r *CreateRequest) {
				assert.Nil(t, r.Summary)
			},
		},
		{
			name: "title with surrounding whitespace",
			body: fmt.Sprintf(`{"title":"  Go  ","author":"Rob","published_year":%d}`, year),
			checkResult: func(t *testing.T, r *CreateRequest) {
				assert.Equal(t, "Go", r.Title)
			},
		},
		{
			name: "author with surrounding whitespace",
			body: fmt.Sprintf(`{"title":"Go","author":"  Rob  ","published_year":%d}`, year),
			checkResult: func(t *testing.T, r *CreateRequest) {
				assert.Equal(t, "Rob", r.Author)
			},
		},
		{
			name: "summary with surrounding whitespace",
			body: fmt.Sprintf(`{"title":"Go","author":"Rob","published_year":%d,"summary":"  nice  "}`, year),
			checkResult: func(t *testing.T, r *CreateRequest) {
				require.NotNil(t, r.Summary)
				assert.Equal(t, "nice", *r.Summary)
			},
		},
		{
			name: "summary empty string collapses to nil",
			body: fmt.Sprintf(`{"title":"Go","author":"Rob","published_year":%d,"summary":""}`, year),
			checkResult: func(t *testing.T, r *CreateRequest) {
				assert.Nil(t, r.Summary)
			},
		},
		{
			name: "summary whitespace-only collapses to nil",
			body: fmt.Sprintf(`{"title":"Go","author":"Rob","published_year":%d,"summary":"   "}`, year),
			checkResult: func(t *testing.T, r *CreateRequest) {
				assert.Nil(t, r.Summary)
			},
		},
		{
			name: "summary explicit null",
			body: fmt.Sprintf(`{"title":"Go","author":"Rob","published_year":%d,"summary":null}`, year),
			checkResult: func(t *testing.T, r *CreateRequest) {
				assert.Nil(t, r.Summary)
			},
		},
		{
			name: "published_year exactly 1000",
			body: `{"title":"Go","author":"Rob","published_year":1000}`,
			checkResult: func(t *testing.T, r *CreateRequest) {
				assert.Equal(t, 1000, r.PublishedYear)
			},
		},
		{
			name: "published_year exactly current year",
			body: fmt.Sprintf(`{"title":"Go","author":"Rob","published_year":%d}`, year),
			checkResult: func(t *testing.T, r *CreateRequest) {
				assert.Equal(t, year, r.PublishedYear)
			},
		},
		// ── missing required fields ────────────────────────────────────────────
		{
			name:     "missing title",
			body:     fmt.Sprintf(`{"author":"Rob","published_year":%d}`, year),
			wantErr:  true,
			errField: "title",
			errMsg:   "Field required",
			errType:  "missing",
		},
		{
			name:     "missing author",
			body:     fmt.Sprintf(`{"title":"Go","published_year":%d}`, year),
			wantErr:  true,
			errField: "author",
			errMsg:   "Field required",
			errType:  "missing",
		},
		{
			name:     "missing published_year",
			body:     `{"title":"Go","author":"Rob"}`,
			wantErr:  true,
			errField: "published_year",
			errMsg:   "Field required",
			errType:  "missing",
		},
		// ── title errors ──────────────────────────────────────────────────────
		{
			name:     "title empty string",
			body:     fmt.Sprintf(`{"title":"","author":"Rob","published_year":%d}`, year),
			wantErr:  true,
			errField: "title",
			errMsg:   "Title cannot be empty",
			errType:  "value_error",
		},
		{
			name:     "title whitespace only",
			body:     fmt.Sprintf(`{"title":"   ","author":"Rob","published_year":%d}`, year),
			wantErr:  true,
			errField: "title",
			errMsg:   "Title cannot be empty",
			errType:  "value_error",
		},
		{
			name:     "title too long",
			body:     fmt.Sprintf(`{"title":%q,"author":"Rob","published_year":%d}`, longStr, year),
			wantErr:  true,
			errField: "title",
			errMsg:   "ensure this value has at most 255 characters",
			errType:  "value_error",
		},
		// ── author errors ─────────────────────────────────────────────────────
		{
			name:     "author empty string",
			body:     fmt.Sprintf(`{"title":"Go","author":"","published_year":%d}`, year),
			wantErr:  true,
			errField: "author",
			errMsg:   "Author cannot be empty",
			errType:  "value_error",
		},
		{
			name:     "author whitespace only",
			body:     fmt.Sprintf(`{"title":"Go","author":"   ","published_year":%d}`, year),
			wantErr:  true,
			errField: "author",
			errMsg:   "Author cannot be empty",
			errType:  "value_error",
		},
		{
			name:     "author too long",
			body:     fmt.Sprintf(`{"title":"Go","author":%q,"published_year":%d}`, longStr, year),
			wantErr:  true,
			errField: "author",
			errMsg:   "ensure this value has at most 255 characters",
			errType:  "value_error",
		},
		// ── published_year errors ─────────────────────────────────────────────
		{
			name:     "published_year below 1000",
			body:     `{"title":"Go","author":"Rob","published_year":999}`,
			wantErr:  true,
			errField: "published_year",
			errMsg:   "Published year must be after year 1000",
			errType:  "value_error",
		},
		{
			name:    "published_year in the future",
			body:    fmt.Sprintf(`{"title":"Go","author":"Rob","published_year":%d}`, year+1),
			wantErr: true,
			errField: "published_year",
			errMsg:  fmt.Sprintf("Published year cannot be in the future (current year: %d)", year),
			errType: "value_error",
		},
		// ── summary errors ────────────────────────────────────────────────────
		{
			name:     "summary too long",
			body:     fmt.Sprintf(`{"title":"Go","author":"Rob","published_year":%d,"summary":%q}`, year, longSummary),
			wantErr:  true,
			errField: "summary",
			errMsg:   "ensure this value has at most 2000 characters",
			errType:  "value_error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := unmarshalCreateRequest(t, tc.body)
			require.NoError(t, err, "unmarshal should not fail")

			valErr := r.Validate()
			if tc.wantErr {
				require.Error(t, valErr)
				var ve *ValidationError
				require.ErrorAs(t, valErr, &ve)
				assert.Equal(t, tc.errField, ve.Field)
				assert.Equal(t, tc.errMsg, ve.Message)
				assert.Equal(t, tc.errType, ve.Type)
				return
			}
			require.NoError(t, valErr)
			if tc.checkResult != nil {
				tc.checkResult(t, r)
			}
		})
	}
}

// ─── UpdateRequest – JSON unmarshalling ───────────────────────────────────────

func TestUpdateRequest_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantErr        bool
		summaryPresent bool
		checkFields    func(t *testing.T, r *UpdateRequest)
	}{
		{
			name:           "all fields",
			body:           `{"title":"Go","author":"Rob","published_year":2009,"summary":"book"}`,
			summaryPresent: true,
			checkFields: func(t *testing.T, r *UpdateRequest) {
				require.NotNil(t, r.Title)
				assert.Equal(t, "Go", *r.Title)
				require.NotNil(t, r.Author)
				assert.Equal(t, "Rob", *r.Author)
				require.NotNil(t, r.PublishedYear)
				assert.Equal(t, 2009, *r.PublishedYear)
				require.NotNil(t, r.Summary)
				assert.Equal(t, "book", *r.Summary)
			},
		},
		{
			name:           "summary explicit null – present",
			body:           `{"summary":null}`,
			summaryPresent: true,
			checkFields: func(t *testing.T, r *UpdateRequest) {
				assert.Nil(t, r.Summary)
			},
		},
		{
			name:           "summary omitted – not present",
			body:           `{"title":"Go"}`,
			summaryPresent: false,
		},
		{
			name:    "invalid JSON",
			body:    `{bad}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := unmarshalUpdateRequest(t, tc.body)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.summaryPresent, r.summaryPresent)
			if tc.checkFields != nil {
				tc.checkFields(t, r)
			}
		})
	}
}

// ─── UpdateRequest.Validate ───────────────────────────────────────────────────

func TestUpdateRequest_Validate(t *testing.T) {
	longStr := strings.Repeat("b", 256)
	longSummary := strings.Repeat("b", 2001)
	year := currentYear()

	tests := []struct {
		name        string
		body        string
		wantErr     bool
		errField    string
		errMsg      string
		errType     string
		checkResult func(t *testing.T, r *UpdateRequest)
	}{
		// ── happy paths ───────────────────────────────────────────────────────