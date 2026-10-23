from fastapi import FastAPI, Request, status
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse

from .common import ErrorDescription, ValidationErrorResponse


def install_exception_handlers(app: FastAPI) -> None:
    app.add_exception_handler(RequestValidationError, _invalid_request)


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
