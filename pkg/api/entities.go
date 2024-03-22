package api

// ErrorResponse represents the structure of the error response
type ErrorResponse struct {
	Fields    []ErrorResponseField `json:"fields,omitempty"`
	Error     string               `json:"error,omitempty"`
	ErrorType string               `json:"errorType,omitempty"`
}

// Field represents an error field
type ErrorResponseField struct {
	Field string `json:"field,omitempty"`
	Error string `json:"error,omitempty"`
}
