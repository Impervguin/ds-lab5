from fastapi import FastAPI, Request, status
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse

from reservation.domain import ReservationAlreadyClosed, ReservationNotFound

from .common import ErrorDescription, ErrorResponse, ValidationErrorResponse


def install_exception_handlers(app: FastAPI) -> None:
    app.add_exception_handler(ReservationNotFound, _not_found)
    app.add_exception_handler(ReservationAlreadyClosed, _already_closed)
    app.add_exception_handler(RequestValidationError, _invalid_request)


async def _not_found(_request: Request, _exc: Exception) -> JSONResponse:
    return _error(status.HTTP_404_NOT_FOUND, "Reservation not found")


async def _already_closed(_request: Request, _exc: Exception) -> JSONResponse:
    return _error(status.HTTP_409_CONFLICT, "Reservation is already closed")


async def _invalid_request(_request: Request, exc: Exception) -> JSONResponse:
    assert isinstance(exc, RequestValidationError)
    payload = ValidationErrorResponse(
        message="request validation failed",
        errors=[
            ErrorDescription(field=_field(error.get("loc", ())), error=error["msg"])
            for error in exc.errors()
        ],
    )
    return JSONResponse(
        status_code=status.HTTP_400_BAD_REQUEST, content=payload.model_dump(by_alias=True)
    )


def _field(location: tuple[int | str, ...]) -> str:
    return ".".join(str(part) for part in location[1:]) or str(location[0] if location else "")


def _error(code: int, message: str) -> JSONResponse:
    return JSONResponse(
        status_code=code, content=ErrorResponse(message=message).model_dump(by_alias=True)
    )
