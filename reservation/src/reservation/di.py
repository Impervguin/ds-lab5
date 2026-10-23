from dependency_injector import containers, providers
from psycopg_pool import AsyncConnectionPool

from reservation.config import get_settings
from reservation.repositories.psycopg import PsycopgUnitOfWork
from reservation.usecases import (
    CountReservationsUseCase,
    CreateReservationUseCase,
    GetReservationUseCase,
    ListReservationsUseCase,
    ReturnReservationUseCase,
)


class Container(containers.DeclarativeContainer):
    config = providers.Singleton(get_settings)

    pool = providers.Singleton(
        AsyncConnectionPool,
        conninfo=config.provided.database_url,
        open=False,
    )

    uow = providers.Factory(PsycopgUnitOfWork, pool=pool)

    list_reservations_use_case = providers.Factory(ListReservationsUseCase, uow_factory=uow.provider)
    count_reservations_use_case = providers.Factory(
        CountReservationsUseCase, uow_factory=uow.provider
    )
    get_reservation_use_case = providers.Factory(GetReservationUseCase, uow_factory=uow.provider)
    create_reservation_use_case = providers.Factory(
        CreateReservationUseCase, uow_factory=uow.provider
    )
    return_reservation_use_case = providers.Factory(
        ReturnReservationUseCase, uow_factory=uow.provider
    )
