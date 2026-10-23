from collections.abc import AsyncIterator

import httpx
import pytest
from dependency_injector import providers

from rating.di import Container
from rating.main import create_app

from .conftest import RESERVATION_UID, USERNAME, make_rating
from .fakes import FakeUnitOfWork

HEADERS = {"X-User-Name": USERNAME}
CLOSED_PAYLOAD = {
    "reservationUid": str(RESERVATION_UID),
    "reservationStatus": "EXPIRED",
    "conditionAtRent": "EXCELLENT",
    "conditionOnReturn": "BAD",
}


@pytest.fixture
async def client(uow: FakeUnitOfWork) -> AsyncIterator[httpx.AsyncClient]:
    container = Container()
    container.uow.override(providers.Object(uow))

    transport = httpx.ASGITransport(app=create_app(container))
    async with httpx.AsyncClient(transport=transport, base_url="http://rating") as client:
        yield client


async def test_health(client):
    assert (await client.get("/manage/health")).status_code == 200


async def test_get_rating(client, repository):
    repository.ratings[USERNAME] = make_rating(75)

    response = await client.get("/api/v1/rating", headers=HEADERS)

    assert response.status_code == 200
    assert response.json() == {"stars": 75, "maxBooks": 25}


async def test_get_rating_requires_a_username(client):
    assert (await client.get("/api/v1/rating")).status_code == 400


async def test_reservation_closed(client, repository):
    repository.ratings[USERNAME] = make_rating(75)

    response = await client.post(
        "/api/v1/rating/reservation-closed", json=CLOSED_PAYLOAD, headers=HEADERS
    )

    assert response.status_code == 200
    assert response.json() == {"delta": -20, "stars": 55}
    assert repository.ratings[USERNAME].stars == 55


async def test_reservation_closed_rejects_an_unknown_condition(client):
    response = await client.post(
        "/api/v1/rating/reservation-closed",
        json={**CLOSED_PAYLOAD, "conditionOnReturn": "BROKEN"},
        headers=HEADERS,
    )

    assert response.status_code == 400
    assert response.json()["errors"][0]["field"] == "conditionOnReturn"
