from typing import Final

from psycopg import sql

from rating.domain import Rating

from ..errors import RepositoryError
from ..protocols import RatingRepository
from .base import PsycopgRepository

TABLE: Final = sql.Identifier("rating")

GET_OR_CREATE: Final = sql.SQL(
    "INSERT INTO {table} (username, stars) VALUES (%(username)s, %(stars)s)"
    " ON CONFLICT (username) DO UPDATE SET stars = {table}.stars"
    " RETURNING username, stars"
).format(table=TABLE)

UPDATE_STARS: Final = sql.SQL(
    "UPDATE {table} SET stars = %(stars)s WHERE username = %(username)s"
).format(table=TABLE)


class PsycopgRatingRepository(PsycopgRepository, RatingRepository):
    async def get_or_create(self, initial: Rating) -> Rating:
        async with self._cursor() as cur:
            await cur.execute(GET_OR_CREATE, initial.model_dump())
            row = await cur.fetchone()
            if row is None:
                raise RepositoryError(f"Failed to read rating of {initial.username}")
            return Rating.model_validate(row)

    async def update_stars(self, username: str, stars: int) -> None:
        async with self._cursor() as cur:
            await cur.execute(UPDATE_STARS, {"username": username, "stars": stars})
            if cur.rowcount != 1:
                raise RepositoryError(f"Failed to update rating of {username}")
