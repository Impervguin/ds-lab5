from reservation.domain import ReservationStatus
from reservation.repositories import UnitOfWorkFactory


class CountReservationsUseCase:
    def __init__(self, uow_factory: UnitOfWorkFactory) -> None:
        self._uow_factory = uow_factory

    async def execute(self, username: str, status: ReservationStatus | None = None) -> int:
        async with self._uow_factory() as uow:
            return await uow.reservations.count(username, status)
