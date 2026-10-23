from datetime import date
from uuid import UUID

from reservation.domain import BookCondition, Reservation
from reservation.logging import get_logger
from reservation.repositories import UnitOfWorkFactory

logger = get_logger(__name__)


class CreateReservationUseCase:
    def __init__(self, uow_factory: UnitOfWorkFactory) -> None:
        self._uow_factory = uow_factory

    async def execute(
        self,
        username: str,
        book_uid: UUID,
        library_uid: UUID,
        till_date: date,
        condition_at_rent: BookCondition,
    ) -> Reservation:
        reservation = Reservation.open(
            username=username,
            book_uid=book_uid,
            library_uid=library_uid,
            till_date=till_date,
            condition_at_rent=condition_at_rent,
        )

        async with self._uow_factory() as uow:
            await uow.reservations.add(reservation)

        logger.info(
            "reservation %s created for %s: book=%s library=%s condition=%s",
            reservation.reservation_uid,
            username,
            book_uid,
            library_uid,
            condition_at_rent,
        )
        return reservation
