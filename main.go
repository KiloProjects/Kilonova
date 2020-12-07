package main

import (
	"flag"
	"log"
	"os"

	"github.com/KiloProjects/Kilonova/internal/config"
	"github.com/KiloProjects/Kilonova/internal/version"
	"github.com/davecgh/go-spew/spew"
	"github.com/urfave/cli/v2"

	_ "github.com/lib/pq"
)

//go:generate pkger

var (
	confPath = flag.String("config", "./config.toml", "Config path")
)

func main() { // TODO: finish this
	flag.Parse()
	if err := config.Load(*confPath); err != nil {
		panic(err)
	}

	spew.Dump(config.C)

	app := &cli.App{
		Name:    "Kilonova",
		Usage:   "Control the platform",
		Version: version.Version,
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
