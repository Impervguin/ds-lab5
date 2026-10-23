from rating.domain import Rating
from rating.repositories import UnitOfWorkFactory


class GetRatingUseCase:
    def __init__(self, uow_factory: UnitOfWorkFactory) -> None:
        self._uow_factory = uow_factory

    async def execute(self, username: str) -> Rating:
        async with self._uow_factory() as uow:
            return await uow.ratings.get_or_create(Rating.initial(username))
