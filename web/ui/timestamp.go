package ui

import (
	"strconv"
	"time"

	. "maragu.dev/gomponents"
)

// ServerTimestamp renders t as a <server-timestamp>, which the frontend formats in the
// viewer's timezone.
func ServerTimestamp(t time.Time) Node {
	return El("server-timestamp", Attr("timestamp", strconv.FormatInt(t.UnixMilli(), 10)))
}
