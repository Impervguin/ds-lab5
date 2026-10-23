from types import TracebackType
from typing import Self

from rating.domain import Rating


class FakeRatingRepository:
    def __init__(self, ratings: list[Rating] | None = None) -> None:
        self.ratings = {rating.username: rating for rating in ratings or []}

    async def get_or_create(self, initial: Rating) -> Rating:
        return self.ratings.setdefault(initial.username, initial)

    async def update_stars(self, username: str, stars: int) -> None:
        if username not in self.ratings:
            raise AssertionError(f"Rating of {username} is not stored")
        self.ratings[username] = self.ratings[username].model_copy(update={"stars": stars})


class FakeUnitOfWork:
    def __init__(self, ratings: FakeRatingRepository | None = None) -> None:
        self.ratings = ratings or FakeRatingRepository()
        self.committed = False
        self.rolled_back = False

    async def __aenter__(self) -> Self:
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc: BaseException | None,
        tb: TracebackType | None,
    ) -> None:
        if exc_type is None:
            self.committed = True
        else:
            self.rolled_back = True

    async def rollback(self) -> None:
        self.rolled_back = True
