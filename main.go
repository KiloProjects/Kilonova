package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/internal/config"
	"github.com/KiloProjects/Kilonova/internal/db"
	"github.com/KiloProjects/Kilonova/internal/logic"
	"github.com/urfave/cli/v2"

	_ "github.com/lib/pq"
)

//go:generate pkger

var (
	confPath = flag.String("config", "./config.toml", "Config path")
)

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()
	if err := config.Load(*confPath); err != nil {
		panic(err)
	}

	app := &cli.App{
		Name:    "Kilonova",
		Usage:   "Control the platform",
		Version: logic.Version,
		// flags won't be used, they are here so an error isnt thrown
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "Config path",
				Value: "./config.toml",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "main",
				Usage:  "Website/API",
				Flags:  []cli.Flag{},
				Action: Kilonova,
			},
			{
				Name:   "eval",
				Usage:  "Eval Server",
				Action: Eval,
			},
			{
				Name:   "migrateTests",
				Usage:  "Migrate test storage, should be used when updating from v0.6.0 to v0.6.1",
				Action: migrate,
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Println(err)
	}
	os.Exit(0)
}

// Whoever reads this code in the future, this is just lazy code
// don't expect it to be a masterpiece, it will be deleted anyway
func migrate(_ *cli.Context) error {
	mgr, err := datamanager.NewManager(config.C.Common.DataDir)
	if err != nil {
		return err
	}
	dbc, err := db.New(config.C.Database.String(), nil)
	_, _ = mgr, dbc
	s, err := os.ReadDir(path.Join(config.C.Common.DataDir, "problems"))
	if err != nil {
		return err
	}
	for _, entry := range s {
		pbID := entry.Name()
		inDir := path.Join(config.C.Common.DataDir, "problems", entry.Name(), "input")
		outDir := path.Join(config.C.Common.DataDir, "problems", entry.Name(), "output")

		in, err := os.ReadDir(inDir)
		if err == nil {
			for _, entry1 := range in {
				testID := strings.TrimSuffix(entry1.Name(), ".txt")
				test, err := dbc.Test(context.Background(), must(pbID), must(testID))
				if err != nil {
					continue
				}
				log.Println(pbID, testID, "in", test.ID)
				f, err := os.Open(path.Join(inDir, entry1.Name()))
				if err != nil {
					log.Println("Couldnt open test in", pbID, testID)
					continue
				}
				if err := mgr.SaveTestInput(test.ID, f); err != nil {
					log.Println("Couldnt save test in", test.ID)
					f.Close()
					continue
				}
				f.Close()
			}
		}
		out, err := os.ReadDir(outDir)
		if err == nil {
			for _, entry1 := range out {
				testID := strings.TrimSuffix(entry1.Name(), ".txt")
				test, err := dbc.Test(context.Background(), must(pbID), must(testID))
				if err != nil {
					continue
				}
				log.Println(pbID, testID, "out", test.ID)
				f, err := os.Open(path.Join(outDir, entry1.Name()))
				if err != nil {
					log.Println("Couldnt open test out", pbID, testID)
					continue
				}
				if err := mgr.SaveTestOutput(test.ID, f); err != nil {
					log.Println("Couldnt save test out", test.ID)
					f.Close()
					continue
				}
				f.Close()
			}
		}
	}
	return nil
}

func must(id string) int64 {
	x, err := strconv.Atoi(id)
	if err != nil {
		panic(err)
	}
	return int64(x)
}
