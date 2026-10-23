from __future__ import annotations

from datetime import date
from enum import StrEnum
from uuid import UUID, uuid4

from pydantic import BaseModel, ConfigDict, Field

from .errors import ReservationAlreadyClosed


class ReservationStatus(StrEnum):
    RENTED = "RENTED"
    RETURNED = "RETURNED"
    EXPIRED = "EXPIRED"


class BookCondition(StrEnum):
    EXCELLENT = "EXCELLENT"
    GOOD = "GOOD"
    BAD = "BAD"


class Reservation(BaseModel):
    model_config = ConfigDict(frozen=True)

    reservation_uid: UUID
    username: str = Field(min_length=1, max_length=80)
    book_uid: UUID
    library_uid: UUID
    status: ReservationStatus
    start_date: date
    till_date: date
    condition_at_rent: BookCondition

    @staticmethod
    def open(
        username: str,
        book_uid: UUID,
        library_uid: UUID,
        till_date: date,
        condition_at_rent: BookCondition,
        *,
        start_date: date | None = None,
    ) -> Reservation:
        return Reservation(
            reservation_uid=uuid4(),
            username=username,
            book_uid=book_uid,
            library_uid=library_uid,
            status=ReservationStatus.RENTED,
            start_date=start_date or date.today(),
            till_date=till_date,
            condition_at_rent=condition_at_rent,
        )

    def close(self, return_date: date) -> Reservation:
        if self.status is not ReservationStatus.RENTED:
            raise ReservationAlreadyClosed(self.reservation_uid)

        status = (
            ReservationStatus.EXPIRED
            if return_date > self.till_date
            else ReservationStatus.RETURNED
        )
        return self.model_copy(update={"status": status})
