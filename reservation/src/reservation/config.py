from __future__ import annotations

import os
from dataclasses import dataclass
from functools import lru_cache

from reservation.auth import DEFAULT_USERNAME_CLAIM

DEFAULT_HTTP_PORT = 8070
DEFAULT_DATABASE_URL = "postgresql://program:test@localhost:5432/reservations"
MIGRATION_LOCK_KEY = 8070


@dataclass(frozen=True, slots=True)
class Settings:
    http_port: int = DEFAULT_HTTP_PORT
    database_url: str = DEFAULT_DATABASE_URL
    migration_lock_key: int = MIGRATION_LOCK_KEY
    oidc_issuer: str = ""
    oidc_audience: str = ""
    oidc_jwks_url: str = ""
    oidc_username_claim: str = DEFAULT_USERNAME_CLAIM


@lru_cache
def get_settings() -> Settings:
    return Settings(
        http_port=_int("HTTP_PORT", DEFAULT_HTTP_PORT),
        database_url=os.getenv("DATABASE_URL", DEFAULT_DATABASE_URL),
        oidc_issuer=os.getenv("OIDC_ISSUER", ""),
        oidc_audience=os.getenv("OIDC_AUDIENCE", ""),
        oidc_jwks_url=os.getenv("OIDC_JWKS_URL", ""),
        oidc_username_claim=os.getenv("OIDC_USERNAME_CLAIM") or DEFAULT_USERNAME_CLAIM,
    )


def _int(name: str, fallback: int) -> int:
    value = os.getenv(name)
    return int(value) if value and value.lstrip("-").isdigit() else fallback
