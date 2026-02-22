package kilonova

import (
	"log/slog"
	"net/url"
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

var (
	debug           bool
	defaultLanguage string
	hostPrefix      string
	hostURL         *url.URL
)

func DebugMode() bool {
	return debug
}

func DefaultLanguage() string {
	if defaultLanguage == "" {
		slog.Warn("No default language set, defaulting to English")
		defaultLanguage = "en"
	}
	return defaultLanguage
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

func SetDefaultLanguage(lng string) {
	defaultLanguage = lng
	switch lng {
	case "en", "ro":
	case "":
		slog.Warn("No default language set, defaulting to English")
		defaultLanguage = "en"
	default:
		panic("invalid language (only 'en' and 'ro' allowed): " + lng)
	}
}

func SetHostPrefix(prefix string) {
	hostPrefix = prefix
	var err error
	hostURL, err = url.Parse(prefix)
	if err != nil {
		panic("invalid host prefix: " + err.Error())
	}
}
