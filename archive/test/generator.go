package test

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/sudoapi"
	"go.uber.org/zap"
)

type ArchiveGenOptions struct {
	Brief bool
	Name  bool

	Submissions bool
	Editors     bool
}

func GenerateArchive(ctx context.Context, pb *kilonova.Problem, w io.Writer, base *sudoapi.BaseAPI, opts *ArchiveGenOptions) *kilonova.StatusError {
	ar := zip.NewWriter(w)
	tests, err := base.Tests(ctx, pb.ID)
	defer ar.Close()
	if err != nil {
		return err
	}

	testName := strings.TrimSpace(pb.TestName)
	if testName == "" {
		testName = kilonova.MakeSlug(testName)
	}

	// First, save the tests
	for _, test := range tests {
		{
			f, err := ar.Create(fmt.Sprintf("%d-%s.in", test.VisibleID, testName))
			if err != nil {
				return kilonova.WrapError(err, "Couldn't create archive file")
			}

			r, err := base.TestInput(test.ID)
			if err != nil {
				return kilonova.WrapError(err, "Couldn't get test input")
			}

			if _, err := io.Copy(f, r); err != nil {
				r.Close()
				return kilonova.WrapError(err, "Couldn't save test input file")
			}
			r.Close()
		}
		{
			f, err := ar.Create(fmt.Sprintf("%d-%s.ok", test.VisibleID, testName))
			if err != nil {
				return kilonova.WrapError(err, "Couldn't create archive file")
			}

			r, err := base.TestOutput(test.ID)
			if err != nil {
				return kilonova.WrapError(err, "Couldn't get test output")
			}

			if _, err := io.Copy(f, r); err != nil {
				r.Close()
				return kilonova.WrapError(err, "Couldn't save test output file")
			}
			r.Close()
		}
	}
	if !opts.Brief {
		// Then, attachments, if not brief
		atts, err := base.ProblemAttachments(ctx, pb.ID)
		if err != nil {
			return kilonova.WrapError(err, "Couldn't get attachments")
		}
		for _, att := range atts {
			pFile, err := ar.Create("attachments/" + att.Name + ".att_props")
			if err != nil {
				return kilonova.WrapError(err, "Couldn't create archive attachment props file")
			}
			if err := json.NewEncoder(pFile).Encode(attachmentProps{
				Visible: att.Visible,
				Private: att.Private,
				Exec:    att.Exec,
			}); err != nil {
				return kilonova.WrapError(err, "Couldn't encode attachment props")
			}
			attFile, err := ar.Create("attachments/" + att.Name)
			if err != nil {
				return kilonova.WrapError(err, "Couldn't create attachment file")
			}
			data, err1 := base.AttachmentData(ctx, att.ID)
			if err1 != nil {
				return kilonova.WrapError(err, "Couldn't get attachment data")
			}
			if _, err := attFile.Write(data); err != nil {
				return kilonova.WrapError(err, "Couldn't save attachment file")
			}
		}
	}
	{
		// Then, the scores
		testFile, err := ar.Create("tests.txt")
		if err != nil {
			return kilonova.WrapError(err, "Couldn't create archive tests.txt file")
		}
		for _, test := range tests {
			fmt.Fprintf(testFile, "%d %d\n", test.VisibleID, test.Score)
		}
	}
	{
		// Lastly, grader.properties
		gr, err := ar.Create("grader.properties")
		if err != nil {
			return kilonova.WrapError(err, "Couldn't create archive grader.properties file")
		}

		subtasks, err1 := base.SubTasks(ctx, pb.ID)
		if err1 != nil {
			return err1
		}
		if len(subtasks) != 0 {
			tmap := map[int]*kilonova.Test{}
			for _, test := range tests {
				tmap[test.ID] = test
			}

			groups := []string{}
			weights := []string{}

			for _, st := range subtasks {
				group := ""
				for i, t := range st.Tests {
					if i > 0 {
						group += ";"
					}
					tt, ok := tmap[t]
					if !ok {
						zap.S().Warn("Couldn't find test in test map")
					} else {
						group += strconv.Itoa(tt.VisibleID)
					}
				}
				groups = append(groups, group)
				weights = append(weights, st.Score.String())
			}
			fmt.Fprintf(gr, "groups=%s\n", strings.Join(groups, ","))
			fmt.Fprintf(gr, "weights=%s\n", strings.Join(weights, ","))
		}

		fmt.Fprintf(gr, "time=%f\n", pb.TimeLimit)
		fmt.Fprintf(gr, "memory=%f\n", float64(pb.MemoryLimit)/1024.0)
		if !pb.DefaultPoints.IsZero() {
			fmt.Fprintf(gr, "default_score=%s\n", pb.DefaultPoints)
		}
		fmt.Fprintf(gr, "score_precision=%d\n", pb.ScorePrecision)
		if pb.SourceSize != kilonova.DefaultSourceSize {
			fmt.Fprintf(gr, "source_size=%d", pb.SourceSize)
		}
		fmt.Fprintf(gr, "console_input=%t\n", pb.ConsoleInput)
		fmt.Fprintf(gr, "test_name=%s\n", testName)
		fmt.Fprintf(gr, "scoring_strategy=%s\n", pb.ScoringStrategy)

		if opts.Name {
			fmt.Fprintf(gr, "problem_name=%s\n", pb.Name)
		}

		if !opts.Brief {
			tags, err := base.ProblemTags(ctx, pb.ID)
			if err != nil {
				return err
			}
			if len(tags) > 0 {
				var tagNames []string
				for _, tag := range tags {
					tagNames = append(tagNames, fmt.Sprintf("%q:%s", tag.Name, tag.Type))
				}
				fmt.Fprintf(gr, "tags=%s\n", strings.Join(tagNames, ","))
			}

			if pb.SourceCredits != "" {
				fmt.Fprintf(gr, "source=%s\n", pb.SourceCredits)
			}
		}

		if opts.Editors {
			editors, err := base.ProblemEditors(ctx, pb.ID)
			if err != nil {
				return err
			}
			if len(editors) > 0 {
				var editorNames []string
				for _, editor := range editors {
					editorNames = append(editorNames, editor.Name)
				}
				fmt.Fprintf(gr, "editors=%s\n", strings.Join(editorNames, ","))
			}
		}
	}
	// But if submissions are specified, add them too
	if opts.Submissions {
		subs, err := base.RawSubmissions(ctx, kilonova.SubmissionFilter{ProblemID: &pb.ID})
		if err != nil {
			return err
		}
		for _, sub := range subs {
			lang, ok := eval.Langs[sub.Language]
			if !ok {
				zap.S().Info("Skipping submission due to unknown language ", sub.ID)
				continue
			}
			f, err := ar.Create(fmt.Sprintf("submissions/%d%s", sub.ID, lang.Extensions[len(lang.Extensions)-1]))
			if err != nil {
				return kilonova.WrapError(err, "Couldn't create archive submission file")
			}
			if _, err := io.WriteString(f, sub.Code); err != nil {
				return kilonova.WrapError(err, "Couldn't write submission file")
			}
		}
	}
	return nil
}
