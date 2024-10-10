package test

import (
	"archive/zip"
	"bufio"
	"io"
	"log/slog"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/gorilla/schema"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Subtask struct {
	Score decimal.Decimal
	Tests []int
}

type mockTag struct {
	Name string
	Type kilonova.TagType
}

type PropertiesRaw struct {
	Groups       string   `props:"groups"`
	Weights      string   `props:"weights"`
	Dependencies string   `props:"dependencies"`
	Time         *float64 `props:"time"`
	Memory       *float64 `props:"memory"`
	Tags         *string  `props:"tags"`
	Source       *string  `props:"source"`
	ConsoleInput *string  `props:"console_input"`
	TestName     *string  `props:"test_name"`
	ProblemName  *string  `props:"problem_name"`

	Editors *string `props:"editors"`

	DefaultScore *string `props:"default_score"`

	ScorePrecision  *int32  `props:"score_precision"`
	ScoringStrategy *string `props:"scoring_strategy"`
}

func ParsePropertiesFile(r io.Reader) (*PropertiesRaw, bool, error) {
	vals := map[string][]string{}
	buf := bufio.NewScanner(r)
	for buf.Scan() {
		line := strings.TrimSpace(buf.Text())
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		key, val, found := strings.Cut(line, "=")
		if !found {
			return nil, false, nil
		}
		vals[strings.TrimSpace(key)] = []string{strings.TrimSpace(val)}
	}
	if buf.Err() != nil {
		return nil, false, buf.Err()
	}

	dec := schema.NewDecoder()
	dec.SetAliasTag("props")

	rawProps := PropertiesRaw{}
	if err := dec.Decode(&rawProps, vals); err != nil {
		return nil, false, err
	}

	return &rawProps, true, nil
}

// item is the item that we wish to split, field is for error reporting purposes
func parsePropListItem(item string, field string) ([]int, *kilonova.StatusError) {
	glist := []int{}
	gg := strings.Split(item, ";")
	for _, g := range gg {
		vals := strings.Split(g, "-")
		if len(vals) > 2 {
			return nil, kilonova.Statusf(400, "Invalid %q string in properties, too many dashes", field)
		} else if len(vals) == 2 {
			start, err := strconv.Atoi(vals[0])
			if err != nil {
				zap.S().Warn(err)
				return nil, kilonova.Statusf(400, "Invalid %q string in properties, expected int", field)
			}
			end, err := strconv.Atoi(vals[1])
			if err != nil {
				zap.S().Warn(err)
				return nil, kilonova.Statusf(400, "Invalid %q string in properties, expected int", field)
			}
			for i := start; i <= end; i++ {
				glist = append(glist, i)
			}
		} else if len(vals) == 1 && len(vals[0]) > 0 {
			val, err := strconv.Atoi(vals[0])
			if err != nil {
				zap.S().Warn(err)
				return nil, kilonova.Statusf(400, "Invalid %q string in properties, expected int", field)
			}
			glist = append(glist, val)
		}
	}
	return glist, nil
}

var (
	simpleTagRegex  = regexp.MustCompile(`^"(.*)"$`)
	complexTagRegex = regexp.MustCompile(`^"(.*)":(.*)$`)
)

func parseTags(t *string) []*mockTag {
	if t == nil {
		return nil
	}
	tagVals := strings.Split(*t, ",")
	var rez []*mockTag
	for _, val := range tagVals {
		if sm := simpleTagRegex.FindStringSubmatch(val); len(sm) > 0 {
			if len(sm[1]) > 0 {
				rez = append(rez, &mockTag{Name: sm[1], Type: kilonova.TagTypeOther})
				continue
			}
		} else if sm := complexTagRegex.FindStringSubmatch(val); len(sm) > 0 {
			if len(sm[1]) > 0 {
				mt := &mockTag{Name: sm[1], Type: kilonova.TagTypeOther}
				if kilonova.ValidTagType(kilonova.TagType(sm[2])) {
					mt.Type = kilonova.TagType(sm[2])
				}
				rez = append(rez, mt)
				continue
			}
		}
		rez = append(rez, &mockTag{Name: val, Type: kilonova.TagTypeOther})
	}
	return rez
}

func parseEditors(editors *string) []string {
	if editors == nil || *editors == "" {
		return nil
	}
	return strings.Split(*editors, ",")
}

