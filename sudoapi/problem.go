package sudoapi

import (
	"context"
	"path"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/util"
	"go.uber.org/zap"
)

// Problem stuff

func (s *BaseAPI) Problem(ctx context.Context, id int) (*kilonova.Problem, *StatusError) {
	problem, err := s.db.Problem(ctx, id)
	if err != nil || problem == nil {
		return nil, WrapError(err, "Problem not found")
	}
	return problem, nil
}

func (s *BaseAPI) ProblemByName(ctx context.Context, name string) (*kilonova.Problem, *StatusError) {
	problem, err := s.db.ProblemByName(ctx, name)
	if err != nil || problem == nil {
		return nil, WrapError(err, "Problem not found")
	}
	return problem, nil
}

// `updater` is an optional parameter, specifying the author of the change. It handles the visibility change option.
// Visibility is the only parameter that can not be updated by a mere problem editor, it requires admin permissions!
// Please note that, if updater is not specified, the function won't attempt to ensure correct permissions for visibility.
func (s *BaseAPI) UpdateProblem(ctx context.Context, id int, args kilonova.ProblemUpdate, updater *kilonova.UserBrief) *StatusError {
	if args.Name != nil && *args.Name == "" {
		return Statusf(400, "Title can't be empty!")
	}
	if updater != nil && args.Visible != nil && !util.IsAdmin(updater) {
		return Statusf(403, "User can't update visibility!")
	}

	if err := s.db.UpdateProblem(ctx, id, args); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update problem")
	}

	return nil
}

func (s *BaseAPI) DeleteProblem(ctx context.Context, id int) *StatusError {
	if err := s.db.DeleteProblem(ctx, id); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't delete problem")
	}
	return nil
}

func (s *BaseAPI) Problems(ctx context.Context, filter kilonova.ProblemFilter) ([]*kilonova.Problem, *StatusError) {
	problems, err := s.db.Problems(ctx, filter)
	if err != nil {
		zap.S().Warn(err)
		return nil, WrapError(err, "Couldn't get problems")
	}
	return problems, nil
}

func (s *BaseAPI) SolvedProblems(ctx context.Context, uid int) ([]*kilonova.Problem, *StatusError) {
	pbs, err := s.db.SolvedProblems(ctx, uid)
	if err != nil {
		zap.S().Warn(err)
		return nil, WrapError(err, "Couldn't get solved problems")
	}
	return pbs, nil
}

func (s *BaseAPI) InsertProblem(ctx context.Context, problem *kilonova.Problem) (int, *StatusError) {
	if _, err := s.ProblemByName(ctx, problem.Name); err == nil {
		return -1, Statusf(400, "Problem with title already exists")
	}

	err := s.db.CreateProblem(ctx, problem)
	if err != nil {
		return -1, WrapError(err, "Couldn't create problem")
	}
	return problem.ID, nil
}

// CreateProblem is the simple way of creating a new problem. Just provide a title, an author and the type of input.
// The other stuff will be automatically set for sensible defaults.
func (s *BaseAPI) CreateProblem(ctx context.Context, title string, author *UserBrief, consoleInput bool) (int, *StatusError) {
	problem := &kilonova.Problem{
		Name:         title,
		AuthorID:     author.ID,
		ConsoleInput: consoleInput,
	}

	return s.InsertProblem(ctx, problem)
}

func (s *BaseAPI) GetProblemSettings(ctx context.Context, problemID int) (*kilonova.ProblemEvalSettings, error) {
	var settings = &kilonova.ProblemEvalSettings{}

	atts, err := s.ProblemAttachments(ctx, problemID)
	if err != nil {
		return nil, err
	}

	for _, att := range atts {
		filename := path.Base(att.Name)
		filename = strings.TrimSuffix(filename, path.Ext(filename))
		if filename == "checker" && eval.GetLangByFilename(att.Name) != "" {
			settings.CheckerName = att.Name
			continue
		}

		if att.Name[0] == '_' {
			continue
		}
		// If not checker and not skipped, continue searching

		if path.Ext(att.Name) == ".h" || path.Ext(att.Name) == ".hpp" {
			settings.OnlyCPP = true
			settings.HeaderFiles = append(settings.HeaderFiles, att.Name)
		}

		if eval.GetLangByFilename(att.Name) != "" {
			settings.OnlyCPP = true
			settings.GraderFiles = append(settings.GraderFiles, att.Name)
		}
	}

	return settings, nil
}
