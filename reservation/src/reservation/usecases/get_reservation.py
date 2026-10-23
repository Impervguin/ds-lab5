from uuid import UUID

from reservation.domain import Reservation, ReservationNotFound
from reservation.repositories import UnitOfWorkFactory


class GetReservationUseCase:
    def __init__(self, uow_factory: UnitOfWorkFactory) -> None:
        self._uow_factory = uow_factory

    async def execute(self, username: str, reservation_uid: UUID) -> Reservation:
        async with self._uow_factory() as uow:
            reservation = await uow.reservations.get(reservation_uid, username)

        if reservation is None:
            raise ReservationNotFound(reservation_uid)
        return reservation
