package proxy

const (
	NoKey      = "no_key"
	InvalidKey = "invalid_key"
)

type RequestError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}
