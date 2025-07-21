# app/models.py
from sqlalchemy import Column, Integer, String, Text, UniqueConstraint
from sqlalchemy.ext.declarative import declarative_base

Base = declarative_base()


class Book(Base):
    """
    Book model representing a book in the catalog.
    
    Attributes:
        id: Primary key, auto-incrementing integer
        title: Book title (required)
        author: Book author (required)
        published_year: Year the book was published (required)
        summary: Optional summary/description of the book
    """
    __tablename__ = "books"
    
    id = Column(Integer, primary_key=True, index=True, autoincrement=True)
    title = Column(String(255), nullable=False, index=True)
    author = Column(String(255), nullable=False, index=True)
    published_year = Column(Integer, nullable=False, index=True)
    summary = Column(Text, nullable=True)
    
    # Ensure unique combination of title and author
    __table_args__ = (
        UniqueConstraint('title', 'author', name='unique_title_author'),
    )
    
    def __repr__(self):
        return f"<Book(id={self.id}, title='{self.title}', author='{self.author}', year={self.published_year})>"
    
    def __str__(self):
        return f"{self.title} by {self.author} ({self.published_year})"