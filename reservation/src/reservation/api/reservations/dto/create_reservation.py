from datetime import date
from uuid import UUID

from reservation.api.common import CamelModel
from reservation.domain import BookCondition


class CreateReservationRequest(CamelModel):
    book_uid: UUID
    library_uid: UUID
    till_date: date
    condition_at_rent: BookCondition
