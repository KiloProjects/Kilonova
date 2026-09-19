package kilonova

import (
	"net/url"
	"time"

	"github.com/shopspring/decimal"
)

const Version = "v26.09"

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

var (
	debug      bool
	hostPrefix string
	hostURL    *url.URL
)

func DebugMode() bool {
	return debug
}

func HostPrefix() string {
	return hostPrefix
}

func HostURL() *url.URL {
	return hostURL
}

func SetDebugMode(dbg bool) {
	debug = dbg
}

func SetHostPrefix(prefix string) {
	hostPrefix = prefix
	var err error
	hostURL, err = url.Parse(prefix)
	if err != nil {
		panic("invalid host prefix: " + err.Error())
	}
}
