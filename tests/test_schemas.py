# tests/test_schemas.py
import pytest
from pydantic import ValidationError
from datetime import datetime
from app.schemas import BookCreate, BookUpdate, BookResponse


class TestBookCreate:
    """Test cases for BookCreate schema"""
    
    def test_valid_book_create(self):
        """Test creating a valid book"""
        book_data = {
            "title": "Valid Book",
            "author": "Valid Author",
            "published_year": 2023,
            "summary": "A valid book summary"
        }
        
        book = BookCreate(**book_data)
        
        assert book.title == "Valid Book"
        assert book.author == "Valid Author"
        assert book.published_year == 2023
        assert book.summary == "A valid book summary"
    
    def test_book_create_without_summary(self):
        """Test creating a book without summary (optional)"""
        book_data = {
            "title": "Book Without Summary",
            "author": "Author",
            "published_year": 2023
        }
        
        book = BookCreate(**book_data)
        
        assert book.title == "Book Without Summary"
        assert book.author == "Author"
        assert book.published_year == 2023
        assert book.summary is None
    
    def test_book_create_strips_whitespace(self):
        """Test that whitespace is stripped from title and author"""
        book_data = {
            "title": "  Whitespace Book  ",
            "author": "  Whitespace Author  ",
            "published_year": 2023,
            "summary": "  Whitespace summary  "
        }
        
        book = BookCreate(**book_data)
        
        assert book.title == "Whitespace Book"
        assert book.author == "Whitespace Author"
        assert book.summary == "Whitespace summary"
    
    def test_empty_summary_becomes_none(self):
        """Test that empty summary becomes None"""
        book_data = {
            "title": "Book",
            "author": "Author",
            "published_year": 2023,
            "summary": "   "  # Only whitespace
        }
        
        book = BookCreate(**book_data)
        assert book.summary is None
    
    def test_missing_required_fields(self):
        """Test validation errors for missing required fields"""
        # Missing title
        with pytest.raises(ValidationError) as exc_info:
            BookCreate(author="Author", published_year=2023)
        assert "title" in str(exc_info.value)
        
        # Missing author
        with pytest.raises(ValidationError) as exc_info:
            BookCreate(title="Title", published_year=2023)
        assert "author" in str(exc_info.value)
        
        # Missing published_year
        with pytest.raises(ValidationError) as exc_info:
            BookCreate(title="Title", author="Author")
        assert "published_year" in str(exc_info.value)
    
    def test_empty_title_validation(self):
        """Test validation for empty title"""
        with pytest.raises(ValidationError) as exc_info:
            BookCreate(title="", author="Author", published_year=2023)
        assert "Title cannot be empty" in str(exc_info.value)
        
        with pytest.raises(ValidationError) as exc_info:
            BookCreate(title="   ", author="Author", published_year=2023)
        assert "Title cannot be empty" in str(exc_info.value)
    
    def test_empty_author_validation(self):
        """Test validation for empty author"""
        with pytest.raises(ValidationError) as exc_info:
            BookCreate(title="Title", author="", published_year=2023)
        assert "Author cannot be empty" in str(exc_info.value)
        
        with pytest.raises(ValidationError) as exc_info:
            BookCreate(title="Title", author="   ", published_year=2023)
        assert "Author cannot be empty" in str(exc_info.value)
    
    def test_published_year_validation(self):
        """Test validation for published year"""
        current_year = datetime.now().year
        
        # Year too early
        with pytest.raises(ValidationError) as exc_info:
            BookCreate(title="Title", author="Author", published_year=999)
        assert "Published year must be after year 1000" in str(exc_info.value)
        
        # Future year
        with pytest.raises(ValidationError) as exc_info:
            BookCreate(title="Title", author="Author", published_year=current_year + 1)
        assert "cannot be in the future" in str(exc_info.value)
        
        # Valid edge cases
        book_min = BookCreate(title="Title", author="Author", published_year=1000)
        assert book_min.published_year == 1000
        
        book_current = BookCreate(title="Title", author="Author", published_year=current_year)
        assert book_current.published_year == current_year
    
    def test_title_length_validation(self):
        """Test title length validation"""
        # Title too long (over 255 characters)
        long_title = "A" * 256
        with pytest.raises(ValidationError) as exc_info:
            BookCreate(title=long_title, author="Author", published_year=2023)
        assert "ensure this value has at most 255 characters" in str(exc_info.value)
    
    def test_author_length_validation(self):
        """Test author length validation"""
        # Author too long (over 255 characters)
        long_author = "B" * 256
        with pytest.raises(ValidationError) as exc_info:
            BookCreate(title="Title", author=long_author, published_year=2023)
        assert "ensure this value has at most 255 characters" in str(exc_info.value)
    
    def test_summary_length_validation(self):
        """Test summary length validation"""
        # Summary too long (over 2000 characters)
        long_summary = "C" * 2001
        with pytest.raises(ValidationError) as exc_info:
            BookCreate(title="Title", author="Author", published_year=2023, summary=long_summary)
        assert "ensure this value has at most 2000 characters" in str(exc_info.value)


class TestBookUpdate:
    """Test cases for BookUpdate schema"""
    
    def test_valid_partial_update(self):
        """Test updating only some fields"""
        update_data = {
            "title": "Updated Title",
            "published_year": 2024
        }
        
        book_update = BookUpdate(**update_data)
        
        assert book_update.title == "Updated Title"
        assert book_update.author is None
        assert book_update.published_year == 2024
        assert book_update.summary is None
    
    def test_empty_update(self):
        """Test update with no fields provided"""
        book_update = BookUpdate()
        
        assert book_update.title is None
        assert book_update.author is None
        assert book_update.published_year is None
        assert book_update.summary is None
    
    def test_update_validation_same_as_create(self):
        """Test that update validation follows same rules as create"""
        # Empty title should still fail
        with pytest.raises(ValidationError) as exc_info:
            BookUpdate(title="")
        assert "Title cannot be empty" in str(exc_info.value)
        
        # Invalid year should still fail
        with pytest.raises(ValidationError) as exc_info:
            BookUpdate(published_year=999)
        assert "Published year must be after year 1000" in str(exc_info.value)


class TestBookResponse:
    """Test cases for BookResponse schema"""
    
    def test_valid_book_response(self):
        """Test creating a valid book response"""
        book_data = {
            "id": 1,
            "title": "Response Book",
            "author": "Response Author",
            "published_year": 2023,
            "summary": "Response summary"
        }
        
        book = BookResponse(**book_data)
        
        assert book.id == 1
        assert book.title == "Response Book"
        assert book.author == "Response Author"
        assert book.published_year == 2023
        assert book.summary == "Response summary"
    
    def test_book_response_missing_id(self):
        """Test that ID is required in response"""
        with pytest.raises(ValidationError) as exc_info:
            BookResponse(
                title="Title",
                author="Author",
                published_year=2023
            )
        assert "id" in str(exc_info.value)