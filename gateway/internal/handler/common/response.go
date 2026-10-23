package common

import (
	"encoding/json"
	"net/http"

	"github.com/Impervguin/ds-lab2/gateway/internal/logger"
)

type ErrorResponse struct {
	Message string `json:"message"`
}

type ErrorDescription struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

type ValidationErrorResponse struct {
	Message string             `json:"message"`
	Errors  []ErrorDescription `json:"errors,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logger.Named("handler.common").Error("write response body", "error", err)
	}
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{Message: message})
}

func WriteValidationError(w http.ResponseWriter, message string, errs []ErrorDescription) {
	WriteJSON(w, http.StatusBadRequest, ValidationErrorResponse{Message: message, Errors: errs})
}
