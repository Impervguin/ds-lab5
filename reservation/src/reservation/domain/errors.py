from uuid import UUID


class DomainError(Exception):
    pass


class ReservationNotFound(DomainError):
    def __init__(self, reservation_uid: UUID) -> None:
        super().__init__(f"Reservation {reservation_uid} not found")
        self.reservation_uid = reservation_uid


class ReservationAlreadyClosed(DomainError):
    def __init__(self, reservation_uid: UUID) -> None:
        super().__init__(f"Reservation {reservation_uid} is already closed")
        self.reservation_uid = reservation_uid
