import time
from typing import Any

import jwt
from cryptography.hazmat.primitives.asymmetric import rsa

from reservation.auth import TokenVerifier

ISSUER = "https://idp.test/"
AUDIENCE = "https://library.test/api"
KEY_ID = "test-key"

SIGNING_KEY = rsa.generate_private_key(public_exponent=65537, key_size=2048)


def claims(username: str, **overrides: Any) -> dict[str, Any]:
    now = int(time.time())
    return {
        "iss": ISSUER,
        "sub": username,
        "aud": [AUDIENCE, ISSUER + "userinfo"],
        "iat": now,
        "exp": now + 3600,
        "email": "test@test.com",
    } | overrides


def sign(payload: dict[str, Any], key: Any = SIGNING_KEY, algorithm: str = "RS256") -> str:
    return jwt.encode(payload, key, algorithm=algorithm, headers={"kid": KEY_ID})


def bearer(username: str) -> dict[str, str]:
    return {"Authorization": "Bearer " + sign(claims(username))}


def verifier(username_claim: str = "sub") -> TokenVerifier:
    return TokenVerifier(ISSUER, AUDIENCE, lambda _raw: SIGNING_KEY.public_key(), username_claim)
