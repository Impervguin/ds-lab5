from typing import Any, Final
from uuid import UUID

from psycopg import sql

from reservation.domain import Reservation, ReservationStatus

from ..errors import RepositoryError
from ..protocols import ReservationRepository
from .base import PsycopgRepository

TABLE: Final = sql.Identifier("reservation")

COLUMNS: Final[tuple[str, ...]] = (
    "reservation_uid",
    "username",
    "book_uid",
    "library_uid",
    "status",
    "start_date",
    "till_date",
    "condition_at_rent",
)

COLUMN_LIST: Final = sql.SQL(", ").join(sql.Identifier(column) for column in COLUMNS)
PLACEHOLDERS: Final = sql.SQL(", ").join(sql.Placeholder(column) for column in COLUMNS)

OWNED_BY_USER: Final = sql.SQL("username = %(username)s")
OF_STATUS: Final = sql.SQL("(%(status)s::varchar IS NULL OR status = %(status)s)")
FOR_UPDATE: Final = sql.SQL("FOR UPDATE")
NO_LOCK: Final = sql.SQL("")


class PsycopgReservationRepository(PsycopgRepository, ReservationRepository):
    async def list(
        self, username: str, status: ReservationStatus | None = None
    ) -> list[Reservation]:
        query = sql.SQL(
            "SELECT {columns} FROM {table} WHERE {owned} AND {of_status}"
            " ORDER BY start_date DESC, id DESC"
        ).format(columns=COLUMN_LIST, table=TABLE, owned=OWNED_BY_USER, of_status=OF_STATUS)

        async with self._cursor() as cur:
            await cur.execute(query, {"username": username, "status": _status(status)})
            return [_to_domain(row) for row in await cur.fetchall()]

    async def count(self, username: str, status: ReservationStatus | None = None) -> int:
        query = sql.SQL("SELECT COUNT(*) AS total FROM {table} WHERE {owned} AND {of_status}").format(
            table=TABLE, owned=OWNED_BY_USER, of_status=OF_STATUS
        )

        async with self._cursor() as cur:
            await cur.execute(query, {"username": username, "status": _status(status)})
            row = await cur.fetchone()
            if row is None:
                raise RepositoryError("Failed to count reservations")
            return int(row["total"])

    async def get(
        self, reservation_uid: UUID, username: str, *, for_update: bool = False
    ) -> Reservation | None:
        query = sql.SQL(
            "SELECT {columns} FROM {table}"
            " WHERE reservation_uid = %(reservation_uid)s AND {owned} {lock}"
        ).format(
            columns=COLUMN_LIST,
            table=TABLE,
            owned=OWNED_BY_USER,
            lock=FOR_UPDATE if for_update else NO_LOCK,
        )

        async with self._cursor() as cur:
            await cur.execute(query, {"reservation_uid": reservation_uid, "username": username})
            row = await cur.fetchone()
            return _to_domain(row) if row else None

    async def add(self, reservation: Reservation) -> None:
        query = sql.SQL("INSERT INTO {table} ({columns}) VALUES ({values})").format(
            table=TABLE, columns=COLUMN_LIST, values=PLACEHOLDERS
        )

        async with self._cursor() as cur:
            await cur.execute(query, reservation.model_dump(mode="python"))

    async def update_status(self, reservation_uid: UUID, status: ReservationStatus) -> None:
        query = sql.SQL(
            "UPDATE {table} SET status = %(status)s"
            " WHERE reservation_uid = %(reservation_uid)s"
        ).format(table=TABLE)

        async with self._cursor() as cur:
            await cur.execute(
                query, {"status": _status(status), "reservation_uid": reservation_uid}
            )
            if cur.rowcount != 1:
                raise RepositoryError(f"Failed to update reservation {reservation_uid}")


def _status(status: ReservationStatus | None) -> str | None:
    return status.value if status is not None else None


def _to_domain(row: dict[str, Any]) -> Reservation:
    return Reservation.model_validate(row)
