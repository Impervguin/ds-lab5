from collections.abc import Callable
from types import TracebackType
from typing import Protocol, Self, runtime_checkable

from rating.domain import Rating


@runtime_checkable
class RatingRepository(Protocol):
    async def get_or_create(self, initial: Rating) -> Rating: ...

    async def update_stars(self, username: str, stars: int) -> None: ...


@runtime_checkable
class UnitOfWork(Protocol):
    @property
    def ratings(self) -> RatingRepository: ...

    async def __aenter__(self) -> Self: ...

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc: BaseException | None,
        tb: TracebackType | None,
    ) -> bool | None: ...

    async def rollback(self) -> None: ...


UnitOfWorkFactory = Callable[[], UnitOfWork]
