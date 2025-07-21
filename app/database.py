# app/database.py
from sqlalchemy import create_engine
from sqlalchemy.ext.asyncio import AsyncSession, create_async_engine
from sqlalchemy.orm import sessionmaker
import os
from .models import Base

# Database configuration
DATABASE_URL = os.getenv("DATABASE_URL", "sqlite:///./books.db")
ASYNC_DATABASE_URL = os.getenv("ASYNC_DATABASE_URL", "sqlite+aiosqlite:///./books.db")

# Create async engine for async operations
async_engine = create_async_engine(
    ASYNC_DATABASE_URL,
    echo=True,  # Set to False in production
    future=True,
    connect_args={"check_same_thread": False} if "sqlite" in ASYNC_DATABASE_URL else {}
)

# Create sync engine for sync operations (like creating tables)
sync_engine = create_engine(
    DATABASE_URL,
    echo=True,  # Set to False in production
    connect_args={"check_same_thread": False} if "sqlite" in DATABASE_URL else {}
)

# Create async session factory
async_session = sessionmaker(
    async_engine,
    class_=AsyncSession,
    expire_on_commit=False
)

# Create sync session factory
sync_session = sessionmaker(
    autocommit=False,
    autoflush=False,
    bind=sync_engine
)


async def init_db():
    """Initialize the database by creating all tables"""
    # Use sync engine to create tables
    Base.metadata.create_all(bind=sync_engine)


async def get_db() -> AsyncSession:
    """
    Dependency to get database session.
    This creates a new session for each request and closes it when done.
    """
    async with async_session() as session:
        try:
            yield session
        except Exception:
            await session.rollback()
            raise
        finally:
            await session.close()


def get_sync_db():
    """
    Dependency to get sync database session.
    Used for testing and synchronous operations.
    """
    db = sync_session()
    try:
        yield db
    except Exception:
        db.rollback()
        raise
    finally:
        db.close()