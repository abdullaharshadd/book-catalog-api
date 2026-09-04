package internal

// BookCatalogAPI represents the metadata of the Book Catalog API.
type BookCatalogAPI struct {
	Version string
	Author  string
	Email   string
}

// NewBookCatalogAPI returns a new instance of BookCatalogAPI.
func NewBookCatalogAPI() *BookCatalogAPI {
	return &BookCatalogAPI{
		Version: "1.0.0",
		Author:  "Abdullah Arshad",
		Email:   "abdullah.arshad.314@gmail.com",
	}
}

// GetBookCatalogAPI returns the BookCatalogAPI metadata.
func GetBookCatalogAPI() *BookCatalogAPI {
	return NewBookCatalogAPI()
}
