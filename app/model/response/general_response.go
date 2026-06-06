package response

type GeneralResponse struct {
	IsSuccess bool   `json:"is_success"`
	Message   string `json:"message,omitempty"`
}

func (m *GeneralResponse) SetError(message string) {
	m.IsSuccess = false
	m.Message = message
}
