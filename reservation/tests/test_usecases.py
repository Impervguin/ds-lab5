from datetime import date

import pytest

from reservation.domain import (
    BookCondition,
    ReservationAlreadyClosed,
    ReservationNotFound,
    ReservationStatus,
)
from reservation.usecases import (
    CountReservationsUseCase,
    CreateReservationUseCase,
    GetReservationUseCase,
    ListReservationsUseCase,
    ReturnReservationUseCase,
)

from .conftest import BOOK_UID, LIBRARY_UID, TILL_DATE, USERNAME, make_reservation


async def test_list_returns_only_reservations_of_the_user(repository, uow_factory):
    mine = make_reservation()
    repository.reservations = [mine, make_reservation(username="Alice")]

    assert await ListReservationsUseCase(uow_factory).execute(USERNAME) == [mine]


async def test_count_filters_by_status(repository, uow_factory):
    repository.reservations = [
        make_reservation(),
        make_reservation(status=ReservationStatus.RETURNED),
    ]

    count = await CountReservationsUseCase(uow_factory).execute(USERNAME, ReservationStatus.RENTED)

    assert count == 1


async def test_get_rejects_a_reservation_of_another_user(repository, uow_factory):
    reservation = make_reservation(username="Alice")
    repository.reservations = [reservation]

    with pytest.raises(ReservationNotFound):
        await GetReservationUseCase(uow_factory).execute(USERNAME, reservation.reservation_uid)


async def test_create_opens_a_rented_reservation(repository, uow, uow_factory):
    created = await CreateReservationUseCase(uow_factory).execute(
        username=USERNAME,
        book_uid=BOOK_UID,
        library_uid=LIBRARY_UID,
        till_date=TILL_DATE,
        condition_at_rent=BookCondition.GOOD,
    )

    assert created.status is ReservationStatus.RENTED
    assert created.start_date == date.today()
    assert created.condition_at_rent is BookCondition.GOOD
    assert repository.reservations == [created]
    assert uow.committed


@pytest.mark.parametrize(
    ("return_date", "expected"),
    [
        (TILL_DATE, ReservationStatus.RETURNED),
        (date(2021, 10, 15), ReservationStatus.EXPIRED),
    ],
)
async def test_return_decides_the_status(repository, uow_factory, return_date, expected):
    reservation = make_reservation()
    repository.reservations = [reservation]

    closed = await ReturnReservationUseCase(uow_factory).execute(
        USERNAME, reservation.reservation_uid, return_date
    )

    assert closed.status is expected
    assert repository.reservations[0].status is expected


async def test_return_rejects_an_already_closed_reservation(repository, uow, uow_factory):
    reservation = make_reservation(status=ReservationStatus.RETURNED)
    repository.reservations = [reservation]

    with pytest.raises(ReservationAlreadyClosed):
        await ReturnReservationUseCase(uow_factory).execute(
            USERNAME, reservation.reservation_uid, TILL_DATE
        )

    assert uow.rolled_back
    assert repository.reservations == [reservation]


async def test_return_rejects_a_reservation_of_another_user(repository, uow_factory):
    reservation = make_reservation(username="Alice")
    repository.reservations = [reservation]

    with pytest.raises(ReservationNotFound):
        await ReturnReservationUseCase(uow_factory).execute(
            USERNAME, reservation.reservation_uid, TILL_DATE
        )
