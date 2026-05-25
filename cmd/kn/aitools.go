package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var aiTools = &cli.Command{
	Name: "statement-export",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "outputPath",
			Aliases:  []string{"o"},
			Usage:    "Output path for statement export",
			Required: true,
		},
		&cli.IntFlag{
			Name:  "listID",
			Usage: "Problem List ID",
		},
		&cli.BoolFlag{
			Name:  "saveSubmissions",
			Usage: "Also save submissions",
		},
	},
	Action: AITools,
}

type statementFrontMatter struct {
	// ID is mostly for human interaction
	ID int `yaml:"problem_id"`
	// Name might give a small hint to the models
	Name string `yaml:"problem_name"`
	// Contest and author tags have been removed
	// Only method hints are included
	Tags     []string `yaml:"tags,flow,omitempty"`
	Language string   `yaml:"language"`

	// Problems that support only C++ solutions
	OnlyCPP bool `yaml:"only_cpp"`

	TimeLimit   float64 `yaml:"time_limit_s"`
	MemoryLimit float64 `yaml:"memory_limit_mb"`

	DefaultPoints float64 `yaml:"default_points"`

	// If console_input is true, input_filename and output_filename are hidden
	ConsoleInput bool `yaml:"console_input"`
	// test_name + ".in"
	InputFilename *string `yaml:"input_filename,omitempty"`
	// test_name + ".out"
	OutputFilename *string `yaml:"output_filename,omitempty"`

	// Has checker
	MultipleSolutions bool `yaml:"multiple_solutions"`

	// For humans to compare relative difficulty
	Source string `yaml:"original_source"`
}

func getFile(matter statementFrontMatter, text []byte, lang string) ([]byte, error) {
	matter.Language = lang
	frontMatter, err := yaml.Marshal(matter)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(frontMatter)
	buf.WriteString("---\n")
	buf.Write(text)
	return buf.Bytes(), nil
}

