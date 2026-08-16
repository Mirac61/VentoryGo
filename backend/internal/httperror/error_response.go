package httperror

type ErrorResponse struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields"`
	RequestID string            `json:"requestId,omitempty"`
}
