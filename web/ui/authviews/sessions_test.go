package authviews

import (
	"net/netip"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/infra/maxmind"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
)

func TestSessions(t *testing.T) {
	ip := netip.MustParseAddr("10.0.0.1")
	ua := "<agent>"
	entry := &SessionEntry{
		Session: &sudoapi.Session{ID: "sid123", UserID: 5, ExpiresAt: time.Now().Add(time.Hour)},
		User:    &kilonova.UserBrief{ID: 5, Name: "eve"},
		Devices: []*SessionDeviceEntry{{
			SessionDevice: &sudoapi.SessionDevice{IPAddr: &ip, UserAgent: &ua},
			Location:      &maxmind.Data{City: "Cluj", Subdivisions: []string{"CJ"}, Country: "RO"},
		}},
	}
	t.Run("admin sees id, owner and devices", func(t *testing.T) {
		got := uitest.Render(t, Sessions(ctx, SessionsParams{IsAdmin: true, Sessions: []*SessionEntry{entry}}))
		uitest.Contains(t, got, "<code>sid123</code>", `<a href="/profile/eve">eve (#5)</a>`, "10.0.0.1 (Cluj, CJ,  RO)", "<code>&lt;agent&gt;</code>", ": false")
	})
	t.Run("non-admin sees only timestamps", func(t *testing.T) {
		got := uitest.Render(t, Sessions(ctx, SessionsParams{Sessions: []*SessionEntry{entry}}))
		uitest.Contains(t, got, "server-timestamp")
		uitest.NotContains(t, got, "sid123", "/profile/eve", "10.0.0.1")
	})
	t.Run("device without location or agent", func(t *testing.T) {
		e := &SessionEntry{Session: &sudoapi.Session{}, Devices: []*SessionDeviceEntry{{SessionDevice: &sudoapi.SessionDevice{IPAddr: &ip}, Location: &maxmind.Data{}}}}
		got := uitest.Render(t, Sessions(ctx, SessionsParams{IsAdmin: true, Sessions: []*SessionEntry{e}}))
		uitest.Contains(t, got, "<li>10.0.0.1</li>")
	})
}
