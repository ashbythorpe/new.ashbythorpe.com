package utils

type AppError struct {
	Status    int    `json:"-"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}
