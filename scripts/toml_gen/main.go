package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

var (
	tomlPath = flag.String("toml_path", "./translations.toml", "Path to translation file")
)

type strArray []string

func (i *strArray) String() string {
	return strings.Join(*i, ";")
}

func (i *strArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var outPaths strArray

func main() {
	flag.Var(&outPaths, "target", "File paths where JSON is written. Specify multiple times for multiple targets")
	flag.Parse()

	if len(outPaths) == 0 {
		log.Fatalln("No targets specified")
	}

	var vals any
	_, err := toml.DecodeFile(*tomlPath, &vals)
	if err != nil {
		log.Fatalln(err)
	}

	data, err := json.Marshal(vals)
	if err != nil {
		log.Fatalln(err)
	}

	for _, path := range outPaths {
		if err := os.WriteFile(path, data, 0666); err != nil {
			log.Printf("Could not write translations to %s: %v\n", path, err)
		}
	}
}
