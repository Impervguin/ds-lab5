from reservation.domain import Reservation, ReservationStatus
from reservation.repositories import UnitOfWorkFactory


class ListReservationsUseCase:
    def __init__(self, uow_factory: UnitOfWorkFactory) -> None:
        self._uow_factory = uow_factory

    async def execute(
        self, username: str, status: ReservationStatus | None = None
    ) -> list[Reservation]:
        async with self._uow_factory() as uow:
            return await uow.reservations.list(username, status)
