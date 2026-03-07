package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/davecgh/go-spew/spew"
	"github.com/urfave/cli/v3"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type ContestProfile struct {
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

type ContestConfiguration struct {
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

	Profiles []ContestProfile `json:"profiles"`
}

var contestUtils = &cli.Command{
	Name: "contest-utils",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "profiles",
			Value: "./profiles.json",
			Usage: "Profiles file",
		},
		&cli.BoolFlag{
			Name:  "dryRun",
			Usage: "Dry run to check data",
			Value: false,
		},
	},
	Action: func(ctx context.Context, command *cli.Command) error {

		auditLogFile, err := os.Create("./generator-" + time.Now().Format(time.RFC3339) + ".log")
		if err != nil {
			return err
		}
		defer func() {
			if err := auditLogFile.Close(); err != nil {
				slog.WarnContext(ctx, "Error closing audit log", slog.Any("err", err))
			}
		}()

		handler := slog.NewJSONHandler(auditLogFile, &slog.HandlerOptions{
			AddSource: true,
		})

		auditLog := slog.New(handler)

		// Print welcome message
		slog.InfoContext(ctx, "Starting Kilonova Contest Registration Manager")

		base, err := sudoapi.InitializeBaseAPI(ctx)
		if err != nil {
			return err
		}
		defer base.Close()

		var config ContestConfiguration
		configDataRaw, err := os.ReadFile(command.String("profiles"))
		if err != nil {
			return err
		}
		if err := json.Unmarshal(configDataRaw, &config); err != nil {
			return err
		}

		if !(config.Language == "en" || config.Language == "ro") {
			return fmt.Errorf("invalid language: %q", config.Language)
		}

		if config.ContestID > 0 {
			if _, err := base.Contest(ctx, config.ContestID); err != nil {
				return err
			}
		}

		var anyExternalID bool
		for i := range config.Profiles {
			if len(config.Profiles[i].Slug) == 0 {
				t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
				normalized, _, _ := transform.String(t, config.Profiles[i].Name)
				config.Profiles[i].Slug = strings.TrimSpace(strings.ReplaceAll(
					strings.ReplaceAll(normalized, " ", "_"),
					"-", "_",
				))
			}
			fmt.Println(config.Profiles[i].Slug)
			if err := base.CheckValidUsername(config.Profiles[i].Slug); err != nil {
				return err
			}
			if config.Profiles[i].ExternalID != nil {
				anyExternalID = true
			}
		}

		outTeams := make([]ContestProfile, 0, len(config.Profiles))

		if command.Bool("dryRun") {
			outTeams = config.Profiles
			spew.Dump(config.Profiles)
		} else {
			var subject *string
			if len(config.MailSubject) > 0 {
				subject = &config.MailSubject
			}
			for _, team := range config.Profiles {
				if len(team.Password) > 0 {
					slog.InfoContext(ctx, "Skipping already created account", slog.String("slug", team.Slug))
					continue
				}

				if len(team.Email) == 0 {
					if config.SendMail {
						slog.ErrorContext(ctx, "SendMail is set but no email specified for user, skipping..", slog.String("slug", team.Slug))
						auditLog.ErrorContext(ctx, "SendMail is set but no email specified for user, skipping..", slog.String("slug", team.Slug))
						continue
					}
					team.Email = "email_" + team.Slug + "@kilonova.ro"
				}

				user, err := base.UserFullByName(ctx, team.Slug)
				if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
					slog.ErrorContext(ctx, "Could not test user existence", slog.Any("err", err))
					continue
				} else if user != nil && user.Email == team.Email {
					slog.InfoContext(ctx, "Skipping already created account, even though it has no password in profile", slog.String("slug", team.Slug))
					continue
				}

				var bio strings.Builder
				if config.WriteBio {
					fmt.Fprintf(&bio, "Name: %s\n", team.Name)
					if config.Hybrid {
						fmt.Fprintf(&bio, "Online: %t\n", team.Online)
					}
					if config.Teams {
						fmt.Fprintf(&bio, "Contestants: %s\n", strings.Join(team.MemberNames, ", "))
					}

				}
				if len(team.BioData) > 0 {
					fmt.Fprintln(&bio, "\n", team.BioData)
				}

				displayName := team.Name

				if config.Teams && len(team.MemberNames) > 0 {
					var lastNames []string
					for _, name := range team.MemberNames {
						first, _, _ := strings.Cut(name, " ")
						lastNames = append(lastNames, first)
					}
					displayName = strings.Join(lastNames, ", ")
				}

				contestID := config.ContestID
				if team.ContestID > 0 {
					contestID = team.ContestID
				}

				if contestID <= 0 {
					auditLog.ErrorContext(ctx, "Did not create: no contest ID for user", slog.String("email", team.Email), slog.String("slug", team.Slug))
					slog.ErrorContext(ctx, "Did not create: no contest ID for user", slog.String("email", team.Email), slog.String("slug", team.Slug))
					continue
				}

				req := sudoapi.UserGenerationRequest{
					Name: team.Slug,
					Lang: config.Language,

					Bio: bio.String(),

					Email:       &team.Email,
					DisplayName: &displayName,

					ContestID:      &contestID,
					PasswordByMail: config.SendMail,
					MailSubject:    subject,
				}

				pwd, _, err := base.GenerateUserFlow(ctx, req)
				if err != nil {
					slog.ErrorContext(ctx, "Error creating user", slog.Any("err", err))
					continue
				}
				auditLog.InfoContext(ctx, "Created user", slog.String("email", team.Email), slog.String("password", pwd), slog.String("slug", team.Slug), slog.Int("contestID", contestID))
				slog.InfoContext(ctx, "Created user", slog.String("email", team.Email), slog.String("password", pwd), slog.String("slug", team.Slug), slog.Int("contestID", contestID))

				team.Password = pwd
				outTeams = append(outTeams, team)

				if config.SendMail {
					time.Sleep(100 * time.Millisecond)
				}
			}
		}

		if len(outTeams) == 0 {
			return nil
		}

		teamsRaw, err := json.Marshal(outTeams)
		if err != nil {
			return err
		}
		if err := os.WriteFile("./teamsOut.json", teamsRaw, 0644); err != nil {
			return err
		}

		var buf bytes.Buffer
		cw := csv.NewWriter(&buf)

		headers := []string{"Team Name", "Online/Physical", "Members", "Contact Email", "Username", "Password"}
		if !config.Hybrid {
			headers = slices.Delete(headers, 1, 2)
		}
		if !config.Teams {
			headers = slices.Delete(headers, 1, 2)
		}
		if anyExternalID {
			headers = append(headers, "External ID")
		}
		cw.Write(headers)
		for _, t := range outTeams {
			status := "Physical"
			if t.Online {
				status = "Online"
			}

			values := []string{
				t.Name, status, strings.Join(t.MemberNames, ", "), t.Email, t.Slug, t.Password,
			}
			if !config.Hybrid {
				values = slices.Delete(values, 1, 2)
			}
			if !config.Teams {
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
		if err := cw.Error(); err != nil {
			return err
		}
		if err := os.WriteFile("./teamsOut.csv", buf.Bytes(), 0644); err != nil {
			return err
		}

		return nil
	},
}
