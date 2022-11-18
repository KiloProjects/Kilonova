package kilonova

import (
	"embed"
	"time"
)

const Version = "v0.10.3"

//go:embed docs
var Docs embed.FS

const IndexDescription = `
Disclaimer: Această platformă este în continuă dezvoltare. Versiunea finală s-ar putea să arate semnificativ diferit față de ce vedeți acum.

[Probleme OJI 2002-2022 XI-XII pe ani](https://kilonova.ro/docs/OJI)
[Probleme ONI 2001-2022 XI-XII pe ani](https://kilonova.ro/docs/ONI)
[Probleme Baraje/Loturi Seniori 2021-2022](https://kilonova.ro/docs/BARAJ)
`

type AuditLog struct {
	ID        int        `json:"id"`
	LogTime   time.Time  `json:"log_time"`
	SystemLog bool       `json:"system_log"`
	Message   string     `json:"message"`
	Author    *UserBrief `json:"author"`
}
