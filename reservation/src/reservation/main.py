import asyncio
import os
from contextlib import asynccontextmanager

from fastapi import FastAPI

from reservation.api import health_router, reservations_router
from reservation.api.errors import install_exception_handlers
from reservation.di import Container
from reservation.logging import LoggerSettings, configure_logging, get_logger
from reservation.migrations import apply_migrations

logger = get_logger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    container: Container = app.state.container
    settings = container.config()

    await asyncio.to_thread(apply_migrations, settings.database_url, settings.migration_lock_key)

    pool = container.pool()
    await pool.open(wait=True)
    yield
    await pool.close()


def create_app(container: Container | None = None) -> FastAPI:
    configure_logging(LoggerSettings(level=os.getenv("LOG_LEVEL", "INFO")))
    logger.info("creating application")

    container = container or Container()
    container.wire(modules=["reservation.api.reservations.router"])

    app = FastAPI(title="Reservation Service", lifespan=lifespan)
    app.state.container = container
    app.include_router(health_router)
    app.include_router(reservations_router)
    install_exception_handlers(app)
    return app


app = create_app()
