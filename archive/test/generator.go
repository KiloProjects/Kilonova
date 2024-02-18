package test

import (
	"archive/zip"
	"bytes"
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
	Tests bool `json:"tests"`

	Attachments bool `json:"attachments"`
	// If PrivateAttachments is true, the generator also adds attachments marked private
	// It's an option since non-editors should not be able to download private attachments.
	PrivateAttachments bool `json:"private_attachments"`

	// grader.properties options

	// ProblemDetails includes problem metadata such as time/memory limits, problem name, whether it's console input, etc
	ProblemDetails bool `json:"details"`
	// Tags also adds tag list in a safe format
	Tags bool `json:"tags"`

	// Editors also includes a list of editor usernames
	Editors bool `json:"editors"`

	// Submissions only includes submissions from the problem editors,
	// whereas AllSubmissions includes ALL submissions
	Submissions     bool                `json:"submissions"`
	AllSubmissions  bool                `json:"all_submissions"`
	SubsLook        bool                `json:"-"`
	SubsLookingUser *kilonova.UserBrief `json:"-"`
}

type archiveGenerator struct {
	pb   *kilonova.Problem
	ar   *zip.Writer
	base *sudoapi.BaseAPI

	opts *ArchiveGenOptions

	testName string
}

func (ag *archiveGenerator) addTests(ctx context.Context) *kilonova.StatusError {
	tests, err := ag.base.Tests(ctx, ag.pb.ID)
	if err != nil {
		return err
	}

	// Add test files
	for _, test := range tests {
		if err := func() *kilonova.StatusError {
			f, err := ag.ar.Create(fmt.Sprintf("%d-%s.in", test.VisibleID, ag.testName))
			if err != nil {
				return kilonova.WrapError(err, "Couldn't create archive file")
			}

			r, err := ag.base.GraderStore().TestInput(test.ID)
			if err != nil {
				return kilonova.WrapError(err, "Couldn't get test input")
			}
			defer r.Close()

			if _, err := io.Copy(f, r); err != nil {
				return kilonova.WrapError(err, "Couldn't save test input file")
			}
			return nil
		}(); err != nil {
			return err
		}
		if err := func() *kilonova.StatusError {
			f, err := ag.ar.Create(fmt.Sprintf("%d-%s.ok", test.VisibleID, ag.testName))
			if err != nil {
				return kilonova.WrapError(err, "Couldn't create archive file")
			}

			r, err := ag.base.GraderStore().TestOutput(test.ID)
			if err != nil {
				return kilonova.WrapError(err, "Couldn't get test output")
			}
			defer r.Close()

			if _, err := io.Copy(f, r); err != nil {
				return kilonova.WrapError(err, "Couldn't save test output file")
			}
			return nil
		}(); err != nil {
			return err
		}
	}

	// Add test scores
	{
		// Then, the scores
		testFile, err := ag.ar.Create("tests.txt")
		if err != nil {
			return kilonova.WrapError(err, "Couldn't create archive tests.txt file")
		}
		for _, test := range tests {
			fmt.Fprintf(testFile, "%d %s\n", test.VisibleID, test.Score.String())
		}
	}
	return nil
}

func (ag *archiveGenerator) addAttachments(ctx context.Context) *kilonova.StatusError {
	atts, err := ag.base.ProblemAttachments(ctx, ag.pb.ID)
	if err != nil {
		return kilonova.WrapError(err, "Couldn't get attachments")
	}
	for _, att := range atts {
		if att.Private && !ag.opts.PrivateAttachments {
			// Skip private attachments if not downloading them
			continue
		}

		if !(!att.Visible && !att.Private && !att.Exec) {
			// If any of the flags is not false, generate an att_props file
			pFile, err := ag.ar.Create("attachments/" + att.Name + ".att_props")
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
		}
		attFile, err := ag.ar.Create("attachments/" + att.Name)
		if err != nil {
			return kilonova.WrapError(err, "Couldn't create attachment file")
		}
		data, err1 := ag.base.AttachmentData(ctx, att.ID)
		if err1 != nil {
			return kilonova.WrapError(err, "Couldn't get attachment data")
		}
		if _, err := attFile.Write(data); err != nil {
			return kilonova.WrapError(err, "Couldn't save attachment file")
		}
	}
	return nil
}

