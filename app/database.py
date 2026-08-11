# app/database.py
from sqlalchemy import create_engine
from sqlalchemy.ext.asyncio import AsyncSession, create_async_engine
from sqlalchemy.orm import sessionmaker
import os
from .models import Base

# Database configuration - use postgres://app:app@db:5432/app?sslmode=disable
DATABASE_URL = os.getenv(
    "DATABASE_URL",
    os.getenv(
        "DB_URL",
        "postgresql://app:app@db:5432/app"
    )
)

# Build async URL from sync URL
def _make_async_url(url: str) -> str:
    if url.startswith("postgresql://"):
        return url.replace("postgresql://", "postgresql+asyncpg://", 1)
    if url.startswith("postgres://"):
        return url.replace("postgres://", "postgresql+asyncpg://", 1)
    return url

def _make_sync_url(url: str) -> str:
    if url.startswith("postgresql+asyncpg://"):
        return url.replace("postgresql+asyncpg://", "postgresql://", 1)
    if url.startswith("postgres://"):
        return url.replace("postgres://", "postgresql://", 1)
    return url

SYNC_DATABASE_URL = _make_sync_url(DATABASE_URL)
ASYNC_DATABASE_URL = _make_async_url(DATABASE_URL)

# Remove sslmode from URL for asyncpg (it uses connect_args instead)
import re
ASYNC_DATABASE_URL_CLEAN = re.sub(r'\?sslmode=\w+', '', ASYNC_DATABASE_URL)
SYNC_DATABASE_URL_CLEAN = SYNC_DATABASE_URL

# Create async engine for async operations
async_engine = create_async_engine(
    ASYNC_DATABASE_URL_CLEAN,
    echo=True,
    future=True,
)

# Create sync engine for sync operations
sync_engine = create_engine(
    SYNC_DATABASE_URL_CLEAN,
    echo=True,
)

# Session factories
AsyncSessionLocal = sessionmaker(
    async_engine,
    class_=AsyncSession,
    expire_on_commit=False
)

SyncSessionLocal = sessionmaker(
    sync_engine,
    autocommit=False,
    autoflush=False,
)


async def init_db():
    """Initialize database tables"""
    async with async_engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)


async def get_db():
    """Dependency for async database sessions"""
    async with AsyncSessionLocal() as session:
        try:
            yield session
            await session.commit()
        except Exception:
            await session.rollback()
            raise
        finally:
            await session.close()


def get_sync_db():
    """Dependency for sync database sessions"""
    db = SyncSessionLocal()
    try:
        yield db
    finally:
        db.close()