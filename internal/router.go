package internal

type BookCreate struct {
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear *int    `json:"published_year,omitempty"`
	ISBN          *string `json:"isbn,omitempty"`
	Description   *string `json:"description,omitempty"`
}