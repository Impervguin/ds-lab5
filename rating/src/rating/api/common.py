from pydantic import BaseModel, ConfigDict, Field
from pydantic.alias_generators import to_camel


class CamelModel(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True)


class ErrorDescription(CamelModel):
    field: str
    error: str


class ValidationErrorResponse(CamelModel):
    message: str
    errors: list[ErrorDescription] = Field(default_factory=list)
