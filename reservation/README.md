# Reservation Service

Жизненный цикл аренды книги: кто, какую книгу, в какой библиотеке и до какой даты взял.
Сервис владеет статусами резерва (`RENTED`, `RETURNED`, `EXPIRED`), решает, просрочен ли
возврат, и хранит состояние экземпляра на момент выдачи (`condition_at_rent`).

Подробности — в [design.md](../design.md).

## Запуск

```bash
poetry install
poetry run uvicorn reservation.main:app --port 8070
```

## Тесты

```bash
poetry run pytest
```
