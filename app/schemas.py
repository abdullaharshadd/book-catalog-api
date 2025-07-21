from pydantic import BaseModel, field_validator, ConfigDict
from typing import Optional
from datetime import datetime


class BookCreate(BaseModel):
    title: str
    author: str
    published_year: int
    summary: Optional[str] = None

    model_config = ConfigDict(str_strip_whitespace=True)

    @field_validator('title')
    @classmethod
    def validate_title(cls, v: str) -> str:
        if not v or not v.strip():
            raise ValueError('Title cannot be empty')
        if len(v) > 255:
            raise ValueError('ensure this value has at most 255 characters')
        return v.strip()

    @field_validator('author')
    @classmethod
    def validate_author(cls, v: str) -> str:
        if not v or not v.strip():
            raise ValueError('Author cannot be empty')
        if len(v) > 255:
            raise ValueError('ensure this value has at most 255 characters')
        return v.strip()

    @field_validator('published_year')
    @classmethod
    def validate_published_year(cls, v: int) -> int:
        current_year = datetime.now().year
        if v < 1000:
            raise ValueError('Published year must be after year 1000')
        if v > current_year:
            raise ValueError(f'Published year cannot be in the future (current year: {current_year})')
        return v

    @field_validator('summary')
    @classmethod
    def validate_summary(cls, v: Optional[str]) -> Optional[str]:
        if v is None:
            return None
        v = v.strip()
        if not v:
            return None
        if len(v) > 2000:
            raise ValueError('ensure this value has at most 2000 characters')
        return v


class BookUpdate(BaseModel):
    title: Optional[str] = None
    author: Optional[str] = None
    published_year: Optional[int] = None
    summary: Optional[str] = None

    model_config = ConfigDict(str_strip_whitespace=True)

    @field_validator('title')
    @classmethod
    def validate_title(cls, v: Optional[str]) -> Optional[str]:
        if v is None:
            return None
        if not v or not v.strip():
            raise ValueError('Title cannot be empty')
        if len(v) > 255:
            raise ValueError('ensure this value has at most 255 characters')
        return v.strip()

    @field_validator('author')
    @classmethod
    def validate_author(cls, v: Optional[str]) -> Optional[str]:
        if v is None:
            return None
        if not v or not v.strip():
            raise ValueError('Author cannot be empty')
        if len(v) > 255:
            raise ValueError('ensure this value has at most 255 characters')
        return v.strip()

    @field_validator('published_year')
    @classmethod
    def validate_published_year(cls, v: Optional[int]) -> Optional[int]:
        if v is None:
            return None
        current_year = datetime.now().year
        if v < 1000:
            raise ValueError('Published year must be after year 1000')
        if v > current_year:
            raise ValueError(f'Published year cannot be in the future (current year: {current_year})')
        return v

    @field_validator('summary')
    @classmethod
    def validate_summary(cls, v: Optional[str]) -> Optional[str]:
        if v is None:
            return None
        v = v.strip()
        if not v:
            return None
        if len(v) > 2000:
            raise ValueError('ensure this value has at most 2000 characters')
        return v


class BookResponse(BaseModel):
    id: int
    title: str
    author: str
    published_year: int
    summary: Optional[str] = None

    model_config = ConfigDict(from_attributes=True)