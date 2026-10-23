from __future__ import annotations

from uuid import UUID

from rating.api.common import CamelModel
from rating.domain import BookCondition, Rating, RatingChange, ReservationStatus


class RatingResponse(CamelModel):
    stars: int
    max_books: int

    @classmethod
    def of(cls, rating: Rating) -> RatingResponse:
        return cls(stars=rating.stars, max_books=rating.max_books)


class ReservationClosedRequest(CamelModel):
    reservation_uid: UUID
    reservation_status: ReservationStatus
    condition_at_rent: BookCondition
    condition_on_return: BookCondition


class RatingChangeResponse(CamelModel):
    delta: int
    stars: int

    @classmethod
    def of(cls, change: RatingChange) -> RatingChangeResponse:
        return cls(delta=change.delta, stars=change.rating.stars)
