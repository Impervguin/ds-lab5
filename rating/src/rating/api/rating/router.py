from typing import Annotated

from dependency_injector.wiring import Provide, inject
from fastapi import APIRouter, Depends, Header

from rating.di import Container
from rating.usecases import CloseReservationUseCase, GetRatingUseCase

from .dto import RatingChangeResponse, RatingResponse, ReservationClosedRequest

rating_router = APIRouter(prefix="/api/v1/rating", tags=["rating"])

Username = Annotated[str, Header(alias="X-User-Name", min_length=1, max_length=80)]


@rating_router.get("", response_model=RatingResponse)
@inject
async def get_rating(
    username: Username,
    use_case: GetRatingUseCase = Depends(Provide[Container.get_rating_use_case]),
) -> RatingResponse:
    return RatingResponse.of(await use_case.execute(username))


@rating_router.post("/reservation-closed", response_model=RatingChangeResponse)
@inject
async def reservation_closed(
    request: ReservationClosedRequest,
    username: Username,
    use_case: CloseReservationUseCase = Depends(Provide[Container.close_reservation_use_case]),
) -> RatingChangeResponse:
    change = await use_case.execute(
        username=username,
        reservation_uid=request.reservation_uid,
        status=request.reservation_status,
        condition_at_rent=request.condition_at_rent,
        condition_on_return=request.condition_on_return,
    )
    return RatingChangeResponse.of(change)
