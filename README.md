# Book Catalog API

A simple CRUD (Create, Read, Update, Delete) RESTful API for managing books, built with FastAPI, SQLAlchemy, and Pydantic.

## Features

- **FastAPI**: Modern, fast web framework for building APIs
- **SQLAlchemy**: SQL toolkit and Object-Relational Mapping (ORM) library
- **Pydantic**: Data validation using Python type annotations
- **Async Support**: Asynchronous database operations
- **SQLite**: Lightweight database for development and testing
- **Comprehensive Testing**: Unit and integration tests
- **Auto-generated Documentation**: OpenAPI/Swagger docs
- **Data Validation**: Input validation and error handling

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | Root endpoint with API information |
| GET | `/books/` | List all books (with pagination) |
| GET | `/books/{id}` | Get a specific book by ID |
| POST | `/books/` | Create a new book |
| PUT | `/books/{id}` | Update an existing book |
| DELETE | `/books/{id}` | Delete a book |
| GET | `/health` | Health check endpoint |
| GET | `/docs` | Interactive API documentation (Swagger UI) |
| GET | `/redoc` | Alternative API documentation (ReDoc) |

## Book Model

Each book has the following fields:

- **id**: Integer (auto-generated primary key)
- **title**: String (required, max 255 characters)
- **author**: String (required, max 255 characters)  
- **published_year**: Integer (required, must be between 1000 and current year)
- **summary**: String (optional, max 2000 characters)

**Note**: The combination of title and author must be unique.

## Quick Start

### Prerequisites

- Python 3.8 or higher
- pip (Python package installer)

### Installation

1. **Clone the repository:**
```bash
git clone <repository-url>
cd book-catalog-api
```

2. **Create a virtual environment:**
```bash
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
```

3. **Install dependencies:**
```bash
pip install -r requirements.txt
# OR using pyproject.toml
pip install -e .
```

### Running the Application

1. **Start the development server:**
```bash
uvicorn app.main:app --reload
```

2. **Access the API:**
- API Base URL: http://localhost:8000
- Interactive Docs: http://localhost:8000/docs
- Alternative Docs: http://localhost:8000/redoc

### Example API Usage

**Create a book:**
```bash
curl -X POST "http://localhost:8000/books/" \
     -H "Content-Type: application/json" \
     -d '{
       "title": "The Great Gatsby",
       "author": "F. Scott Fitzgerald", 
       "published_year": 1925,
       "summary": "A classic American novel set in the summer of 1922."
     }'
```

**Get all books:**
```bash
curl -X GET "http://localhost:8000/books/"
```

**Get a specific book:**
```bash
curl -X GET "http://localhost:8000/books/1"
```

**Update a book:**
```bash
curl -X PUT "http://localhost:8000/books/1" \
     -H "Content-Type: application/json" \
     -d '{
       "summary": "An updated summary of this classic novel."
     }'
```

**Delete a book:**
```bash
curl -X DELETE "http://localhost:8000/books/1"
```

## Testing

The project includes comprehensive unit and integration tests.

### Running Tests

**Run all tests:**
```bash
pytest
```

**Run tests with coverage:**
```bash
pytest --cov=app --cov-report=html
```

**Run specific test files:**
```bash
pytest tests/test_models.py      # Test SQLAlchemy models
pytest tests/test_schemas.py     # Test Pydantic schemas  
pytest tests/test_api.py         # Test API endpoints
```

**Run tests with verbose output:**
```bash
pytest -v
```

### Test Structure

```
tests/
├── test_models.py      # Unit tests for SQLAlchemy models
├── test_schemas.py     # Unit tests for Pydantic schemas
└── test_api.py         # Integration tests for API endpoints
```

## Project Structure

```
book-catalog-api/
├── app/
│   ├── __init__.py
│   ├── main.py         # FastAPI application and endpoints
│   ├── models.py       # SQLAlchemy database models
│   ├── schemas.py      # Pydantic schemas for validation
│   └── database.py     # Database configuration and session management
├── tests/
│   ├── __init__.py
│   ├── test_models.py  # Model unit tests
│   ├── test_schemas.py # Schema unit tests
│   └── test_api.py     # API integration tests
├── requirements.txt    # Python dependencies
├── pyproject.toml     # Project configuration
└── README.md          # This file
```

## Development

### Code Quality

The project uses several tools to maintain code quality:

**Format code:**
```bash
black app tests
```

**Sort imports:**
```bash
isort app tests  
```

**Lint code:**
```bash
flake8 app tests
```

### Environment Variables

You can configure the database URL using environment variables:

```bash
export DATABASE_URL="sqlite:///./books.db"
export ASYNC_DATABASE_URL="sqlite+aiosqlite:///./books.db"
```

### Production Deployment

For production deployment, you can use Gunicorn:

```bash
gunicorn app.main:app -w 4 -k uvicorn.workers.UvicornWorker
```

Or with Docker:

```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY app/ ./app/

CMD ["gunicorn", "app.main:app", "-w", "4", "-k", "uvicorn.workers.UvicornWorker", "--bind", "0.0.0.0:8000"]
```

## Architecture Highlights

### Async Support
- The `/books/` (GET) endpoint demonstrates proper async implementation
- Uses `aiosqlite` for asynchronous SQLite operations
- Async database session management

### Data Validation
- Comprehensive Pydantic schemas with custom validators
- Input sanitization (whitespace trimming)
- Realistic constraints (e.g., published year validation)

### Error Handling
- Proper HTTP status codes
- Detailed error messages
- Database integrity error handling
- Logging for debugging

### Testing Strategy
- **Unit Tests**: Test individual components (models, schemas)
- **Integration Tests**: Test complete API workflows
- **Edge Cases**: Test validation, error conditions, and constraints
- **CRUD Workflow**: Test complete create-read-update-delete cycles

## API Documentation

The API automatically generates comprehensive documentation:

- **Swagger UI**: Interactive documentation at `/docs`
- **ReDoc**: Alternative documentation at `/redoc`  
- **OpenAPI Schema**: Raw schema available at `/openapi.json`

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Support

For questions and support, please open an issue on the GitHub repository.