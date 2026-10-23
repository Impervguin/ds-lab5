from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from typing import Any

import jwt

DEFAULT_USERNAME_CLAIM = "sub"

ALGORITHMS = ["RS256", "RS384", "RS512", "PS256", "PS384", "PS512", "ES256", "ES384", "ES512"]

JWKS_TIMEOUT_SECONDS = 5

KeyResolver = Callable[[str], Any]


class InvalidToken(Exception):
    pass


@dataclass(frozen=True, slots=True)
class Identity:
    username: str
    token: str


class TokenVerifier:
    def __init__(
        self,
        issuer: str,
        audience: str,
        resolve_key: KeyResolver,
        username_claim: str = DEFAULT_USERNAME_CLAIM,
    ) -> None:
        if not issuer:
            raise ValueError("issuer is required")
        if not audience:
            raise ValueError("audience is required")

        self._issuer = issuer
        self._audience = audience
        self._resolve_key = resolve_key
        self._username_claim = username_claim or DEFAULT_USERNAME_CLAIM

    @classmethod
    def from_jwks(
        cls, issuer: str, audience: str, jwks_url: str, username_claim: str
    ) -> TokenVerifier:
        if not jwks_url:
            raise ValueError("jwks url is required")

        client = jwt.PyJWKClient(jwks_url, timeout=JWKS_TIMEOUT_SECONDS)
        return cls(
            issuer,
            audience,
            lambda raw: client.get_signing_key_from_jwt(raw).key,
            username_claim,
        )

    def verify(self, raw: str) -> Identity:
        try:
            claims = jwt.decode(
                raw,
                self._resolve_key(raw),
                algorithms=ALGORITHMS,
                issuer=self._issuer,
                audience=self._audience,
                options={"require": ["exp", "iss", "aud"]},
            )
        except jwt.PyJWTError as exc:
            raise InvalidToken(str(exc)) from exc

        username = claims.get(self._username_claim)
        if not isinstance(username, str) or not username:
            raise InvalidToken(f"claim {self._username_claim!r} is missing")
        return Identity(username=username, token=raw)
