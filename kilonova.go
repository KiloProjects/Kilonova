package kilonova

import (
	"time"

	"github.com/shopspring/decimal"
)

const Version = "v0.25.2"

type AuditLog struct {
	ID        int        `json:"id"`
	LogTime   time.Time  `json:"log_time"`
	SystemLog bool       `json:"system_log"`
	Message   string     `json:"message"`
	Author    *UserBrief `json:"author"`
}

func init() {
	// For returning submission data for fractional scores
	// We do not offer enough precision for this to be a problem
	decimal.MarshalJSONWithoutQuotes = true
}
