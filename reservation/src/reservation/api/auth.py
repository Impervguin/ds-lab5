from typing import Annotated

from dependency_injector.wiring import Provide, inject
from fastapi import Depends
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

from reservation.auth import Identity, InvalidToken, TokenVerifier
from reservation.di import Container
from reservation.logging import get_logger

logger = get_logger(__name__)

# auto_error=False: a missing token is answered with our 401, not FastAPI's 403.
_bearer = HTTPBearer(auto_error=False)


class Unauthorized(Exception):
    def __init__(self, message: str) -> None:
        super().__init__(message)
        self.message = message


# Sync on purpose: fetching the JWKs blocks, so FastAPI runs it in a thread.
@inject
def current_identity(
    credentials: Annotated[HTTPAuthorizationCredentials | None, Depends(_bearer)],
    verifier: TokenVerifier = Depends(Provide[Container.token_verifier]),
) -> Identity:
    if credentials is None or not credentials.credentials:
        raise Unauthorized("Authorization token is required")

    try:
        return verifier.verify(credentials.credentials)
    except InvalidToken as exc:
        logger.info("token rejected: %s", exc)
        raise Unauthorized("Invalid authorization token") from exc


def current_username(identity: Annotated[Identity, Depends(current_identity)]) -> str:
    return identity.username


Username = Annotated[str, Depends(current_username)]
