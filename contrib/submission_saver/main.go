package main

import (
	"archive/zip"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
	"go.uber.org/zap"
)

var (
	confPath = flag.String("config", "./config.toml", "Config path")
	username = flag.String("username", "", "Username to save submissions for")
)

func main() {
	flag.Parse()

	if len(*username) == 0 {
		zap.S().Fatal("Empty username")
	}

	config.SetConfigPath(*confPath)
	if err := config.Load(); err != nil {
		zap.S().Fatal(err)
	}

	if err := Kilonova(); err != nil {
		zap.S().Fatal(err)
	}

	os.Exit(0)
}

func Kilonova() error {
	ctx := context.Background()

	// Print welcome message
	zap.S().Infof("Starting Kilonova Submission Exporter")
	zap.S().Infof("Saving for user %q...", *username)

	base, err := sudoapi.InitializeBaseAPI(context.Background())
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
		zap.S().Fatal(err)
	}

	f, err1 := os.Create("./" + user.Name + ".zip")
	if err1 != nil {
		zap.S().Fatal(err1)
	}

	defer func() {
		if err := f.Close(); err != nil {
			zap.S().Warn(err)
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
		w, err1 := wr.Create(fmt.Sprintf("%d-%s-%dp%s", sub.ID, pb.TestName, sub.Score.IntPart(), ext))
		if err1 != nil {
			return err1
		}
		if _, err := w.Write(code); err != nil {
			return err
		}
	}

	return wr.Close()
}

func initLogger(debug bool) {
	core := kilonova.GetZapCore(debug, os.Stdout)
	logg := zap.New(core, zap.AddCaller())

	zap.ReplaceGlobals(logg)
}

func init() {
	initLogger(true)
}
