package utils

import "strings"

// status code
const (
	StatusCodeUnauthorized       = "401" // HTTP 401 Unauthorized
	StatusCodePageNotFound       = "404" // HTTP 404 Not Found
	StatusCodeTooManyRequests    = "429" // HTTP 429 Too Many Requests
	StatusCodeServiceUnavailable = "503" // HTTP 503 Service Unavailable
)

const (
	BaseWriteTimeoutSeconds = 120
)

// IsAbortRetryError checks if the error indicates we should abort retrying
func IsAbortRetryError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()
	return strings.Contains(errorStr, StatusCodeUnauthorized) ||
		strings.Contains(errorStr, StatusCodePageNotFound) ||
		strings.Contains(errorStr, StatusCodeTooManyRequests) ||
		strings.Contains(errorStr, StatusCodeServiceUnavailable)
}

func IsUnauthorizedError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), StatusCodeUnauthorized)
}

func IsPageNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), StatusCodePageNotFound)
}

func IsTooManyRequestsError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), StatusCodeTooManyRequests)
}

func IsServiceUnavailableError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), StatusCodeServiceUnavailable)
}
