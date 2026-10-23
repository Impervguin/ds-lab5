from datetime import date
from uuid import UUID

from reservation.domain import Reservation, ReservationNotFound
from reservation.logging import get_logger
from reservation.repositories import UnitOfWorkFactory

logger = get_logger(__name__)


class ReturnReservationUseCase:
    def __init__(self, uow_factory: UnitOfWorkFactory) -> None:
        self._uow_factory = uow_factory

    async def execute(self, username: str, reservation_uid: UUID, return_date: date) -> Reservation:
        async with self._uow_factory() as uow:
            rented = await uow.reservations.get(reservation_uid, username, for_update=True)
            if rented is None:
                raise ReservationNotFound(reservation_uid)

            closed = rented.close(return_date)
            await uow.reservations.update_status(reservation_uid, closed.status)

        logger.info(
            "reservation %s closed for %s: status=%s condition_at_rent=%s",
            reservation_uid,
            username,
            closed.status,
            closed.condition_at_rent,
        )
        return closed
