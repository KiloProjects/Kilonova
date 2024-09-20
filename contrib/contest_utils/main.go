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

	teamsPath = flag.String("teams", "./teams.json", "Teams file")

	dryRun  = flag.Bool("dryRun", false, "Dry run to check data")
	fullRun = flag.Bool("fullRun", false, "Also submit emails")

	contestID = flag.Int("contestID", -1, "Contest ID to add")

	subject = flag.String("mail_subject", "", "Mail subject")
)

type Team struct {
	Name string `json:"team_name"`
	Slug string `json:"team_slug"`

	Online bool `json:"online"`

	MemberNames []string `json:"member_names"`

	Email string `json:"email"`

	Password string `json:"password"`
}

func Kilonova() error {
	ctx := context.Background()

	// Print welcome message
	zap.S().Infof("Starting Kilonova Contest Registration Manager")

	base, err1 := sudoapi.InitializeBaseAPI(ctx)
	if err1 != nil {
		return err1
	}
	defer base.Close()

	var teams []Team
	teamsRaw, err := os.ReadFile(*teamsPath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(teamsRaw, &teams); err != nil {
		return err
	}

	if _, err := base.Contest(ctx, *contestID); err != nil {
		return err
	}

	for i := range teams {
		if len(teams[i].Slug) == 0 {
			t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
			normalized, _, _ := transform.String(t, teams[i].Name)
			teams[i].Slug = strings.TrimSpace(strings.ReplaceAll(
				strings.ReplaceAll(normalized, " ", "_"),
				"-", "_",
			))
		}
		if err := base.CheckValidUsername(teams[i].Slug); err != nil {
			return err
		}
	}

	outTeams := make([]Team, 0, len(teams))

	if *dryRun {
		outTeams = teams
		spew.Dump(teams)
	} else {
		if len(*subject) == 0 {
			subject = nil
		}
		for _, team := range teams {

			bio := fmt.Sprintf(
				"Team name: %s\nOnline: %t\nContestants: %s",
				team.Name,
				team.Online,
				strings.Join(team.MemberNames, ", "),
			)

			displayName := team.Name

			if len(team.MemberNames) > 0 {
				var lastNames []string
				for _, name := range team.MemberNames {
					first, _, _ := strings.Cut(name, " ")
					lastNames = append(lastNames, first)
				}
				displayName = strings.Join(lastNames, ", ")
			}

			req := sudoapi.UserGenerationRequest{
				Name: team.Slug,
				Lang: "ro",

				Bio: bio,

				Email:       &team.Email,
				DisplayName: &displayName,

				ContestID:      contestID,
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

	teamsRaw, err = json.Marshal(outTeams)
	if err != nil {
		return err
	}
	if err := os.WriteFile("./teamsOut.json", teamsRaw, 0644); err != nil {
		return err
	}

	var buf bytes.Buffer
	cw := csv.NewWriter(&buf)

	cw.Write([]string{"Team Name", "Online/Physical", "Members", "Contact Email", "Username", "Password"})
	for _, t := range outTeams {
		status := "Physical"
		if t.Online {
			status = "Online"
		}
		cw.Write([]string{
			t.Name, status, strings.Join(t.MemberNames, ", "), t.Email, t.Slug, t.Password,
		})
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
		zap.S().Fatal(err)
	}
	if err := config.LoadConfigV2(); err != nil {
		zap.S().Fatal(err)
	}

	if err := Kilonova(); err != nil {
		zap.S().Fatal(err)
	}

	os.Exit(0)
}
