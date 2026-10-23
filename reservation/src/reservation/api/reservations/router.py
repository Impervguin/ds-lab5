from typing import Annotated
from uuid import UUID

from dependency_injector.wiring import Provide, inject
from fastapi import APIRouter, Depends, Header, Query

from reservation.di import Container
from reservation.domain import ReservationStatus
from reservation.usecases import (
    CountReservationsUseCase,
    CreateReservationUseCase,
    GetReservationUseCase,
    ListReservationsUseCase,
    ReturnReservationUseCase,
)

from .dto import (
    CreateReservationRequest,
    ReservationCountResponse,
    ReservationResponse,
    ReturnReservationRequest,
)

reservations_router = APIRouter(prefix="/api/v1/reservations", tags=["reservations"])

Username = Annotated[str, Header(alias="X-User-Name", min_length=1, max_length=80)]
StatusFilter = Annotated[ReservationStatus | None, Query()]


@reservations_router.get("", response_model=list[ReservationResponse])
@inject
async def list_reservations(
    username: Username,
    status: StatusFilter = None,
    use_case: ListReservationsUseCase = Depends(Provide[Container.list_reservations_use_case]),
) -> list[ReservationResponse]:
    reservations = await use_case.execute(username, status)
    return [ReservationResponse.of(reservation) for reservation in reservations]


@reservations_router.get("/count", response_model=ReservationCountResponse)
@inject
async def count_reservations(
    username: Username,
    status: StatusFilter = None,
    use_case: CountReservationsUseCase = Depends(Provide[Container.count_reservations_use_case]),
) -> ReservationCountResponse:
    return ReservationCountResponse(count=await use_case.execute(username, status))


@reservations_router.get("/{reservation_uid}", response_model=ReservationResponse)
@inject
async def get_reservation(
    reservation_uid: UUID,
    username: Username,
    use_case: GetReservationUseCase = Depends(Provide[Container.get_reservation_use_case]),
) -> ReservationResponse:
    return ReservationResponse.of(await use_case.execute(username, reservation_uid))


@reservations_router.post("", response_model=ReservationResponse)
@inject
async def create_reservation(
    request: CreateReservationRequest,
    username: Username,
    use_case: CreateReservationUseCase = Depends(Provide[Container.create_reservation_use_case]),
) -> ReservationResponse:
    reservation = await use_case.execute(
        username=username,
        book_uid=request.book_uid,
        library_uid=request.library_uid,
        till_date=request.till_date,
        condition_at_rent=request.condition_at_rent,
    )
    return ReservationResponse.of(reservation)


@reservations_router.post("/{reservation_uid}/return", response_model=ReservationResponse)
@inject
async def return_reservation(
    reservation_uid: UUID,
    request: ReturnReservationRequest,
    username: Username,
    use_case: ReturnReservationUseCase = Depends(Provide[Container.return_reservation_use_case]),
) -> ReservationResponse:
    reservation = await use_case.execute(username, reservation_uid, request.return_date)
    return ReservationResponse.of(reservation)
