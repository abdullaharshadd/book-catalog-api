from fastapi import FastAPI, HTTPException, Depends, status
from fastapi.responses import JSONResponse
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import Session
from sqlalchemy.future import select
from sqlalchemy.exc import IntegrityError
from typing import List, Optional
import logging
import os

# Ensure DATABASE_URL points to PostgreSQL if not already set to a valid non-MySQL URL
_db_url = os.getenv("DATABASE_URL", "")
_async_db_url = os.getenv("ASYNC_DATABASE_URL", "")

# If DATABASE_URL is empty or points to MySQL/SQLite, override with PostgreSQL defaults
if not _db_url or "3306" in _db_url or _db_url.startswith("sqlite") or _db_url.startswith("mysql"):
    _pg_host = os.getenv("DB_HOST", os.getenv("POSTGRES_HOST", "db"))
    _pg_port = os.getenv("DB_PORT", os.getenv("POSTGRES_PORT", "5432"))
    _pg_user = os.getenv("DB_USER", os.getenv("POSTGRES_USER", os.getenv("PGUSER", "postgres")))
    _pg_pass = os.getenv("DB_PASSWORD", os.getenv("POSTGRES_PASSWORD", os.getenv("PGPASSWORD", "postgres")))
    _pg_db = os.getenv("DB_NAME", os.getenv("POSTGRES_DB", os.getenv("PGDATABASE", "app")))
    _db_url = f"postgresql://{_pg_user}:{_pg_pass}@{_pg_host}:{_pg_port}/{_pg_db}"
    _async_db_url = f"postgresql+asyncpg://{_pg_user}:{_pg_pass}@{_pg_host}:{_pg_port}/{_pg_db}"
    os.environ["DATABASE_URL"] = _db_url
    os.environ["ASYNC_DATABASE_URL"] = _async_db_url

from .database import get_db, get_sync_db, init_db
from .models import Book
from .schemas import BookCreate, BookUpdate, BookResponse

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Create FastAPI app
app = FastAPI(
    title="Book Catalog API",
    description="A simple CRUD service for managing books",
    version="1.0.0",
    docs_url="/docs",
    redoc_url="/redoc"
)


@app.on_event("startup")
async def startup_event():
    """Initialize database on startup"""
    await init_db()
    logger.info("Database initialized successfully")


@app.exception_handler(HTTPException)
async def http_exception_handler(request, exc):
    """Custom HTTP exception handler"""
    return JSONResponse(
        status_code=exc.status_code,
        content={"detail": exc.detail}
    )


@app.get("/", tags=["Root"])
async def root():
    """Root endpoint with API information"""
    return {
        "message": "Welcome to Book Catalog API",
        "version": "1.0.0",
        "docs_url": "/docs"
    }


@app.get("/books/", response_model=List[BookResponse], tags=["Books"])
async def list_books(
    skip: int = 0, 
    limit: int = 100, 
    db: AsyncSession = Depends(get_db)
):
    """
    Retrieve all books with pagination (async endpoint).
    
    - **skip**: Number of books to skip (default: 0)
    - **limit**: Maximum number of books to return (default: 100, max: 1000)
    """
    try:
        limit = min(limit, 1000)
        
        result = await db.execute(
            select(Book).offset(skip).limit(limit)
        )
        books = result.scalars().all()
        
        logger.info(f"Retrieved {len(books)} books (skip={skip}, limit={limit})")
        return books
        
    except Exception as e:
        logger.error(f"Error retrieving books: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error while retrieving books"
        )


@app.get("/books/{book_id}", response_model=BookResponse, tags=["Books"])
def get_book(book_id: int, db: Session = Depends(get_sync_db)):
    """
    Retrieve a single book by its ID.
    
    - **book_id**: The ID of the book to retrieve
    """
    try:
        result = db.execute(select(Book).filter(Book.id == book_id))
        book = result.scalar_one_or_none()
        
        if book is None:
            logger.warning(f"Book with ID {book_id} not found")
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Book with ID {book_id} not found"
            )
        
        logger.info(f"Retrieved book: {book.title}")
        return book
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error retrieving book {book_id}: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error while retrieving book"
        )


@app.post("/books/", response_model=BookResponse, status_code=status.HTTP_201_CREATED, tags=["Books"])
def create_book(book: BookCreate, db: Session = Depends(get_sync_db)):
    """
    Create a new book.
    
    - **title**: The book's title (required)
    - **author**: The book's author (required)
    - **published_year**: The year the book was published (required)
    - **summary**: Optional summary of the book
    """
    try:
        db_book = Book(**book.dict())
        db.add(db_book)
        db.commit()
        db.refresh(db_book)
        
        logger.info(f"Created new book: {db_book.title} by {db_book.author}")
        return db_book
        
    except IntegrityError as e:
        db.rollback()
        logger.error(f"Integrity error creating book: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Book with this title and author already exists"
        )
    except Exception as e:
        db.rollback()
        logger.error(f"Error creating book: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error while creating book"
        )


@app.put("/books/{book_id}", response_model=BookResponse, tags=["Books"])
def update_book(book_id: int, book_update: BookUpdate, db: Session = Depends(get_sync_db)):
    """
    Update an existing book.
    
    - **book_id**: The ID of the book to update
    - **title**: The book's title (optional)
    - **author**: The book's author (optional)
    - **published_year**: The year the book was published (optional)
    - **summary**: Summary of the book (optional)
    """
    try:
        result = db.execute(select(Book).filter(Book.id == book_id))
        db_book = result.scalar_one_or_none()
        
        if db_book is None:
            logger.warning(f"Book with ID {book_id} not found for update")
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Book with ID {book_id} not found"
            )
        
        update_data = book_update.dict(exclude_unset=True)
        for field, value in update_data.items():
            setattr(db_book, field, value)
        
        db.commit()
        db.refresh(db_book)
        
        logger.info(f"Updated book: {db_book.title}")
        return db_book
        
    except HTTPException:
        raise
    except IntegrityError as e:
        db.rollback()
        logger.error(f"Integrity error updating book: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Book with this title and author already exists"
        )
    except Exception as e:
        db.rollback()
        logger.error(f"Error updating book {book_id}: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error while updating book"
        )


@app.delete("/books/{book_id}", status_code=status.HTTP_204_NO_CONTENT, tags=["Books"])
def delete_book(book_id: int, db: Session = Depends(get_sync_db)):
    """
    Delete a book by its ID.
    
    - **book_id**: The ID of the book to delete
    """
    try:
        result = db.execute(select(Book).filter(Book.id == book_id))
        db_book = result.scalar_one_or_none()
        
        if db_book is None:
            logger.warning(f"Book with ID {book_id} not found for deletion")
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Book with ID {book_id} not found"
            )
        
        db.delete(db_book)
        db.commit()
        
        logger.info(f"Deleted book: {db_book.title}")
        return None
        
    except HTTPException:
        raise
    except Exception as e:
        db.rollback()
        logger.error(f"Error deleting book {book_id}: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error while deleting book"
        )


@app.get("/health", tags=["Health"])
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy", "service": "book-catalog-api"}