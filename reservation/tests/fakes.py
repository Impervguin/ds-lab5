from types import TracebackType
from typing import Self
from uuid import UUID

from reservation.domain import Reservation, ReservationStatus


class FakeReservationRepository:
    def __init__(self, reservations: list[Reservation] | None = None) -> None:
        self.reservations = list(reservations or [])

    async def list(
        self, username: str, status: ReservationStatus | None = None
    ) -> list[Reservation]:
        return [r for r in self.reservations if _matches(r, username, status)]

    async def count(self, username: str, status: ReservationStatus | None = None) -> int:
        return len(await self.list(username, status))

    async def get(
        self, reservation_uid: UUID, username: str, *, for_update: bool = False
    ) -> Reservation | None:
        return next(
            (
                r
                for r in self.reservations
                if r.reservation_uid == reservation_uid and r.username == username
            ),
            None,
        )

    async def add(self, reservation: Reservation) -> None:
        self.reservations.append(reservation)

    async def update_status(self, reservation_uid: UUID, status: ReservationStatus) -> None:
        for index, reservation in enumerate(self.reservations):
            if reservation.reservation_uid == reservation_uid:
                self.reservations[index] = reservation.model_copy(update={"status": status})
                return
        raise AssertionError(f"Reservation {reservation_uid} is not stored")


class FakeUnitOfWork:
    def __init__(self, reservations: FakeReservationRepository | None = None) -> None:
        self.reservations = reservations or FakeReservationRepository()
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


def _matches(
    reservation: Reservation, username: str, status: ReservationStatus | None
) -> bool:
    return reservation.username == username and (status is None or reservation.status == status)
