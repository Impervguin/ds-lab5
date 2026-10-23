from dependency_injector import containers, providers
from psycopg_pool import AsyncConnectionPool

from rating.auth import TokenVerifier
from rating.config import get_settings
from rating.repositories.psycopg import PsycopgUnitOfWork
from rating.usecases import CloseReservationUseCase, GetRatingUseCase


class Container(containers.DeclarativeContainer):
    config = providers.Singleton(get_settings)

    pool = providers.Singleton(
        AsyncConnectionPool,
        conninfo=config.provided.database_url,
        open=False,
    )

    token_verifier = providers.Singleton(
        TokenVerifier.from_jwks,
        issuer=config.provided.oidc_issuer,
        audience=config.provided.oidc_audience,
        jwks_url=config.provided.oidc_jwks_url,
        username_claim=config.provided.oidc_username_claim,
    )

    uow = providers.Factory(PsycopgUnitOfWork, pool=pool)

    get_rating_use_case = providers.Factory(GetRatingUseCase, uow_factory=uow.provider)
    close_reservation_use_case = providers.Factory(
        CloseReservationUseCase, uow_factory=uow.provider
    )
