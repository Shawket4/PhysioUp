package Models

type NotificationRequest struct {
	Tokens []string `json:"tokens"` // Multiple device tokens
	Title  string   `json:"title"`  // Notification title
	Body   string   `json:"body"`   // Notification body
}

type ResponseMessage struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}