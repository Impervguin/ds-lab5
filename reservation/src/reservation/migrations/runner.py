from importlib import resources
from pathlib import Path

import psycopg
from yoyo import get_backend, read_migrations

from reservation.logging import get_logger

logger = get_logger(__name__)

YOYO_SCHEME = "postgresql+psycopg://"
KNOWN_SCHEMES = ("postgresql+psycopg://", "postgresql://", "postgres://")


def apply_migrations(database_url: str, lock_key: int) -> None:
    logger.info("Applying migrations")
    with psycopg.connect(database_url) as conn, conn.cursor() as cur:
        logger.info("Acquiring migration advisory lock")
        cur.execute("SELECT pg_try_advisory_lock(%s)", (lock_key,))
        row = cur.fetchone()
        if not (row and row[0]):
            logger.info("Migration advisory lock is already held")
            return

        logger.info("Migration advisory lock acquired")
        try:
            _run_yoyo(database_url)
        finally:
            cur.execute("SELECT pg_advisory_unlock(%s)", (lock_key,))
            logger.info("Migration advisory lock released")


def _run_yoyo(database_url: str) -> None:
    backend = None
    try:
        backend = get_backend(_to_yoyo_url(database_url))
        migrations = read_migrations(str(_migrations_dir()))
        to_apply = backend.to_apply(migrations)
        if not to_apply:
            logger.info("No migrations to apply")
            return

        logger.info("Applying %d migration(s)", len(to_apply))
        backend.apply_migrations(to_apply)
        logger.info("Migrations applied")
    finally:
        if backend is not None and backend.connection is not None:
            backend.connection.close()


def _migrations_dir() -> Path:
    return Path(str(resources.files(__package__) / "sql"))


def _to_yoyo_url(url: str) -> str:
    for scheme in KNOWN_SCHEMES:
        if url.startswith(scheme):
            return YOYO_SCHEME + url[len(scheme) :]
    raise ValueError(f"Unsupported database URL scheme: {url!r}")
