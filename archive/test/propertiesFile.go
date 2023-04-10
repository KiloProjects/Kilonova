package test

import (
	"archive/zip"
	"bufio"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type PropertiesRaw struct {
	Groups       string   `props:"groups"`
	Weights      string   `props:"weights"`
	Dependencies string   `props:"dependencies"`
	Time         *float64 `props:"time"`
	Memory       *float64 `props:"memory"`
	DefaultScore *int     `props:"default_score"`
	Author       *string  `props:"author"`
	Source       *string  `props:"source"`
	ConsoleInput *string  `props:"console_input"`
	TestName     *string  `props:"test_name"`

	ScoringStrategy *string `props:"scoring_strategy"`
}

func ParsePropertiesFile(r io.Reader) (*PropertiesRaw, bool, error) {
	vals := map[string][]string{}
	buf := bufio.NewScanner(r)
	for buf.Scan() {
		line := buf.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			return nil, false, nil
		}
		vals[strings.TrimSpace(kv[0])] = []string{strings.TrimSpace(kv[1])}
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

	props := &Properties{
		TimeLimit:     rawProps.Time,
		DefaultPoints: rawProps.DefaultScore,
		Author:        rawProps.Author,
		Source:        rawProps.Source,
		TestName:      rawProps.TestName,
	}
	if rawProps.Memory != nil {
		mem := int((*rawProps.Memory) * 1024.0)
		props.MemoryLimit = &mem
		if *props.MemoryLimit > config.Common.TestMaxMemKB {
			return kilonova.Statusf(400, "Maximum memory must not exceed %f MB", float64(config.Common.TestMaxMemKB)/1024.0)
		}
	}
	if rawProps.ScoringStrategy != nil && (*rawProps.ScoringStrategy == string(kilonova.ScoringTypeMaxSub) || *rawProps.ScoringStrategy == string(kilonova.ScoringTypeSumSubtasks)) {
		props.ScoringStrategy = kilonova.ScoringType(*rawProps.ScoringStrategy)
	}
	if rawProps.ConsoleInput != nil && (*rawProps.ConsoleInput == "true" || *rawProps.ConsoleInput == "false") {
		val := *rawProps.ConsoleInput == "true"
		props.ConsoleInput = &val
	}

	// handle subtasks
	if rawProps.Groups != "" {
		subtaskedTests := map[int]bool{}

		groups := map[int][]int{}
		subTaskGroups := map[int][][]int{}

		groupStrings := strings.Split(rawProps.Groups, ",")
		for i, grp := range groupStrings {
			glist := []int{}
			gg := strings.Split(grp, ";")
			for _, g := range gg {
				// start, end := -1, -1
				vals := strings.Split(g, "-")
				if len(vals) > 2 {
					return kilonova.Statusf(400, "Invalid `group` string in properties, too many dashes")
				} else if len(vals) == 2 {
					start, err := strconv.Atoi(vals[0])
					if err != nil {
						zap.S().Warn(err)
						return kilonova.Statusf(400, "Invalid `group` string in properties, expected int")
					}
					end, err := strconv.Atoi(vals[1])
					if err != nil {
						zap.S().Warn(err)
						return kilonova.Statusf(400, "Invalid `group` string in properties, expected int")
					}
					for i := start; i <= end; i++ {
						glist = append(glist, i)
					}
				} else if len(vals) == 1 && len(vals[0]) > 0 {
					val, err := strconv.Atoi(vals[0])
					if err != nil {
						zap.S().Warn(err)
						return kilonova.Statusf(400, "Invalid `group` string in properties, expected int")
					}
					glist = append(glist, val)
				}
			}
			groups[i+1] = glist
		}

		weights := map[int]int{}
		weightStrings := strings.Split(rawProps.Weights, ",")
		if len(groupStrings) != len(weightStrings) {
			return kilonova.Statusf(400, "Number of weights must match number of groups")
		}
		for i, w := range weightStrings {
			val, err := strconv.Atoi(w)
			if err != nil {
				return kilonova.Statusf(400, "Invalid `weight` string in properties")
			}
			weights[i+1] = val
		}

		if rawProps.Dependencies != "" {
			depStrings := strings.Split(rawProps.Dependencies, ",")
			if len(depStrings) != len(weightStrings) {
				return kilonova.Statusf(400, "Number of dependencies must match number of groups")
			}

			for i, d := range depStrings {
				subTaskGroups[i+1] = [][]int{groups[i+1]}
				if d == "" {
					continue
				}
				depGroups := strings.Split(d, ";")
				for _, dg := range depGroups {
					val, err := strconv.Atoi(dg)
					if err != nil {
						return kilonova.Statusf(400, "Invalid `dependencies` string in properties: %q is not a number", dg)
					}
					if val <= 0 || val > len(groupStrings) {
						return kilonova.Statusf(400, "Dependency number out of range")
					}
					subTaskGroups[i+1] = append(subTaskGroups[i+1], groups[val])
				}
			}
		} else {
			for i := range groupStrings {
				subTaskGroups[i+1] = [][]int{groups[i+1]}
			}
		}

		// coalesce maps into a single data type
		stks := map[int]Subtask{}

		for id, groups := range subTaskGroups {
			stk := Subtask{}
			stk.Score = weights[id]

			for _, groupList := range groups {
				for _, test := range groupList {
					subtaskedTests[test] = true
					stk.Tests = append(stk.Tests, test)
				}
			}
			sort.Ints(stk.Tests)

			stks[id] = stk
		}

		tests := []int{}
		for k := range subtaskedTests {
			tests = append(tests, k)
		}

		props.SubtaskedTests = tests
		props.Subtasks = stks
	}

	ctx.props = props
	return nil
}
