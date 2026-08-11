# app/schemas.py
from pydantic import BaseModel, Field
from typing import Optional
from datetime import datetime


class BookCreate(BaseModel):
    title: str = Field(..., min_length=1, max_length=255)
    author: str = Field(..., min_length=1, max_length=255)
    published_year: int = Field(..., ge=1000, le=9999)
    summary: Optional[str] = None


class BookUpdate(BaseModel):
    title: Optional[str] = Field(None, min_length=1, max_length=255)
    author: Optional[str] = Field(None, min_length=1, max_length=255)
    published_year: Optional[int] = Field(None, ge=1000, le=9999)
    summary: Optional[str] = None


class BookResponse(BaseModel):
    id: int
    title: str
    author: str
    published_year: int
    summary: Optional[str] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    class Config:
        from_attributes = True
        # For older pydantic v1 compatibility
        orm_mode = True