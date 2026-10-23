from fastapi import APIRouter, status

health_router = APIRouter(tags=["health"])


@health_router.get("/manage/health", status_code=status.HTTP_200_OK)
async def health() -> None:
    return None
