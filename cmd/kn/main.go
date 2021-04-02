package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
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

	// save the config for formatting
	if err := config.Save(*confPath); err != nil {
		panic(err)
	}

	if err := eval.Initialize(); err != nil {
		log.Println("WARNING: Could not initialize the box manager, will likely run in remote grader mode if running main")
	}

	app := &cli.App{
		Name:    "Kilonova",
		Usage:   "Control the platform",
		Version: kilonova.Version,
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
				Action: Kilonova,
			},
			{
				Name:   "eval",
				Usage:  "Eval Server",
				Action: Eval,
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Println(err)
	}
	os.Exit(0)
}
