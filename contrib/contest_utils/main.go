package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	confPath = flag.String("config", "./config.toml", "Config path")
	flagPath = flag.String("flags", "./flags.json", "Flag configuration path")

	teamsPath = flag.String("profiles", "./profiles.json", "Profiles file")

	dryRun  = flag.Bool("dryRun", false, "Dry run to check data")
	fullRun = flag.Bool("fullRun", false, "Also submit emails")
)

type Profile struct {
	Name string `json:"name"`
	Slug string `json:"slug"`

	Email string `json:"email"`
	// Password is in the generated output
	Password string `json:"password"`

	Online      bool     `json:"online"`
	MemberNames []string `json:"member_names"`
}

type Configuration struct {
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

func Kilonova() error {
	ctx := context.Background()

	// Print welcome message
	slog.Info("Starting Kilonova Contest Registration Manager")

	base, err1 := sudoapi.InitializeBaseAPI(ctx)
	if err1 != nil {
		return err1
	}
	defer base.Close()

	var config Configuration
	configDataRaw, err := os.ReadFile(*teamsPath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(configDataRaw, &config); err != nil {
		return err
	}

	if !(config.Language == "en" || config.Language == "ro") {
		return fmt.Errorf("invalid language: %q", config.Language)
	}

	if _, err := base.Contest(ctx, config.ContestID); err != nil {
		return err
	}

	for i := range config.Profiles {
		if len(config.Profiles[i].Slug) == 0 {
			t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
			normalized, _, _ := transform.String(t, config.Profiles[i].Name)
			config.Profiles[i].Slug = strings.TrimSpace(strings.ReplaceAll(
				strings.ReplaceAll(normalized, " ", "_"),
				"-", "_",
			))
		}
		if err := base.CheckValidUsername(config.Profiles[i].Slug); err != nil {
			return err
		}
	}

	outTeams := make([]Profile, 0, len(config.Profiles))

	if *dryRun {
		outTeams = config.Profiles
		spew.Dump(config.Profiles)
	} else {
		var subject *string
		if len(config.MailSubject) == 0 {
			subject = &config.MailSubject
		}
		for _, team := range config.Profiles {

			var bio string
			if config.WriteBio {
				bio = fmt.Sprintf(
					"Team name: %s\nOnline: %t\nContestants: %s",
					team.Name,
					team.Online,
					strings.Join(team.MemberNames, ", "),
				)
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

			req := sudoapi.UserGenerationRequest{
				Name: team.Slug,
				Lang: config.Language,

				Bio: bio,

				Email:       &team.Email,
				DisplayName: &displayName,

				ContestID:      &config.ContestID,
				PasswordByMail: *fullRun,
				MailSubject:    subject,
			}

			pwd, _, err := base.GenerateUserFlow(ctx, req)
			if err != nil {
				slog.Error("Error creating user", slog.Any("err", err))
			}

			team.Password = pwd
			outTeams = append(outTeams, team)
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
}

func initLogger(debug bool) {
	core := kilonova.GetZapCore(debug, true, os.Stdout)
	logg := zap.New(core, zap.AddCaller())

	zap.ReplaceGlobals(logg)

	slog.SetDefault(slog.New(zapslog.NewHandler(core, &zapslog.HandlerOptions{AddSource: true})))
}

func init() {
	initLogger(true)
}

func main() {
	flag.Parse()

	config.SetConfigPath(*confPath)
	config.SetConfigV2Path(*flagPath)
	if err := config.Load(); err != nil {
		slog.Error("Could not load config", slog.Any("err", err))
		os.Exit(1)
	}
	if err := config.LoadConfigV2(); err != nil {
		slog.Error("Could not load flags", slog.Any("err", err))
		os.Exit(1)
	}

	if err := Kilonova(); err != nil {
		slog.Error("Error running Kilonova", slog.Any("err", err))
		os.Exit(1)
	}

	os.Exit(0)
}