func saveProblemList(ctx context.Context, cmd *cli.Command, base *sudoapi.BaseAPI, list *sudoapi.FullProblemList, dataPath string) error {
	for _, list := range list.SubLists {
		newPath := path.Join(dataPath, list.Title)
		if err := os.MkdirAll(newPath, 0777); err != nil {
			return err
		}

		if err := saveProblemList(ctx, cmd, base, list, newPath); err != nil {
			return err
		}
	}
	for _, pb := range list.Problems {
		settings, err := base.ProblemSettings(ctx, &pb.Problem)
		if err != nil {
			zap.S().Fatal(err)
		}
		if slices.Contains(settings.LanguageWhitelist, "output_only") {
			continue
		}
		atts, err := base.ProblemAttachments(ctx, pb.ID)
		if err != nil {
			zap.S().Fatal(err)
		}
		if strings.TrimSpace(pb.SourceCredits) == "" {
			pb.SourceCredits = "Unknown"
		}
		// lists, err := base.ProblemParentLists(ctx, pb.ID, true)
		// if err != nil {
		// 	zap.S().Fatal(err)
		// }
		var mdRO, mdEN int
		for _, att := range atts {
			switch att.Name {
			case "statement-ro.md":
				mdRO = att.ID
			case "statement-en.md":
				mdEN = att.ID
			}
		}
		tags, err := base.ProblemTags(ctx, pb.ID)
		if err != nil {
			zap.S().Fatal(err)
		}
		var tagNames []string
		for _, tag := range tags {
			if tag.Type == kilonova.TagTypeMethod {
				tagNames = append(tagNames, tag.Name)
			}
		}

		dPoints, _ := pb.DefaultPoints.Float64()

		var matter = statementFrontMatter{
			ID:                pb.ID,
			Name:              pb.Name,
			Tags:              tagNames,
			OnlyCPP:           len(settings.HeaderFiles) > 0 || len(settings.GraderFiles) > 0,
			TimeLimit:         pb.TimeLimit,
			MemoryLimit:       float64(pb.MemoryLimit) / 1024.0,
			ConsoleInput:      pb.ConsoleInput,
			MultipleSolutions: settings.CheckerName != "",

			DefaultPoints: dPoints,

			Source: pb.SourceCredits,
		}
		if !pb.ConsoleInput {
			var filename = pb.TestName + ".in"
			if filename == ".in" {
				zap.S().Warn("WTF")
			}
			matter.InputFilename = &filename
			filename = pb.TestName + ".out"
			matter.OutputFilename = &filename
		}
		if mdRO > 0 {
			data, err := base.AttachmentData(ctx, mdRO)
			if err != nil {
				zap.S().Fatal(err)
			}
			f, err1 := getFile(matter, data, "romanian")
			if err1 != nil {
				zap.S().Fatal(err)
			}
			if err := os.WriteFile(
				path.Join(dataPath, fmt.Sprintf("%04d_ro.md", pb.ID)),
				f,
				0666,
			); err != nil {
				zap.S().Fatal(err)
			}

		}
		if mdEN > 0 {
			data, err := base.AttachmentData(ctx, mdEN)
			if err != nil {
				zap.S().Fatal(err)
			}
			f, err1 := getFile(matter, data, "english")
			if err1 != nil {
				zap.S().Fatal(err)
			}
			if err := os.WriteFile(
				path.Join(dataPath, fmt.Sprintf("%04d_en.md", pb.ID)),
				f,
				0666,
			); err != nil {
				zap.S().Fatal(err)
			}
		}

		if cmd.Bool("saveSubmissions") {
			hundred := decimal.NewFromInt(100)
			authorSubs, err := base.RawSubmissions(ctx, kilonova.SubmissionFilter{ProblemID: &pb.ID, FromAuthors: true, Score: &hundred})
			if err != nil {
				zap.S().Fatal(err)
			}
			uid := 1
			alexvSubs, err := base.RawSubmissions(ctx, kilonova.SubmissionFilter{ProblemID: &pb.ID, UserID: &uid, Score: &hundred})
			if err != nil {
				zap.S().Fatal(err)
			}
			subs := slices.Concat(authorSubs, alexvSubs)
			if len(subs) == 0 {
				subs, err = base.RawSubmissions(ctx, kilonova.SubmissionFilter{ProblemID: &pb.ID, Score: &hundred})
				if err != nil {
					zap.S().Fatal(err)
				}
			}

			slices.SortFunc(subs, func(a, b *kilonova.Submission) int { return a.CreatedAt.Compare(b.CreatedAt) })
			fmt.Println(pb.ID, len(subs))
			if len(subs) > 5 {
				subs = subs[:5]
			}

			if len(subs) < 5 {
				subs2, err := base.RawSubmissions(ctx, kilonova.SubmissionFilter{ProblemID: &pb.ID, Score: &hundred, Limit: 5 - len(subs)})
				if err != nil {
					zap.S().Fatal(err)
				}
				subs = slices.Concat(subs, subs2)
			}

			if err := os.MkdirAll(path.Join(dataPath, "submissions", strconv.Itoa(pb.ID)), 0777); err != nil {
				zap.S().Fatal(err)
			}

			for _, sub := range subs {
				lang := base.Language(sub.Language)
				extensions := lang.Extensions()
				f, err := os.Create(path.Join(dataPath, "submissions", strconv.Itoa(pb.ID), fmt.Sprintf("%d-%sp%s", sub.ID, sub.Score.String(), extensions[len(extensions)-1])))
				if err != nil {
					zap.S().Fatal(err)
				}
				code, err1 := base.RawSubmissionCode(ctx, sub.ID)
				if err1 != nil {
					zap.S().Fatal(err)
				}
				n, err := f.Write(code)
				if err != nil || n < len(code) {
					zap.S().Fatal(err)
				}
				if err := f.Close(); err != nil {
					zap.S().Fatal(err)
				}
			}
		}
	}
	return nil
}

func AITools(ctx context.Context, command *cli.Command) error {
	dataPath := command.String("outputPath")
	base, err := sudoapi.InitializeBaseAPI(ctx, command)
	if err != nil {
		return err
	}
	defer base.Close()

	list, err := base.FullProblemList(ctx, command.Int("listID"), nil, nil)
	if err != nil {
		return err
	}

	if err := saveProblemList(ctx, command, base, list, dataPath); err != nil {
		return err
	}

	return nil
}
