package browserlist

import (
	"context"
	"log/slog"
	"slices"
	"strings"

	"github.com/gosimple/slug"
	"github.com/mileusna/useragent"
)

type baselineEntry struct {
	Year        any
	ReleaseDate string
	Supports    string
}

type Baseliner struct {
	data map[string]map[string]*baselineEntry
}

func (b Baseliner) SupportsWidelyAvailable(ctx context.Context, userAgent string) (bool, bool) {
	if b.data == nil {
		return false, false
	}

	ua := useragent.Parse(userAgent)
	browserID, ok := GetBrowserID(ua)
	if !ok {
		return false, false
	}

	if b.data[browserID] == nil {
		slog.WarnContext(ctx, "valid id not found in baseline data", slog.String("browserID", browserID))
		return false, false
	}

	browserData := b.data[browserID]
	version := ua.VersionNoShort()
	for len(version) > 0 {
		if browserData[version] != nil {
			return browserData[version].Supports == "newly" || browserData[version].Supports == "widely", true
		}
		if i := strings.LastIndex(version, "."); i >= 0 {
			version = version[:i]
		} else {
			break
		}
	}

	return false, true
}

var androidBrowsers = []string{"chrome", "firefox", "opera",
	"qq", "samsunginternet", "uc", "webview", "ya", "facebook", "instagram",
}

// GetBrowserID returns a browser ID for the given user agent.
// It uses the convention from here: https://github.com/web-platform-dx/baseline-browser-mapping#downstream-browsers
// The second parameter is false if it could not be determined as one of the downstream browsers.
func GetBrowserID(ua useragent.UserAgent) (string, bool) {

	var id string
	ok := true
	switch ua.Name {
	case useragent.Chrome, useragent.Edge, useragent.Firefox, useragent.Opera, useragent.Safari:
		id = strings.ToLower(ua.Name)
	case useragent.FacebookApp:
		id = "facebook"
	case useragent.InstagramApp:
		id = "instagram"
	case useragent.SamsungBrowser:
		id = "samsunginternet"
	default:
		id = slug.Make(ua.Name)
		ok = false
	}
	if ok && ua.IsAndroid() && slices.Contains(androidBrowsers, id) {
		id += "_android"
	}
	if ok && ua.IsIOS() && id == "safari" {
		id += "_ios"
	}

	return id, ok
}
