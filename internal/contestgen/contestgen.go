// Package contestgen bulk-creates contest accounts from a profiles bundle.
// It is shared by the contest-utils CLI commands and the admin web page.
package contestgen

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/sudoapi"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type Profile struct {
	Name string `json:"name"`
	Slug string `json:"slug"`

	Email string `json:"email"`
	// Password is in the generated output
	Password string `json:"password"`

	// Configuration override
	ContestID int    `json:"contest_id"`
	BioData   string `json:"bio_data"`

	// For printing purposes
	ExternalID any `json:"external_id,omitempty,omitzero"`

	Online      bool     `json:"online"`
	MemberNames []string `json:"member_names"`
}

type Config struct {
	ContestID int    `json:"contest_id"`
	Language  string `json:"language"`

	SendMail    bool   `json:"send_mail"`
	MailSubject string `json:"mail_subject"`

	// If true, additional data is written into the user bios
	WriteBio bool `json:"write_bio"`
	// If true, Profile.Online is recorded in the final CSV
	Hybrid bool `json:"hybrid"`
	// If true, Profile.MemberNames is used
	Teams bool `json:"teams"`

	Profiles []Profile `json:"profiles"`
}

// ErrAlreadyCreated is returned by Backend.Generate when the account already exists.
var ErrAlreadyCreated = errors.New("account already created")

// Backend abstracts where accounts get created: directly through sudoapi or through the HTTP API.
type Backend struct {
	CheckContest func(ctx context.Context, id int) error
	Generate     func(ctx context.Context, req sudoapi.UserGenerationRequest) (string, error)
}

// LocalBackend creates accounts directly through sudoapi.
func LocalBackend(base *sudoapi.BaseAPI) Backend {
	return Backend{
		CheckContest: func(ctx context.Context, id int) error {
			_, err := base.Contest(ctx, id)
			return err
		},
		Generate: func(ctx context.Context, req sudoapi.UserGenerationRequest) (string, error) {
			u, err := base.UserFullByName(ctx, req.Name)
			if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
				return "", fmt.Errorf("could not test user existence: %w", err)
			} else if u != nil && u.Email == *req.Email {
				return "", ErrAlreadyCreated
			}
			pwd, _, err := base.GenerateUserFlow(ctx, req)
			return pwd, err
		},
	}
}

// Parse decodes and validates a profiles bundle, filling in missing slugs.
func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid profiles bundle: %w", err)
	}
	if cfg.Language != "en" && cfg.Language != "ro" {
		return nil, fmt.Errorf("invalid language: %q", cfg.Language)
	}
	for i := range cfg.Profiles {
		p := &cfg.Profiles[i]
		if len(p.Slug) == 0 {
			t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
			normalized, _, _ := transform.String(t, p.Name)
			p.Slug = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(normalized, " ", "_"), "-", "_"))
		}
		if err := user.ValidUsername(p.Slug); err != nil {
			return nil, fmt.Errorf("profile %q: %w", p.Slug, err)
		}
	}
	return &cfg, nil
}

type Result struct {
	cfg *Config
	// Teams holds the created accounts, with passwords.
	Teams []Profile
	// Requests holds what would have been sent, on a dry run.
	Requests []sudoapi.UserGenerationRequest
}

