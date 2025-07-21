# tests/test_models.py
import pytest
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from app.models import Base, Book


@pytest.fixture
def db_session():
    """Create a test database session"""
    engine = create_engine("sqlite:///:memory:", echo=True)
    Base.metadata.create_all(bind=engine)
    
    TestingSessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)
    session = TestingSessionLocal()
    
    yield session
    
    session.close()


class TestBookModel:
    """Test cases for the Book model"""
    
    def test_create_book(self, db_session):
        """Test creating a basic book"""
        book = Book(
            title="Test Book",
            author="Test Author",
            published_year=2023,
            summary="A test book summary"
        )
        
        db_session.add(book)
        db_session.commit()
        db_session.refresh(book)
        
        assert book.id is not None
        assert book.title == "Test Book"
        assert book.author == "Test Author"
        assert book.published_year == 2023
        assert book.summary == "A test book summary"
    
    def test_create_book_without_summary(self, db_session):
        """Test creating a book without summary (optional field)"""
        book = Book(
            title="Test Book No Summary",
            author="Test Author",
            published_year=2023
        )
        
        db_session.add(book)
        db_session.commit()
        db_session.refresh(book)
        
        assert book.id is not None
        assert book.title == "Test Book No Summary"
        assert book.author == "Test Author"
        assert book.published_year == 2023
        assert book.summary is None
    
    def test_book_repr(self, db_session):
        """Test book string representation"""
        book = Book(
            title="Repr Test",
            author="Repr Author",
            published_year=2023,
            summary="Test summary"
        )
        
        db_session.add(book)
        db_session.commit()
        db_session.refresh(book)
        
        expected_repr = f"<Book(id={book.id}, title='Repr Test', author='Repr Author', year=2023)>"
        assert repr(book) == expected_repr
    
    def test_book_str(self, db_session):
        """Test book string method"""
        book = Book(
            title="String Test",
            author="String Author",
            published_year=2023
        )
        
        db_session.add(book)
        db_session.commit()
        
        expected_str = "String Test by String Author (2023)"
        assert str(book) == expected_str
    
    def test_unique_constraint_violation(self, db_session):
        """Test that duplicate title-author combinations are not allowed"""
        # Create first book
        book1 = Book(
            title="Duplicate Test",
            author="Duplicate Author",
            published_year=2023
        )
        db_session.add(book1)
        db_session.commit()
        
        # Try to create second book with same title and author
        book2 = Book(
            title="Duplicate Test",
            author="Duplicate Author",
            published_year=2024  # Different year, but same title-author
        )
        db_session.add(book2)
        
        # Should raise an IntegrityError due to unique constraint
        with pytest.raises(Exception):  # SQLAlchemy will raise IntegrityError
            db_session.commit()
    
    def test_books_with_same_title_different_authors(self, db_session):
        """Test that books with same title but different authors are allowed"""
        book1 = Book(
            title="Common Title",
            author="Author One",
            published_year=2023
        )
        
        book2 = Book(
            title="Common Title",
            author="Author Two",
            published_year=2023
        )
        
        db_session.add_all([book1, book2])
        db_session.commit()
        
        # Both books should be created successfully
        assert book1.id is not None
        assert book2.id is not None
        assert book1.id != book2.id
    
    def test_books_with_same_author_different_titles(self, db_session):
        """Test that books with same author but different titles are allowed"""
        book1 = Book(
            title="First Book",
            author="Prolific Author",
            published_year=2023
        )
        
        book2 = Book(
            title="Second Book",
            author="Prolific Author",
            published_year=2024
        )
        
        db_session.add_all([book1, book2])
        db_session.commit()
        
        # Both books should be created successfully
        assert book1.id is not None
        assert book2.id is not None
        assert book1.id != book2.id