{
  "code": "package tests\n\nimport (\n\t\"context\"\n\t\"encoding/json\"\n\t\"net/http\"\n\t\"testing\"\n\t\"github.com/go-chi/chi/v5\"\n\t\"github.com/go-chi/chi/v5/middleware\"\n\t\"github.com/stretchr/testify/assert\"\n\t\"github.com/stretchr/testify/require\"\n\t\"internal\"\n\t\"internal/database\"\n\t\"internal/model\"\n\t\"internal/schemas\"\n)\n\n// TestSuite represents the test suite for the Book Catalog API.\n// It contains unit tests for models and schemas, plus integration tests for API endpoints.\nfunc TestSuite(t *testing.T) {\n\t// Initialize API and database for testing\n\ta := assert.New(t)\n\treq := require.New(t)\n\tapi := internal.NewBookCatalogAPI()\n\tdb, err := database.NewDatabase()\n\treq.NoError(err)\n\tdb = getSyncDB(t, db)\n\tdefer db.Close()\n\tdropAllTables(t, db)\n\ttestAPIEndpoints(t, api, db)\n}\n\nfunc testAPIEndpoints(t *testing.T, api *internal.BookCatalogAPI, db *sqlx.DB) {\n\t// Setup\n\tclient := &httptest.Server{\n\t\tConfig: &http.Server{Handler: api.Router()},\n\t}\n\tdefer client.Close()\n\n\t// Test root endpoint\n\ttestReadRoot(t, client)\n\n\t// Test health check endpoint\n\ttestHealthCheck(t, client)\n\n\t// Test create book endpoint\n\ttestCreateBook(t, client, db)\n\ttestCreateBookWithoutSummary(t, client, db)\n\ttestCreateBookValidationError(t, client, db)\n\ttestCreateDuplicateBook(t, client, db)\n\ttestCreateBooksSameTitleDifferentAuthors(t, client, db)\n\ttestFullCRUDWorkflow(t, client, db)\n\n\t// Test get books endpoint\n\ttestGetBooksEmpty(t, client, db)\n\ttestGetBooksWithData(t, client, db)\n\ttestGetBooksWithPagination(t, client, db)\n\n\t// Test get book by ID endpoint\n\ttestGetBookByID(t, client, db)\n\ttestGetBookNotFound(t, client, db)\n\n\t// Test update book endpoint\n\ttestUpdateBook(t, client, db)\n\ttestUpdateBookNotFound(t, client, db)\n\ttestUpdateBookValidationError(t, client, db)\n\n\t// Test delete book endpoint\n\ttestDeleteBook(t, client, db)\n\ttestDeleteBookNotFound(t, client, db)\n}\n\nfunc testReadRoot(t *testing.T, client *httptest.Server) {\n\tresp, err := client.Client().Get(client.URL + \"/\")\n\trequire.NoError(t, err)\n\trequire.Equal(t, http.StatusOK, resp.StatusCode)\n\n\tvar data map[string]interface{}\n\terr = json.NewDecoder(resp.Body).Decode(&data)\n\trequire.NoError(t, err)\n\tassert.Equal(t, \"Welcome to Book Catalog API\", data[\"message\"])\n\tassert.Equal(t, \"1.0.0\", data[\"version\"])\n\tassert.Contains(t, data, \"docs_url\")\n}\n\nfunc testHealthCheck(t *testing.T, client *httptest.Server) {\n\tresp, err := client.Client().Get(client.URL + \"/health\")\n\trequire.NoError(t, err)\n\trequire.Equal(t, http.StatusOK, resp.StatusCode)\n\n\tvar data map[string]interface{}\n\terr = json.NewDecoder(resp.Body).Decode(&data)\n\trequire.NoError(t, err)\n\tassert.Equal(t, \"healthy\", data[\"status\"])\n\tassert.Equal(t, \"book-catalog-api\", data[\"service\"])\n}\n\nfunc testCreateBook(t *testing.T, client *httptest.Server, db *sqlx.DB) {\n\tbookData := schemas.BookCreate{\n\t\tTitle:          \"Test Book\",\n\t\tAuthor:         \"Test Author\",\n\t\tPublishedYear:  2023,\n\t\tSummary:        &\"A test book summary\",\n\t}\n\treq := require.New(t)\n\treq.NoError(bookData.Validate())\n\n\tresp, err := client.Client().Post(client.URL+\"/books/\", \"application/json\", readerOfJSON(t, bookData))\n\treq.NoError(err)\n\treq.Equal(http.StatusCreated, resp.StatusCode)\n\n\tvar createdBook schemas.BookResponse\n\terr = json.NewDecoder(resp.Body).Decode(&createdBook)\n\treq.NoError(err)\n\treq.Equal(bookData.Title, createdBook.Title)\n\treq.Equal(bookData.Author, createdBook.Author)\n\treq.Equal(bookData.PublishedYear, createdBook.PublishedYear)\n\treq.Equal(bookData.Summary, &createdBook.Summary)\n\treq.NotNil(createdBook.ID)\n}\n\nfunc testCreateBookWithoutSummary(t *testing.T, client *httptest.Server, db *sqlx.DB) {\n\tbookData := schemas.BookCreate{\n\t\tTitle:          \"Book Without Summary\",\n\t\tAuthor:         \"Author\",\n\t\tPublishedYear:  2023,\n\t}\n\treq := require.New(t)\n\treq.NoError(bookData.Validate())\n\n\tresp, err := client.Client().Post(client.URL+\"/books/\", \"application/json\", readerOfJSON(t, bookData))\n\treq.NoError(err)\n\treq.Equal(http.StatusCreated, resp.StatusCode)\n\n\tvar createdBook schemas.BookResponse\n\terr = json.NewDecoder(resp.Body).Decode(&createdBook)\n\treq.NoError(err)\n\treq.Equal(bookData.Title, createdBook.Title)\n\treq.Equal(bookData.Author, createdBook.Author)\n\treq.Equal(bookData.PublishedYear, createdBook.PublishedYear)\n\treq.Nil(createdBook.Summary)\n}\n\nfunc testCreateBookValidationError(t *testing.T, client *httptest.Server, db *sqlx.DB) {\n\t// Missing required field\n\tbookData := schemas.BookCreate{\n\t\tTitle: \"Test Book\",\n\t}\n\treq := require.New(t)\n\treq.Error(bookData.Validate())\n\n\tresp, err := client.Client().Post(client.URL+\"/books/\", \"application/json\", readerOfJSON(t, bookData))\n\treq.NoError(err)\n\treq.Equal(http.StatusUnprocessableEntity, resp.StatusCode)\n\n\t// Invalid published year\n\tbookData = schemas.BookCreate{\n\t\tTitle:          \"Test Book\",\n\t\tAuthor:         \"Test Author\",\n\t\tPublishedYear:  999,\n\t}\n\treq.Error(bookData.Validate())\n\n\tresp, err = client.Client().Post(client.URL+\"/books/\", \"application/json\", readerOfJSON(t, bookData))\n\treq.NoError(err)\n\treq.Equal(http.StatusUnprocessableEntity, resp.StatusCode)\n}\n\nfunc testGetBooksEmpty(t *testing.T, client *httptest.Server, db *sqlx.DB) {\n\tresp, err := client.Client().Get(client.URL + \"/books/\")\n\trequire.NoError(t, err)\n\trequire.Equal(t, http.StatusOK, resp.StatusCode)\n\n\tvar books [] []s []s [] []nil []*
  [][\ [20()\ "0022000010\n5000000, []
 2222200 schemas
 0: [] books 22([]
 2({[
 000, err.Empty
 1
 5[5()\ "
  // []
 000\nt {}\\ntest
 05 books books
 0222 \"books
 02 {}\
 05 \" [=[
 0 nil
 05[23 *\ [=[ "
 1
",
 24\" books:nil
 1n\treq
 02\n 203 0\n\t
 200.()\user
 1
 02 ( // created
  in: books
",
 2()\ [ created
",
 5
 1\ninternal
 4some
",
 1:()\nnt\n\t
 05 \"Book
 05()\
  :=
 225't(t't(t't*\
 2 = schemas
  // \n\t books
 5(t(t(t([] [=["
 1 {5 schemas []
 22 //\n",
 5"
 25()\n",
 0545()\
 0't as't not(t, db
",
",
 istringstream
",
 2330005't{\nt()\ in't in't't created []*
 4't in: {\, db Books
 2221 in:5't in't(schema
 031nil't not"
 0't as: created Books
 9\n\" in\"\n\tt \"Equal
 2无人
",
 00. created
 2 #'t(t
",
 2\"\n\t't {}\n()\ 
",
 5\"\ books
", +\nnt(\n't",
 25 in, 5
",
",
",
",
 3:5:Created
  resp
",
",
 3 created
",
",
",
",
",
",
",
 325[ \n",
",
 3't't't()\.JSON
",
}",
",
",
",
 5 books
",
",
 5\ books
",
 3't't()\
 33200't created
 irect model
 35
",
 4505't, err
",
",
",
",
",
",
",
",
 3, \"Book
",
",
",
",
",
 22245, \" \" `()\n\t
 5(err
 31't* created
 5 schemas
  //response
",
",
",
 0.ublish
",
",
 2 ->
",({ [()\",
",
 3.json()\ [ [ [ [  //
",
",
",
",
 200,
  \"n\tserver
",
",
",
 4'\
 5:(&
",
",
",
",
",
",
 235s(t \"book
",
",
",
",
",
",
",
",
",
",
",
",
",
 00.T schemas
",
",
",
",
",
",
 5
",
",
", \"",
 4req
",
 1: ( \()\
",
",
",
",
",
",
",
",
",
  //