func (ag *archiveGenerator) addGraderProperties(ctx context.Context) *kilonova.StatusError {
	var buf bytes.Buffer

	if ag.opts.Tests {
		subtasks, err := ag.base.SubTasks(ctx, ag.pb.ID)
		if err != nil {
			return err
		}
		if len(subtasks) != 0 {
			tests, err := ag.base.Tests(ctx, ag.pb.ID)
			if err != nil {
				return err
			}
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
			fmt.Fprintf(&buf, "groups=%s\n", strings.Join(groups, ","))
			fmt.Fprintf(&buf, "weights=%s\n", strings.Join(weights, ","))
		}
	}
	if ag.opts.ProblemDetails {
		fmt.Fprintf(&buf, "time=%f\n", ag.pb.TimeLimit)
		fmt.Fprintf(&buf, "memory=%f\n", float64(ag.pb.MemoryLimit)/1024.0)
		if !ag.pb.DefaultPoints.IsZero() {
			fmt.Fprintf(&buf, "default_score=%s\n", ag.pb.DefaultPoints)
		}
		fmt.Fprintf(&buf, "score_precision=%d\n", ag.pb.ScorePrecision)
		if ag.pb.SourceSize != kilonova.DefaultSourceSize.Value() {
			fmt.Fprintf(&buf, "source_size=%d", ag.pb.SourceSize)
		}
		fmt.Fprintf(&buf, "console_input=%t\n", ag.pb.ConsoleInput)
		fmt.Fprintf(&buf, "test_name=%s\n", ag.testName)
		fmt.Fprintf(&buf, "scoring_strategy=%s\n", ag.pb.ScoringStrategy)

		fmt.Fprintf(&buf, "problem_name=%s\n", ag.pb.Name)

		if ag.pb.SourceCredits != "" {
			fmt.Fprintf(&buf, "source=%s\n", ag.pb.SourceCredits)
		}
	}

	if ag.opts.Tags {
		tags, err := ag.base.ProblemTags(ctx, ag.pb.ID)
		if err != nil {
			return err
		}
		if len(tags) > 0 {
			var tagNames []string
			for _, tag := range tags {
				tagNames = append(tagNames, fmt.Sprintf("%q:%s", tag.Name, tag.Type))
			}
			fmt.Fprintf(&buf, "tags=%s\n", strings.Join(tagNames, ","))
		}
	}

	if ag.opts.Editors {
		editors, err := ag.base.ProblemEditors(ctx, ag.pb.ID)
		if err != nil {
			return err
		}
		if len(editors) > 0 {
			var editorNames []string
			for _, editor := range editors {
				editorNames = append(editorNames, editor.Name)
			}
			fmt.Fprintf(&buf, "editors=%s\n", strings.Join(editorNames, ","))
		}
	}

	if buf.Len() == 0 {
		return nil // Do not create grader.properties when empty
	}

	gr, err := ag.ar.Create("grader.properties")
	if err != nil {
		return kilonova.WrapError(err, "Couldn't create archive grader.properties file")
	}
	if _, err := io.Copy(gr, &buf); err != nil {
		return kilonova.WrapError(err, "Couldn't write grader.properties file")
	}

	return nil
}

func (ag *archiveGenerator) addSubmissions(ctx context.Context) *kilonova.StatusError {
	filter := kilonova.SubmissionFilter{ProblemID: &ag.pb.ID}
	if !ag.opts.AllSubmissions {
		filter.FromAuthors = true
	}
	if ag.opts.SubsLook {
		filter.Look = true
		filter.LookingUser = ag.opts.SubsLookingUser
	}
	subs, err := ag.base.RawSubmissions(ctx, filter)
	if err != nil {
		return err
	}
	for _, sub := range subs {
		lang, ok := eval.Langs[sub.Language]
		if !ok || lang.Disabled {
			zap.S().Infof("Skipping submission due to unknown/disabled language (%q): %d", sub.Language, sub.ID)
			continue
		}
		f, err := ag.ar.Create(fmt.Sprintf("submissions/%d-%sp%s", sub.ID, sub.Score.String(), lang.Extensions[len(lang.Extensions)-1]))
		if err != nil {
			return kilonova.WrapError(err, "Couldn't create archive submission file")
		}
		code, err1 := ag.base.RawSubmissionCode(ctx, sub.ID)
		if err1 != nil {
			return kilonova.WrapError(err1, "Couldn't get submission code")
		}
		n, err := f.Write(code)
		if err != nil || n < len(code) {
			return kilonova.WrapError(err, "Couldn't write submission file")
		}
	}
	return nil
}

func GenerateArchive(ctx context.Context, pb *kilonova.Problem, w io.Writer, base *sudoapi.BaseAPI, opts *ArchiveGenOptions) *kilonova.StatusError {
	ag := &archiveGenerator{
		pb:   pb,
		ar:   zip.NewWriter(w),
		base: base,
		opts: opts,
	}
	defer ag.ar.Close()

	testName := strings.TrimSpace(ag.pb.TestName)
	if testName == "" {
		testName = kilonova.MakeSlug(testName)
	}
	ag.testName = testName

	// tests
	if opts.Tests {
		if err := ag.addTests(ctx); err != nil {
			return err
		}
	}

	// attachments/
	if opts.Attachments {
		if err := ag.addAttachments(ctx); err != nil {
			return err
		}
	}

	// grader.properties
	if opts.Tests || opts.ProblemDetails || opts.Tags || opts.Editors {
		if err := ag.addGraderProperties(ctx); err != nil {
			return err
		}
	}

	// submissions/
	if opts.Submissions {
		if err := ag.addSubmissions(ctx); err != nil {
			return err
		}
	}

	return nil
}
