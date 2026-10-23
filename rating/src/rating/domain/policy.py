from typing import Final

from .rating import BookCondition, ReservationStatus

LATE_RETURN_PENALTY: Final = -10
CONDITION_CHANGE_PENALTY: Final = -10
GOOD_RETURN_BONUS: Final = 1


def closing_delta(
    status: ReservationStatus,
    condition_at_rent: BookCondition,
    condition_on_return: BookCondition,
) -> int:
    penalty = 0
    if status is ReservationStatus.EXPIRED:
        penalty += LATE_RETURN_PENALTY
    if condition_on_return is not condition_at_rent:
        penalty += CONDITION_CHANGE_PENALTY
    return penalty or GOOD_RETURN_BONUS
