from __future__ import annotations

from datetime import date
from uuid import UUID

from reservation.api.common import CamelModel
from reservation.domain import BookCondition, Reservation, ReservationStatus


class ReservationResponse(CamelModel):
    reservation_uid: UUID
    book_uid: UUID
    library_uid: UUID
    status: ReservationStatus
    start_date: date
    till_date: date
    condition_at_rent: BookCondition

    @classmethod
    def of(cls, reservation: Reservation) -> ReservationResponse:
        return cls.model_validate(reservation, from_attributes=True)
