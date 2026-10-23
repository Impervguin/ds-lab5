from __future__ import annotations

import logging
import sys
from dataclasses import dataclass

LOG_FORMAT = (
    "%(asctime)s | %(levelname)-8s | %(name)s | "
    "%(filename)s:%(lineno)d | %(message)s"
)
DATE_FORMAT = "%Y-%m-%d %H:%M:%S"

UVICORN_LOGGERS = ("uvicorn", "uvicorn.error", "uvicorn.access")


@dataclass(frozen=True, slots=True)
class LoggerSettings:
    level: str = "INFO"


_settings: LoggerSettings | None = None


def configure_logging(settings: LoggerSettings | None = None) -> LoggerSettings:
    global _settings
    if _settings is not None:
        return _settings

    settings = settings or LoggerSettings()
    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(logging.Formatter(LOG_FORMAT, DATE_FORMAT))

    root = logging.getLogger()
    root.handlers.clear()
    root.addHandler(handler)
    root.setLevel(settings.level.upper())

    for name in UVICORN_LOGGERS:
        uvicorn_logger = logging.getLogger(name)
        uvicorn_logger.handlers.clear()
        uvicorn_logger.propagate = True

    _settings = settings
    return settings


def get_logger(name: str) -> logging.Logger:
    return logging.getLogger(name)


def settings() -> LoggerSettings:
    return _settings or LoggerSettings()
