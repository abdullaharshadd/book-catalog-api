from fastapi import FastAPI, HTTPException, Depends, status
from fastapi.responses import JSONResponse
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import Session
from sqlalchemy.future import select
from sqlalchemy.exc import IntegrityError
from typing import List, Optional
import logging

from .database import get_db, get_sync_db, init_db
from .models import Book
from .schemas import BookCreate, BookUpdate, BookResponse

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

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
    return JSONResponse(
        status_code=exc.status_code,
        content={"detail": exc.detail}
    )


@app.get("/", tags=["Root"])
async def root():
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
    try:
        result = db.execute(select(Book).filter(Book.id == book_id))
        book = result.scalar_one_or_none()
        if book is None:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Book with ID {book_id} not found"
            )
        return book
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error while retrieving book"
        )


@app.post("/books/", response_model=BookResponse, status_code=status.HTTP_201_CREATED, tags=["Books"])
def create_book(book: BookCreate, db: Session = Depends(get_sync_db)):
    try:
        db_book = Book(**book.dict())
        db.add(db_book)
        db.commit()
        db.refresh(db_book)
        return db_book
    except IntegrityError:
        db.rollback()
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Book with this title and author already exists"
        )
    except Exception as e:
        db.rollback()
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error while creating book"
        )


@app.put("/books/{book_id}", response_model=BookResponse, tags=["Books"])
def update_book(book_id: int, book_update: BookUpdate, db: Session = Depends(get_sync_db)):
    try:
        result = db.execute(select(Book).filter(Book.id == book_id))
        db_book = result.scalar_one_or_none()
        if db_book is None:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Book with ID {book_id} not found"
            )
        update_data = book_update.dict(exclude_unset=True)
        for field, value in update_data.items():
            setattr(db_book, field, value)
        db.commit()
        db.refresh(db_book)
        return db_book
    except HTTPException:
        raise
    except IntegrityError:
        db.rollback()
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Book with this title and author already exists"
        )
    except Exception as e:
        db.rollback()
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error while updating book"
        )


@app.delete("/books/{book_id}", status_code=status.HTTP_204_NO_CONTENT, tags=["Books"])
def delete_book(book_id: int, db: Session = Depends(get_sync_db)):
    try:
        result = db.execute(select(Book).filter(Book.id == book_id))
        db_book = result.scalar_one_or_none()
        if db_book is None:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Book with ID {book_id} not found"
            )
        db.delete(db_book)
        db.commit()
        return None
    except HTTPException:
        raise
    except Exception as e:
        db.rollback()
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Internal server error while deleting book"
        )


@app.get("/health", tags=["Health"])
async def health_check():
    return {"status": "healthy", "service": "book-catalog-api"}