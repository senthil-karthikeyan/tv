// Package handler provides HTTP handler functions and shared JSON response helpers.
package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

// ---- response envelope types ----

type successEnvelope struct {
	Data any `json:"data"`
}

type collectionEnvelope struct {
	Data any      `json:"data"`
	Meta metaBody `json:"meta"`
}

type metaBody struct {
	Total int `json:"total"`
}

type errorEnvelope struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ---- public helpers ----

// WriteJSON writes a JSON-encoded body with the given HTTP status code.
// Encoding errors are logged but not surfaced to the client.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("response encode error: %v", err)
	}
}

// WriteData writes a successful response wrapping data in {"data": ...}.
func WriteData(w http.ResponseWriter, status int, data any) {
	WriteJSON(w, status, successEnvelope{Data: data})
}

// WriteCollection writes a successful list response with total count metadata.
func WriteCollection(w http.ResponseWriter, data any, total int) {
	WriteJSON(w, http.StatusOK, collectionEnvelope{
		Data: data,
		Meta: metaBody{Total: total},
	})
}

// WriteError writes a JSON error response with the given status, machine-readable code, and human-readable message.
// Internal implementation details must never be passed as the message argument.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, errorEnvelope{Error: errorDetail{Code: code, Message: message}})
}

// WriteNotFound writes a 404 JSON error.
func WriteNotFound(w http.ResponseWriter, resource string) {
	WriteError(w, http.StatusNotFound, "NOT_FOUND", resource+" not found")
}

// WriteMethodNotAllowed writes a 405 JSON error.
func WriteMethodNotAllowed(w http.ResponseWriter) {
	WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
}

// WriteInternalError writes a 500 JSON error without exposing internal details to the client.
func WriteInternalError(w http.ResponseWriter) {
	WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
}

// WriteBadRequest writes a 400 JSON error.
func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, "BAD_REQUEST", message)
}

// WriteValidationError writes a 422 JSON error for invalid input data.
func WriteValidationError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", message)
}

// WriteConflict writes a 409 JSON error.
func WriteConflict(w http.ResponseWriter, code, message string) {
	WriteError(w, http.StatusConflict, code, message)
}
