package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/KiloProjects/Kilonova/internal/config"
	"github.com/KiloProjects/Kilonova/internal/logic"
	"github.com/urfave/cli/v2"

	_ "github.com/lib/pq"
)

//go:generate pkger

var (
	confPath = flag.String("config", "./config.toml", "Config path")
)

func main() { // TODO: finish this
	rand.Seed(time.Now().UnixNano())
	flag.Parse()
	if err := config.Load(*confPath); err != nil {
		panic(err)
	}

	app := &cli.App{
		Name:    "Kilonova",
		Usage:   "Control the platform",
		Version: logic.Version,
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
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Println(err)
	}
	os.Exit(0)
}
