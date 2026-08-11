package internal

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)


type BookCreate struct {
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	PublishedYear *int    `json:"published_year,omitempty"`
	ISBN          *string `json:"isbn,omitempty"`
	Description   *string `json:"description,omitempty"`
}