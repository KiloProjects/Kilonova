package kilonova

import (
	"embed"
	"time"
)

const Version = "v0.17.0"

//go:embed docs
var Docs embed.FS

type AuditLog struct {
	ID        int        `json:"id"`
	LogTime   time.Time  `json:"log_time"`
	SystemLog bool       `json:"system_log"`
	Message   string     `json:"message"`
	Author    *UserBrief `json:"author"`
}
