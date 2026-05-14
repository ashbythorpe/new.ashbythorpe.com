package error

type AppError struct {
    StatusCode int
    PublicMsg  string
    Internal   error
}

type APIError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    TraceID string `json:"trace_id,omitempty"`
}
