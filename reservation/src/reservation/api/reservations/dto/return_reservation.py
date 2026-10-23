from datetime import date

from pydantic import Field

from reservation.api.common import CamelModel


class ReturnReservationRequest(CamelModel):
    return_date: date = Field(alias="date")
