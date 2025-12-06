package utils

import (
	"encoding/json"
	"net/http"
)

// JSONResponse writes a JSON response with the given status code
func JSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// ErrorResponse writes a JSON error response
func ErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	JSONResponse(w, map[string]string{
		"error": message,
	}, statusCode)
}

// SuccessResponse writes a JSON success response
func SuccessResponse(w http.ResponseWriter, message string, data interface{}) {
	JSONResponse(w, map[string]interface{}{
		"message": message,
		"data":    data,
	}, http.StatusOK)
}
