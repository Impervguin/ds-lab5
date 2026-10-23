from rating.domain import INITIAL_STARS, MAX_STARS, MIN_STARS, BookCondition, ReservationStatus
from rating.usecases import CloseReservationUseCase, GetRatingUseCase

from .conftest import RESERVATION_UID, USERNAME, make_rating

EXCELLENT = BookCondition.EXCELLENT
BAD = BookCondition.BAD


async def close(uow_factory, status=ReservationStatus.RETURNED, at_rent=EXCELLENT, on_return=EXCELLENT):
    return await CloseReservationUseCase(uow_factory).execute(
        username=USERNAME,
        reservation_uid=RESERVATION_UID,
        status=status,
        condition_at_rent=at_rent,
        condition_on_return=on_return,
    )


async def test_first_request_creates_the_initial_rating(uow_factory, repository):
    rating = await GetRatingUseCase(uow_factory).execute(USERNAME)

    assert rating.stars == INITIAL_STARS
    assert repository.ratings[USERNAME].stars == INITIAL_STARS


async def test_stars_are_turned_into_a_book_limit(uow_factory, repository):
    repository.ratings[USERNAME] = make_rating(75)

    assert (await GetRatingUseCase(uow_factory).execute(USERNAME)).max_books == 25


async def test_return_in_time_and_in_the_same_condition_is_a_bonus(uow_factory, repository):
    repository.ratings[USERNAME] = make_rating(70)

    change = await close(uow_factory)

    assert (change.delta, change.rating.stars) == (1, 71)
    assert repository.ratings[USERNAME].stars == 71


async def test_late_return_is_penalised(uow_factory, repository):
    repository.ratings[USERNAME] = make_rating(70)

    change = await close(uow_factory, status=ReservationStatus.EXPIRED)

    assert (change.delta, change.rating.stars) == (-10, 60)


async def test_changed_condition_is_penalised(uow_factory, repository):
    repository.ratings[USERNAME] = make_rating(70)

    change = await close(uow_factory, on_return=BAD)

    assert (change.delta, change.rating.stars) == (-10, 60)


async def test_penalties_add_up(uow_factory, repository):
    repository.ratings[USERNAME] = make_rating(75)

    change = await close(uow_factory, status=ReservationStatus.EXPIRED, on_return=BAD)

    assert (change.delta, change.rating.stars) == (-20, 55)


async def test_a_better_condition_on_return_is_still_a_violation(uow_factory, repository):
    repository.ratings[USERNAME] = make_rating(70)

    assert (await close(uow_factory, at_rent=BAD, on_return=EXCELLENT)).delta == -10


async def test_rating_does_not_grow_above_the_maximum(uow_factory, repository):
    repository.ratings[USERNAME] = make_rating(MAX_STARS)

    change = await close(uow_factory)

    assert (change.delta, change.rating.stars) == (0, MAX_STARS)


async def test_rating_does_not_fall_below_the_minimum(uow_factory, repository):
    repository.ratings[USERNAME] = make_rating(5)

    change = await close(uow_factory, status=ReservationStatus.EXPIRED, on_return=BAD)

    assert (change.delta, change.rating.stars) == (-4, MIN_STARS)


async def test_closing_a_reservation_of_an_unknown_user_starts_from_the_initial_rating(
    uow_factory, repository
):
    change = await close(uow_factory, status=ReservationStatus.EXPIRED)

    assert change.rating.stars == INITIAL_STARS - 10
    assert repository.ratings[USERNAME].stars == INITIAL_STARS - 10
