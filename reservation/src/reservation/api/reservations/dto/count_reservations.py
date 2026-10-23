from reservation.api.common import CamelModel


class ReservationCountResponse(CamelModel):
    count: int
