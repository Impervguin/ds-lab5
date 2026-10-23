from uuid import UUID

import pytest

from rating.domain import Rating

from .fakes import FakeRatingRepository, FakeUnitOfWork

USERNAME = "Bob"
RESERVATION_UID = UUID("4a1f2d0e-0d7f-4f6e-9f4a-3d7f0b2c1a55")


def make_rating(stars: int, username: str = USERNAME) -> Rating:
    return Rating(username=username, stars=stars)


@pytest.fixture
def repository() -> FakeRatingRepository:
    return FakeRatingRepository()


@pytest.fixture
def uow(repository: FakeRatingRepository) -> FakeUnitOfWork:
    return FakeUnitOfWork(repository)


@pytest.fixture
def uow_factory(uow: FakeUnitOfWork):
    return lambda: uow
