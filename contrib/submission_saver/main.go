package main

import (
	"archive/zip"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
)

var (
	confPath = flag.String("config", "./config.toml", "Config path")
	username = flag.String("username", "", "Username to save submissions for")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	if len(*username) == 0 {
		slog.ErrorContext(ctx, "Empty username")
		os.Exit(1)
	}

	config.SetConfigPath(*confPath)
	if err := config.Load(); err != nil {
		slog.ErrorContext(ctx, "Error loading config", slog.Any("err", err))
		os.Exit(1)
	}

	if err := Kilonova(ctx); err != nil {
		slog.ErrorContext(ctx, "Error processing", slog.Any("err", err))
		os.Exit(1)
	}
}

func Kilonova(ctx context.Context) error {

	// Print welcome message
	slog.InfoContext(ctx, "Starting Kilonova Submission Exporter")
	slog.InfoContext(ctx, "Saving for user", slog.Any("user", *username))

	base, err := sudoapi.InitializeBaseAPI(ctx)
	if err != nil {
		return err
	}
	defer base.Close()

	user, err := base.UserBriefByName(ctx, *username)
	if err != nil {
		return err
	}

	subs, err := base.RawSubmissions(ctx, kilonova.SubmissionFilter{UserID: &user.ID})
	if err != nil {
		slog.ErrorContext(ctx, "Couldn't get submissions", slog.Any("err", err))
		os.Exit(1)
	}

	f, err := os.Create("./" + user.Name + ".zip")
	if err != nil {
		slog.ErrorContext(ctx, "Couldn't create archive file", slog.Any("err", err))
		os.Exit(1)
	}

	defer func() {
		if err := f.Close(); err != nil {
			slog.ErrorContext(ctx, "Couldn't close file", slog.Any("err", err))
		}
	}()

	wr := zip.NewWriter(f)

	for _, sub := range subs {
		pb, err := base.Problem(ctx, sub.ProblemID)
		if err != nil {
			return err
		}
		code, err := base.RawSubmissionCode(ctx, sub.ID)
		if err != nil {
			return err
		}

		ext := base.Language(ctx, sub.Language).Extension()
		w, err := wr.Create(fmt.Sprintf("%d-%s-%dp%s", sub.ID, pb.TestName, sub.Score.IntPart(), ext))
		if err != nil {
			return err
		}
		if _, err := w.Write(code); err != nil {
			return err
		}
	}

	return wr.Close()
}
