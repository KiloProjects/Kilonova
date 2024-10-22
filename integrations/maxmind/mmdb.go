package maxmind

import (
	"log/slog"
	"net/netip"
	"time"

	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/oschwald/maxminddb-golang/v2"
)

var MaxMindPath = config.GenFlag("integrations.maxmind.db_path", "/usr/share/GeoIP/GeoLite2-City.mmdb", "Path to MaxMind GeoLite2-City database for IPs.")
var reader *maxminddb.Reader

type Data struct {
	City    string
	Country string

	Latitude  float64
	Longitude float64
}

// Note that the data *can* be nil if there is no data available (IP not found, DB not loaded, etc)
func IPData(ip netip.Addr) (*Data, error) {
	if reader == nil {
		return nil, nil
	}
	result := reader.Lookup(ip)
	var data Data
	if err := result.DecodePath(data.City, "city", "names", "en"); err != nil {
		data.City = ""
	}
	if err := result.DecodePath(data.Country, "country", "names", "en"); err != nil {
		data.Country = ""
	}
	if err := result.DecodePath(data.Latitude, "location", "latitude"); err != nil {
		data.Latitude = 0
	}
	if err := result.DecodePath(data.Longitude, "location", "longitude"); err != nil {
		data.Longitude = 0
	}
	return &data, nil
}

func Initialize() {
	var err error
	reader, err = maxminddb.Open(MaxMindPath.Value())
	if err != nil {
		slog.Info("Could not open MaxMind DB")
		slog.Debug("MaxMind DB error", slog.Any("err", err))
		return
	}

	if reader.Metadata.DatabaseType != "GeoLite2-City" {
		slog.Warn("Only GeoLite2-City format is supported for MaxMind DBs")
		reader = nil
		return
	}

	slog.Info("MaxMind DB loaded", slog.String("version", time.Unix(int64(reader.Metadata.BuildEpoch), 0).Format(time.DateOnly)))
}