// Run creates the accounts in cfg. Failures for single profiles are logged and skipped.
// On a dry run, the requests are collected in Result.Requests instead of being sent.
func Run(ctx context.Context, cfg *Config, backend Backend, auditLog *slog.Logger, dryRun bool) (*Result, error) {
	if cfg.ContestID > 0 {
		if err := backend.CheckContest(ctx, cfg.ContestID); err != nil {
			return nil, err
		}
	}

	res := &Result{cfg: cfg, Teams: make([]Profile, 0, len(cfg.Profiles))}
	var subject *string
	if len(cfg.MailSubject) > 0 {
		subject = &cfg.MailSubject
	}
	logErr := func(msg string, attrs ...slog.Attr) {
		slog.LogAttrs(ctx, slog.LevelError, msg, attrs...)
		auditLog.LogAttrs(ctx, slog.LevelError, msg, attrs...)
	}
	for _, team := range cfg.Profiles {
		if len(team.Password) > 0 {
			slog.InfoContext(ctx, "Skipping already created account", slog.String("slug", team.Slug))
			continue
		}

		if len(team.Email) == 0 {
			if cfg.SendMail {
				logErr("SendMail is set but no email specified for user, skipping..", slog.String("slug", team.Slug))
				continue
			}
			team.Email = "email_" + team.Slug + "@kilonova.ro"
		}

		var bio strings.Builder
		if cfg.WriteBio {
			fmt.Fprintf(&bio, "Name: %s\n", team.Name)
			if cfg.Hybrid {
				fmt.Fprintf(&bio, "Online: %t\n", team.Online)
			}
			if cfg.Teams {
				fmt.Fprintf(&bio, "Contestants: %s\n", strings.Join(team.MemberNames, ", "))
			}
		}
		if len(team.BioData) > 0 {
			fmt.Fprintln(&bio, "\n", team.BioData)
		}

		displayName := team.Name
		if cfg.Teams && len(team.MemberNames) > 0 && len(displayName) == 0 {
			var lastNames []string
			for _, name := range team.MemberNames {
				first, _, _ := strings.Cut(name, " ")
				lastNames = append(lastNames, first)
			}
			displayName = strings.Join(lastNames, ", ")
		}

		if team.ContestID <= 0 {
			team.ContestID = cfg.ContestID
		}
		if team.ContestID <= 0 {
			logErr("Did not create: no contest ID for user", slog.String("email", team.Email), slog.String("slug", team.Slug))
			continue
		}

		req := sudoapi.UserGenerationRequest{
			Name: team.Slug,
			Lang: cfg.Language,

			Bio: bio.String(),

			Email:       &team.Email,
			DisplayName: &displayName,

			ContestID:      &team.ContestID,
			PasswordByMail: cfg.SendMail,
			MailSubject:    subject,
		}

		if dryRun {
			res.Requests = append(res.Requests, req)
			continue
		}

		pwd, err := backend.Generate(ctx, req)
		if errors.Is(err, ErrAlreadyCreated) {
			slog.InfoContext(ctx, "Skipping already created account, even though it has no password in profile", slog.String("slug", team.Slug))
			continue
		}
		if err != nil {
			logErr("Error creating user", slog.String("slug", team.Slug), slog.Any("err", err))
			continue
		}
		attrs := []slog.Attr{slog.String("email", team.Email), slog.String("password", pwd), slog.String("slug", team.Slug), slog.Int("contestID", team.ContestID)}
		auditLog.LogAttrs(ctx, slog.LevelInfo, "Created user", attrs...)
		slog.LogAttrs(ctx, slog.LevelInfo, "Created user", attrs...)

		team.Password = pwd
		res.Teams = append(res.Teams, team)

		if cfg.SendMail {
			time.Sleep(100 * time.Millisecond)
		}
	}
	return res, nil
}

// Files returns the output artifacts by file name: requestsOut.json on a dry run,
// teamsOut.json and teamsOut.csv otherwise. It is empty when nothing was created.
func (r *Result) Files() (map[string][]byte, error) {
	if r.Requests != nil {
		raw, err := json.MarshalIndent(r.Requests, "", "\t")
		if err != nil {
			return nil, err
		}
		return map[string][]byte{"requestsOut.json": raw}, nil
	}
	if len(r.Teams) == 0 {
		return map[string][]byte{}, nil
	}
	teamsRaw, err := json.Marshal(r.Teams)
	if err != nil {
		return nil, err
	}
	csvRaw, err := r.csv()
	if err != nil {
		return nil, err
	}
	return map[string][]byte{"teamsOut.json": teamsRaw, "teamsOut.csv": csvRaw}, nil
}

func (r *Result) csv() ([]byte, error) {
	anyExternalID := slices.ContainsFunc(r.cfg.Profiles, func(p Profile) bool { return p.ExternalID != nil })

	var buf bytes.Buffer
	cw := csv.NewWriter(&buf)

	headers := []string{"Team Name", "Online/Physical", "Members", "Contact Email", "Username", "Password"}
	if !r.cfg.Hybrid {
		headers = slices.Delete(headers, 1, 2)
	}
	if !r.cfg.Teams {
		headers = slices.Delete(headers, 1, 2)
	}
	if anyExternalID {
		headers = append(headers, "External ID")
	}
	cw.Write(headers)
	for _, t := range r.Teams {
		status := "Physical"
		if t.Online {
			status = "Online"
		}

		values := []string{t.Name, status, strings.Join(t.MemberNames, ", "), t.Email, t.Slug, t.Password}
		if !r.cfg.Hybrid {
			values = slices.Delete(values, 1, 2)
		}
		if !r.cfg.Teams {
			values = slices.Delete(values, 1, 2)
		}
		if anyExternalID {
			val, err := json.Marshal(t.ExternalID)
			if err != nil {
				val = []byte(err.Error())
			}
			values = append(values, string(val))
		}
		cw.Write(values)
	}
	cw.Flush()
	return buf.Bytes(), cw.Error()
}
