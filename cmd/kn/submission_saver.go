package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/klauspost/compress/zip"
	"github.com/urfave/cli/v3"
)

var submissionSaver = &cli.Command{
	Name: "submission-saver",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "username",
			Aliases: []string{"u"},
			Usage:   "Username to save submissions for",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {

		// Print welcome message
		slog.InfoContext(ctx, "Starting Kilonova Submission Exporter")
		slog.InfoContext(ctx, "Saving for user", slog.Any("user", cmd.String("username")))

		base, err := sudoapi.InitializeBaseAPI(ctx)
		if err != nil {
			return err
		}
		defer base.Close()

		user, err := base.UserBriefByName(ctx, cmd.String("username"))
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
	},
}
