from typing import Any

from psycopg import AsyncConnection, AsyncCursor
from psycopg.rows import dict_row


class PsycopgRepository:
    def __init__(self, connection: AsyncConnection) -> None:
        self._conn = connection

    def _cursor(self) -> AsyncCursor[dict[str, Any]]:
        return self._conn.cursor(row_factory=dict_row)
