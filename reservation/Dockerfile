FROM python:3.14-slim AS build
WORKDIR /app

RUN pip install --no-cache-dir poetry==2.4.3 \
    && poetry config virtualenvs.in-project true

COPY pyproject.toml poetry.lock* README.md ./
RUN poetry install --no-root --only main --no-interaction --no-ansi

COPY src ./src
RUN poetry install --only-root --no-interaction --no-ansi

FROM python:3.14-slim
WORKDIR /app

COPY --from=build /app/.venv ./.venv
COPY --from=build /app/src ./src

ENV PATH="/app/.venv/bin:$PATH"

EXPOSE 8070
CMD ["uvicorn", "reservation.main:app", "--host", "0.0.0.0", "--port", "8070"]
