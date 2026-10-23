import json
import threading
from collections.abc import Iterator
from http.server import BaseHTTPRequestHandler, HTTPServer
from typing import Any

import jwt
import pytest
from cryptography.hazmat.primitives.asymmetric import rsa

from reservation.auth import Identity, InvalidToken, TokenVerifier

from .tokens import AUDIENCE, ISSUER, KEY_ID, SIGNING_KEY, claims, sign, verifier

USERNAME = "auth0|test-user"


def test_verify_returns_the_identity():
    raw = sign(claims(USERNAME))

    assert verifier().verify(raw) == Identity(username=USERNAME, token=raw)


def test_verify_takes_the_username_from_the_configured_claim():
    assert verifier("email").verify(sign(claims(USERNAME))).username == "test@test.com"


def test_verify_accepts_a_single_audience():
    assert verifier().verify(sign(claims(USERNAME, aud=AUDIENCE))).username == USERNAME


def _without(name: str) -> dict[str, Any]:
    payload = claims(USERNAME)
    del payload[name]
    return payload


@pytest.mark.parametrize(
    "raw",
    [
        pytest.param("not-a-jwt", id="garbage"),
        pytest.param(sign(claims(USERNAME, exp=1)), id="expired"),
        pytest.param(sign(_without("exp")), id="without exp"),
        pytest.param(sign(claims(USERNAME, aud="https://another.test/api")), id="foreign audience"),
        pytest.param(sign(claims(USERNAME, iss="https://evil.test/")), id="foreign issuer"),
        pytest.param(sign(_without("sub")), id="without username"),
        pytest.param(
            sign(claims(USERNAME), rsa.generate_private_key(public_exponent=65537, key_size=2048)),
            id="foreign signature",
        ),
        pytest.param(
            sign(claims(USERNAME), "a-symmetric-secret-long-enough-for-hs256", "HS256"),
            id="symmetric algorithm",
        ),
    ],
)
def test_verify_rejects_an_invalid_token(raw):
    with pytest.raises(InvalidToken):
        verifier().verify(raw)


@pytest.fixture
def jwks_url() -> Iterator[str]:
    key = json.loads(jwt.algorithms.RSAAlgorithm.to_jwk(SIGNING_KEY.public_key()))
    body = json.dumps({"keys": [key | {"kid": KEY_ID, "use": "sig", "alg": "RS256"}]}).encode()

    class JWKs(BaseHTTPRequestHandler):
        def do_GET(self) -> None:
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, *_args: Any) -> None:
            pass

    server = HTTPServer(("127.0.0.1", 0), JWKs)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    yield f"http://127.0.0.1:{server.server_port}/.well-known/jwks.json"
    server.shutdown()


def test_verify_checks_tokens_against_the_remote_jwks(jwks_url):
    remote = TokenVerifier.from_jwks(ISSUER, AUDIENCE, jwks_url, "sub")

    assert remote.verify(sign(claims(USERNAME))).username == USERNAME


@pytest.mark.parametrize(
    ("issuer", "audience", "jwks_url"),
    [("", AUDIENCE, "http://jwks"), (ISSUER, "", "http://jwks"), (ISSUER, AUDIENCE, "")],
    ids=["issuer", "audience", "jwks url"],
)
def test_from_jwks_demands_the_provider_settings(issuer, audience, jwks_url):
    with pytest.raises(ValueError):
        TokenVerifier.from_jwks(issuer, audience, jwks_url, "sub")
