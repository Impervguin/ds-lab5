from uuid import UUID

from rating.domain import (
    BookCondition,
    Rating,
    RatingChange,
    ReservationStatus,
    closing_delta,
)
from rating.logging import get_logger
from rating.repositories import UnitOfWorkFactory

logger = get_logger(__name__)


class CloseReservationUseCase:
    def __init__(self, uow_factory: UnitOfWorkFactory) -> None:
        self._uow_factory = uow_factory

    async def execute(
        self,
        username: str,
        reservation_uid: UUID,
        status: ReservationStatus,
        condition_at_rent: BookCondition,
        condition_on_return: BookCondition,
    ) -> RatingChange:
        delta = closing_delta(status, condition_at_rent, condition_on_return)

        async with self._uow_factory() as uow:
            before = await uow.ratings.get_or_create(Rating.initial(username))
            after = before.apply(delta)
            if after.stars != before.stars:
                await uow.ratings.update_stars(username, after.stars)

        change = RatingChange.between(before, after)
        logger.info(
            "reservation %s closed for %s: status=%s condition %s -> %s, rating %d -> %d",
            reservation_uid,
            username,
            status,
            condition_at_rent,
            condition_on_return,
            before.stars,
            change.rating.stars,
        )
        return change
