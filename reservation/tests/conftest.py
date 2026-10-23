from datetime import date
from uuid import UUID, uuid4

import pytest

from reservation.domain import BookCondition, Reservation, ReservationStatus

from .fakes import FakeReservationRepository, FakeUnitOfWork

USERNAME = "Bob"
BOOK_UID = UUID("f7cdc58f-2caf-4b15-9727-f89dcc629b27")
LIBRARY_UID = UUID("83575e12-7ce0-48ee-9931-51919ff3c9ee")
TILL_DATE = date(2021, 10, 11)


def make_reservation(
    username: str = USERNAME,
    status: ReservationStatus = ReservationStatus.RENTED,
    reservation_uid: UUID | None = None,
    till_date: date = TILL_DATE,
    condition_at_rent: BookCondition = BookCondition.EXCELLENT,
) -> Reservation:
    return Reservation(
        reservation_uid=reservation_uid or uuid4(),
        username=username,
        book_uid=BOOK_UID,
        library_uid=LIBRARY_UID,
        status=status,
        start_date=date(2021, 10, 1),
        till_date=till_date,
        condition_at_rent=condition_at_rent,
    )


@pytest.fixture
def repository() -> FakeReservationRepository:
    return FakeReservationRepository()


@pytest.fixture
def uow(repository: FakeReservationRepository) -> FakeUnitOfWork:
    return FakeUnitOfWork(repository)


@pytest.fixture
def uow_factory(uow: FakeUnitOfWork):
    return lambda: uow
