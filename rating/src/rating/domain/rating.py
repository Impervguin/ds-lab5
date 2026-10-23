from __future__ import annotations

from enum import StrEnum
from typing import Final

from pydantic import BaseModel, ConfigDict, Field

MIN_STARS: Final = 1
MAX_STARS: Final = 100
INITIAL_STARS: Final = 75
STARS_PER_BOOK: Final = 3


class ReservationStatus(StrEnum):
    RENTED = "RENTED"
    RETURNED = "RETURNED"
    EXPIRED = "EXPIRED"


class BookCondition(StrEnum):
    EXCELLENT = "EXCELLENT"
    GOOD = "GOOD"
    BAD = "BAD"


class Rating(BaseModel):
    model_config = ConfigDict(frozen=True)

    username: str = Field(min_length=1, max_length=80)
    stars: int = Field(ge=MIN_STARS, le=MAX_STARS)

    @staticmethod
    def initial(username: str) -> Rating:
        return Rating(username=username, stars=INITIAL_STARS)

    @property
    def max_books(self) -> int:
        return self.stars // STARS_PER_BOOK

    def apply(self, delta: int) -> Rating:
        stars = min(max(self.stars + delta, MIN_STARS), MAX_STARS)
        return self.model_copy(update={"stars": stars})


class RatingChange(BaseModel):
    model_config = ConfigDict(frozen=True)

    rating: Rating
    delta: int

    @staticmethod
    def between(before: Rating, after: Rating) -> RatingChange:
        return RatingChange(rating=after, delta=after.stars - before.stars)
