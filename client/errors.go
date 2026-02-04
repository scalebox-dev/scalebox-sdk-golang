package client

// Error types
var (
	ErrNotFound      = &APIError{StatusCode: 404, Message: "Resource not found"}
	ErrUnauthorized  = &APIError{StatusCode: 401, Message: "Unauthorized"}
	ErrForbidden     = &APIError{StatusCode: 403, Message: "Forbidden"}
	ErrBadRequest    = &APIError{StatusCode: 400, Message: "Bad request"}
	ErrInternalError = &APIError{StatusCode: 500, Message: "Internal server error"}
)

// IsNotFound checks if the error is a 404 Not Found error
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 404
	}
	return false
}

// IsUnauthorized checks if the error is a 401 Unauthorized error
func IsUnauthorized(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 401
	}
	return false
}

// IsForbidden checks if the error is a 403 Forbidden error
func IsForbidden(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 403
	}
	return false
}

// StatusCode returns the HTTP status code from an error if it's an APIError
func StatusCode(err error) int {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode
	}
	return 0
}
