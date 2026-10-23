from collections.abc import Callable
from types import TracebackType
from typing import Protocol, Self, runtime_checkable
from uuid import UUID

from reservation.domain import Reservation, ReservationStatus


@runtime_checkable
class ReservationRepository(Protocol):
    async def list(
        self, username: str, status: ReservationStatus | None = None
    ) -> list[Reservation]: ...

    async def count(self, username: str, status: ReservationStatus | None = None) -> int: ...

    async def get(
        self, reservation_uid: UUID, username: str, *, for_update: bool = False
    ) -> Reservation | None: ...

    async def add(self, reservation: Reservation) -> None: ...

    async def update_status(self, reservation_uid: UUID, status: ReservationStatus) -> None: ...


@runtime_checkable
class UnitOfWork(Protocol):
    @property
    def reservations(self) -> ReservationRepository: ...

    async def __aenter__(self) -> Self: ...

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc: BaseException | None,
        tb: TracebackType | None,
    ) -> bool | None: ...

    async def rollback(self) -> None: ...


UnitOfWorkFactory = Callable[[], UnitOfWork]
