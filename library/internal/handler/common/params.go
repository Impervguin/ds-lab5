package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var ErrMissingParam = errors.New("parameter is required")

func IntQuery(r *http.Request, name string, fallback int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%q is not a number", raw)
	}
	return value, nil
}

func BoolQuery(r *http.Request, name string, fallback bool) (bool, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%q is not a boolean", raw)
	}
	return value, nil
}

func RequiredQuery(r *http.Request, name string) (string, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return "", ErrMissingParam
	}
	return value, nil
}

func UUIDParam(r *http.Request, name string) (uuid.UUID, error) {
	raw := chi.URLParam(r, name)
	value, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%q is not a valid UUID", raw)
	}
	return value, nil
}

func DecodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is empty")
		}
		return fmt.Errorf("malformed request body: %w", err)
	}
	return nil
}