// TODO: Allow having multiple .properties files
// And split grader.properties from generator.go into multiple files
func ProcessPropertiesFile(ctx *ArchiveCtx, file *zip.File) *kilonova.StatusError {
	f, err := file.Open()
	if err != nil {
		return kilonova.WrapError(err, "Couldn't open file")
	}
	defer f.Close()
	rawProps, ok, err := ParsePropertiesFile(f)
	if err != nil {
		return kilonova.WrapError(err, "Couldn't parse properties file")
	}
	if !ok {
		return kilonova.Statusf(400, "Invalid properties file")
	}

	props := &properties{
		TimeLimit:      rawProps.Time,
		Tags:           parseTags(rawProps.Tags),
		Editors:        parseEditors(rawProps.Editors),
		Source:         rawProps.Source,
		TestName:       rawProps.TestName,
		ProblemName:    rawProps.ProblemName,
		ScorePrecision: rawProps.ScorePrecision,
	}
	if rawProps.DefaultScore != nil {
		val, err := decimal.NewFromString(*rawProps.DefaultScore)
		if err != nil {
			return kilonova.Statusf(400, "Invalid status score value: %#v", err)
		}
		props.DefaultPoints = &val
	}
	if rawProps.Memory != nil {
		mem := int((*rawProps.Memory) * 1024.0)
		props.MemoryLimit = &mem
		if *props.MemoryLimit > config.Common.TestMaxMemKB {
			return kilonova.Statusf(400, "Maximum memory must not exceed %f MB", float64(config.Common.TestMaxMemKB)/1024.0)
		}
	}
	if rawProps.ScoringStrategy != nil && (*rawProps.ScoringStrategy == string(kilonova.ScoringTypeMaxSub) || *rawProps.ScoringStrategy == string(kilonova.ScoringTypeSumSubtasks) || *rawProps.ScoringStrategy == string(kilonova.ScoringTypeICPC)) {
		props.ScoringStrategy = kilonova.ScoringType(*rawProps.ScoringStrategy)
	}
	if rawProps.ConsoleInput != nil && (*rawProps.ConsoleInput == "true" || *rawProps.ConsoleInput == "false") {
		val := *rawProps.ConsoleInput == "true"
		props.ConsoleInput = &val
	}

	// handle subtasks
	if rawProps.Groups != "" {
		// if using score parameters, groups/weights data is redundant
		if len(ctx.scoreParameters) > 0 {
			return kilonova.Statusf(400, "Properties file cannot contain group parameters if you specified CMS-style score parameters")
		}

		stks := make(map[string]parsedSubtask)

		groupStrings := strings.Split(rawProps.Groups, ",")
		weightStrings := strings.Split(rawProps.Weights, ",")
		if len(groupStrings) != len(weightStrings) {
			return kilonova.Statusf(400, "Number of weights must match number of groups")
		}

		for i := range len(groupStrings) {
			glist, err := parsePropListItem(groupStrings[i], "group")
			if err != nil {
				return err
			}
			val, err1 := decimal.NewFromString(weightStrings[i])
			if err1 != nil {
				return kilonova.Statusf(400, "Invalid `weight` string in properties")
			}
			stk := stks[strconv.Itoa(i+1)]
			stk.Tests = glist
			stk.Score = val
			stks[strconv.Itoa(i+1)] = stk
		}

		if rawProps.Dependencies != "" {
			depStrings := strings.Split(rawProps.Dependencies, ",")
			if len(depStrings) != len(weightStrings) {
				return kilonova.Statusf(400, "Number of dependencies must match number of groups")
			}

			for i, d := range depStrings {
				if d == "" {
					continue
				}
				glist, err := parsePropListItem(d, "dependencies")
				if err != nil {
					return err
				}

				stk := stks[strconv.Itoa(i+1)]
				depStr := make([]string, 0, len(glist))
				for _, vid := range glist {
					if vid <= 0 || vid > len(groupStrings) {
						return kilonova.Statusf(400, "Dependency number out of range")
					}
					depStr = append(depStr, strconv.Itoa(vid))
				}
				stk.Dependencies = depStr
				stks[strconv.Itoa(i+1)] = stk
			}
		}

		props.Subtasks, props.SubtaskedTests = solveSubtaskDependencies(stks)
	}

	ctx.props = props
	return nil
}

type parsedSubtask struct {
	Score decimal.Decimal
	Tests []int

	// The current subtask is automatically considered a dependency
	Dependencies []string
}

func solveSubtaskDependencies(subtasks map[string]parsedSubtask) (stks map[int]Subtask, groupedTests []int) {
	stks = make(map[int]Subtask)
	subtaskedTests := make(map[int]bool)

	finalSubtasks := make(map[string]Subtask)

	for id, group := range subtasks {
		stk := Subtask{Score: group.Score}
		stk.Tests = slices.Clone(group.Tests)
		for _, dependency := range group.Dependencies {
			dep, ok := subtasks[dependency]
			if !ok {
				slog.Debug("Skipping unknown subtask", slog.String("dependency", dependency))
				continue
			}
			stk.Tests = append(stk.Tests, dep.Tests...)
		}
		slices.Sort(stk.Tests)
		stk.Tests = slices.Compact(stk.Tests)
		for _, test := range stk.Tests {
			subtaskedTests[test] = true
		}

		finalSubtasks[id] = stk
	}

	var allInts = true
	for id := range finalSubtasks {
		if _, err := strconv.Atoi(id); err != nil {
			allInts = false
			break
		}
	}
	if allInts {
		for sid, stk := range finalSubtasks {
			id, _ := strconv.Atoi(sid) // Safe to ignore, proven to be ok
			stks[id] = stk
		}
	} else {
		i := 1
		for _, stk := range finalSubtasks {
			stks[i] = stk
			i++
		}
	}

	groupedTests = make([]int, 0, len(subtaskedTests))
	for k := range subtaskedTests {
		groupedTests = append(groupedTests, k)
	}
	return
}
