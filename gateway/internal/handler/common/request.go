package common

import (
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func NewValidator() *validator.Validate {
	validate := validator.New()
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return validate
}

func Username(w http.ResponseWriter, r *http.Request) (string, bool) {
	username := r.Header.Get(UserNameHeader)
	if username == "" {
		BadParam(w, UserNameHeader, ErrMissingHeader)
		return "", false
	}
	return username, true
}

func PathUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	value, err := UUIDParam(r, name)
	if err != nil {
		BadParam(w, name, err)
		return uuid.Nil, false
	}
	return value, true
}

func BadParam(w http.ResponseWriter, name string, err error) {
	WriteValidationError(w, "request validation failed", []ErrorDescription{
		{Field: name, Error: err.Error()},
	})
}

func BadRequest(w http.ResponseWriter, err error) {
	var invalid validator.ValidationErrors
	if !errors.As(err, &invalid) {
		WriteValidationError(w, err.Error(), nil)
		return
	}

	described := make([]ErrorDescription, 0, len(invalid))
	for _, fieldError := range invalid {
		described = append(described, ErrorDescription{
			Field: fieldError.Field(),
			Error: "failed on the " + fieldError.Tag() + " rule",
		})
	}
	WriteValidationError(w, "request validation failed", described)
}
