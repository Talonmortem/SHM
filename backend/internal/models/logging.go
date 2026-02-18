package models

import (
	"encoding/json"
	"time"
)

// RequestLog модель для сохранения
type RequestLog struct {
	Username  string          `json:"username"`
	Method    string          `json:"method"`
	Path      string          `json:"path"`
	Params    json.RawMessage `json:"params"`
	CreatedAt time.Time       `json:"created_at"`
}
