from collections.abc import AsyncIterator
from uuid import uuid4

import httpx
import pytest
from dependency_injector import providers

from reservation.di import Container
from reservation.domain import ReservationStatus
from reservation.main import create_app

from .conftest import BOOK_UID, LIBRARY_UID, USERNAME, make_reservation
from .fakes import FakeUnitOfWork

HEADERS = {"X-User-Name": USERNAME}
CREATE_PAYLOAD = {
    "bookUid": str(BOOK_UID),
    "libraryUid": str(LIBRARY_UID),
    "tillDate": "2021-10-11",
    "conditionAtRent": "GOOD",
}


@pytest.fixture
async def client(uow: FakeUnitOfWork) -> AsyncIterator[httpx.AsyncClient]:
    container = Container()
    container.uow.override(providers.Object(uow))

    app = create_app(container)
    transport = httpx.ASGITransport(app=app)
    async with httpx.AsyncClient(transport=transport, base_url="http://reservation") as client:
        yield client


async def test_health(client):
    assert (await client.get("/manage/health")).status_code == 200


async def test_list_reservations(client, repository):
    reservation = make_reservation()
    repository.reservations = [reservation, make_reservation(username="Alice")]

    response = await client.get("/api/v1/reservations", headers=HEADERS)

    assert response.status_code == 200
    assert response.json() == [
        {
            "reservationUid": str(reservation.reservation_uid),
            "bookUid": str(BOOK_UID),
            "libraryUid": str(LIBRARY_UID),
            "status": "RENTED",
            "startDate": "2021-10-01",
            "tillDate": "2021-10-11",
            "conditionAtRent": "EXCELLENT",
        }
    ]


async def test_count_reservations(client, repository):
    repository.reservations = [
        make_reservation(),
        make_reservation(status=ReservationStatus.EXPIRED),
    ]

    response = await client.get(
        "/api/v1/reservations/count", params={"status": "RENTED"}, headers=HEADERS
    )

    assert response.status_code == 200
    assert response.json() == {"count": 1}


async def test_create_reservation(client, repository):
    response = await client.post("/api/v1/reservations", json=CREATE_PAYLOAD, headers=HEADERS)

    assert response.status_code == 200
    assert response.json()["status"] == "RENTED"
    assert response.json()["conditionAtRent"] == "GOOD"
    assert repository.reservations[0].username == USERNAME


@pytest.mark.parametrize(
    ("headers", "payload", "field"),
    [
        ({}, CREATE_PAYLOAD, "X-User-Name"),
        (HEADERS, CREATE_PAYLOAD | {"conditionAtRent": "PERFECT"}, "conditionAtRent"),
    ],
)
async def test_rejects_an_invalid_request(client, repository, headers, payload, field):
    response = await client.post("/api/v1/reservations", json=payload, headers=headers)

    assert response.status_code == 400
    assert response.json()["errors"][0]["field"] == field
    assert repository.reservations == []


async def test_return_reservation(client, repository):
    reservation = make_reservation()
    repository.reservations = [reservation]

    response = await client.post(
        f"/api/v1/reservations/{reservation.reservation_uid}/return",
        json={"date": "2021-10-15"},
        headers=HEADERS,
    )

    assert response.status_code == 200
    assert response.json()["status"] == "EXPIRED"
    assert response.json()["conditionAtRent"] == "EXCELLENT"


async def test_return_conflicts_on_a_closed_reservation(client, repository):
    reservation = make_reservation(status=ReservationStatus.RETURNED)
    repository.reservations = [reservation]

    response = await client.post(
        f"/api/v1/reservations/{reservation.reservation_uid}/return",
        json={"date": "2021-10-15"},
        headers=HEADERS,
    )

    assert response.status_code == 409
    assert response.json() == {"message": "Reservation is already closed"}


async def test_unknown_reservation_is_not_found(client):
    response = await client.get(f"/api/v1/reservations/{uuid4()}", headers=HEADERS)

    assert response.status_code == 404
    assert response.json() == {"message": "Reservation not found"}
