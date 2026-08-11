package main

type BookCreate struct {
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	ISBN          string  `json:"isbn,omitempty"`
	Description   string  `json:"description,omitempty"`
	PublishedYear int     `json:"published_year,omitempty"`
	Price         float64 `json:"price,omitempty"`
}

type BookUpdate struct {
	Title         *string  `json:"title,omitempty"`
	Author        *string  `json:"author,omitempty"`
	ISBN          *string  `json:"isbn,omitempty"`
	Description   *string  `json:"description,omitempty"`
	PublishedYear *int     `json:"published_year,omitempty"`
	Price         *float64 `json:"price,omitempty"`
}