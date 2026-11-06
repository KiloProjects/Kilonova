package authviews

import (
	"context"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/infra/maxmind"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type SessionsParams struct {
	// IsAdmin shows session IDs, owners and devices.
	IsAdmin  bool
	Sessions []*SessionEntry

	Pagination Node // nil when there's a single page
}

type SessionEntry struct {
	*sudoapi.Session
	// User is the session's owner, set only when it should be shown (admin filter page).
	User    *kilonova.UserBrief
	Devices []*SessionDeviceEntry
}

type SessionDeviceEntry struct {
	*sudoapi.SessionDevice
	Location *maxmind.Data // nil when unknown
}

func Sessions(ctx context.Context, params SessionsParams) Node {
	return Group{
		P(Class("text-muted text-sm"), ui.T(ctx, "session_mgmt_wip")),
		Div(Class("segment-panel"),
			H1(ui.T(ctx, "manage_sessions")),
			params.Pagination,
			Map(params.Sessions, func(s *SessionEntry) Node { return session(ctx, params.IsAdmin, s) }),
		),
	}
}

func session(ctx context.Context, isAdmin bool, s *SessionEntry) Node {
	item := func(key string, val ...Node) Node { return Li(ui.T(ctx, key), Text(": "), Group(val)) }
	return Div(Class("segment-panel reset-list"),
		Ul(
			If(isAdmin, item("id", Code(Text(s.ID)))),
			item("created_at", ui.ServerTimestamp(s.CreatedAt)),
			item("expires_at", ui.ServerTimestamp(s.ExpiresAt)),
			item("expired", Text(strconv.FormatBool(s.Expired()))),
			Iff(isAdmin && s.User != nil, func() Node {
				return item("username", A(Href("/profile/"+s.User.Name), Text(s.User.Name+" (#"+strconv.Itoa(s.User.ID)+")")))
			}),
			If(isAdmin && len(s.Devices) > 0, item("session_devices", Ul(Map(s.Devices, device)))),
		),
	)
}

func device(d *SessionDeviceEntry) Node {
	return Li(
		Iff(d.IPAddr != nil, func() Node {
			return Group{Text(d.IPAddr.String()), Iff(d.Location != nil, func() Node { return location(d.Location) })}
		}),
		Iff(d.UserAgent != nil, func() Node {
			return Group{Text(" ("), Code(Text(*d.UserAgent)), Text(")")}
		}),
	)
}

func location(l *maxmind.Data) Node {
	if l.City == "" && l.Country == "" && len(l.Subdivisions) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(" (")
	if l.City != "" {
		b.WriteString(l.City + ", ")
	}
	for _, s := range l.Subdivisions {
		b.WriteString(s + ", ")
	}
	if l.Country != "" {
		b.WriteString(" " + l.Country)
	}
	b.WriteString(")")
	return Text(b.String())
}
