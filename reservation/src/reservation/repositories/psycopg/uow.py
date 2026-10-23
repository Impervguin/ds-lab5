from contextlib import AbstractAsyncContextManager
from functools import partial
from types import TracebackType
from typing import Self

from psycopg import AsyncConnection
from psycopg_pool import AsyncConnectionPool

from ..protocols import ReservationRepository, UnitOfWorkFactory
from .reservation import PsycopgReservationRepository


class PsycopgUnitOfWork:
    def __init__(self, pool: AsyncConnectionPool) -> None:
        self._pool = pool
        self._connection: AsyncConnection | None = None
        self._context: AbstractAsyncContextManager[AsyncConnection] | None = None
        self._finalized = False

    async def __aenter__(self) -> Self:
        self._context = self._pool.connection()
        connection = await self._context.__aenter__()
        if connection.autocommit:
            await connection.set_autocommit(False)
        self._connection = connection
        self._finalized = False
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc: BaseException | None,
        tb: TracebackType | None,
    ) -> None:
        if self._connection is None or self._context is None:
            return
        try:
            if not self._finalized:
                if exc_type is None:
                    await self._connection.commit()
                else:
                    await self._connection.rollback()
        finally:
            await self._context.__aexit__(exc_type, exc, tb)
            self._connection = None
            self._context = None
            self._finalized = True

    async def rollback(self) -> None:
        if self._connection is None:
            raise RuntimeError("Unit of work is not entered")
        await self._connection.rollback()
        self._finalized = True

    @property
    def reservations(self) -> ReservationRepository:
        return PsycopgReservationRepository(self._active_connection)

    @property
    def _active_connection(self) -> AsyncConnection:
        if self._finalized:
            raise RuntimeError("Unit of work is finalized")
        if self._connection is None:
            raise RuntimeError("Unit of work is not entered")
        return self._connection


def make_uow_factory(pool: AsyncConnectionPool) -> UnitOfWorkFactory:
    return partial(PsycopgUnitOfWork, pool)
