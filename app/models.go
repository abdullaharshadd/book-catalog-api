package main

import "time"

type Book struct {
	ID            int       `json:"id"`
	Title         string    `json:"title"`
	Author        string    `json:"author"`
	ISBN          string    `json:"isbn,omitempty"`
	Description   string    `json:"description,omitempty"`
	PublishedYear int       `json:"published_year,omitempty"`
	Price         float64   `json:"price,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}