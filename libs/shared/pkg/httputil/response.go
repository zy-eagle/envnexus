package httputil

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// APIResponse is the standard API response envelope.
type APIResponse struct {
	RequestID string      `json:"request_id"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// APIError is the standard error structure within an API response.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RespondOK sends a success JSON response with the given data.
func RespondOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// RespondCreated sends a 201 Created JSON response with the given data.
func RespondCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// RespondErr sends an error JSON response.
func RespondErr(c *gin.Context, status int, code, message string) {
	c.JSON(status, APIResponse{
		Error:     &APIError{Code: code, Message: message},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
