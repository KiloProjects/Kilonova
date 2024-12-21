package test

import (
	"fmt"
	"io"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/antchfx/xmlquery"
	"github.com/shopspring/decimal"
)

func ProcessProblemXMLFile(actx *ArchiveCtx, file io.Reader) error {
	node, err := xmlquery.Parse(file)
	if err != nil {
		return fmt.Errorf("Could not read problem.xml: %w", err)
	}

	if actx.props == nil {
		actx.props = &properties{}
	}

	// Select problem name
	for _, node := range xmlquery.Find(node, "//names/name") {
		if lang := node.SelectAttr("language"); lang == "english" || lang == "romanian" {
			val := node.SelectAttr("value")
			actx.props.ProblemName = &val
		}
	}

	// Get main testset
	// Kilonova doesn't support more than one testset
	var testsetNode *xmlquery.Node
	for _, node := range xmlquery.Find(node, "//judging/testset") {
		node := node
		if node.SelectAttr("name") == "tests" {
			testsetNode = node
			break
		}
	}

	if testsetNode == nil {
		return kilonova.Statusf(400, "There must be a `tests` testset")
	}

	// Get task points and subtask
	var isICPC bool = true
	var subtasks = make(map[string]parsedSubtask)
	for id, test := range xmlquery.Find(testsetNode, "//tests/test") {
		if points := test.SelectAttr("points"); points != "" {
			isICPC = false
			val, err := decimal.NewFromString(points)
			if err == nil {
				actx.testScores[id+1] = val
			}
		}
		if group := test.SelectAttr("group"); group != "" {
			isICPC = false
			stk, ok := subtasks[group]
			if !ok {
				stk = parsedSubtask{}
			}
			stk.Tests = append(stk.Tests, id+1)
			subtasks[group] = stk
		}
	}
	if isICPC && actx.params.FirstImport && len(actx.params.ScoreParamsStr) == 0 {
		actx.props.ScoringStrategy = kilonova.ScoringTypeICPC
	}

	if len(subtasks) > 0 {
		// Parse group points and dependencies
		for i, group := range xmlquery.Find(testsetNode, "//groups/group") {
			name := group.SelectAttr("name")
			stk, ok := subtasks[name]
			if !ok {
				continue
			}
			if points := group.SelectAttr("points"); points != "" {
				val, err := decimal.NewFromString(points)
				if err == nil {
					stk.Score = val
				}
			} else {
				val := decimal.Zero
				for _, test := range stk.Tests {
					val = val.Add(actx.testScores[test])
				}
				stk.Score = val
			}

			var dependencies []string
			for _, dep := range xmlquery.Find(group, "//dependencies/dependency") {
				if val := dep.SelectAttr("name"); len(val) > 0 {
					dependencies = append(dependencies, val)
				}
			}
			if len(dependencies) > 0 {
				stk.Dependencies = dependencies
			}

			stk.Index = &i

			subtasks[name] = stk
		}

		// Solve dependencies
		actx.props.Subtasks, actx.props.SubtaskedTests = solveSubtaskDependencies(actx.ctx, subtasks)
	}

	// Parse time/memory limit
	if node := xmlquery.FindOne(testsetNode, "//time-limit"); node != nil {
		timeLimit, err := strconv.Atoi(node.InnerText())
		if err == nil {
			timeLimitF := float64(timeLimit) / 1000.0
			actx.props.TimeLimit = &timeLimitF
		}
	}

	if node := xmlquery.FindOne(testsetNode, "//memory-limit"); node != nil {
		memoryLimit, err := strconv.Atoi(node.InnerText())
		if err == nil {
			memoryLimit /= 1024
			actx.props.MemoryLimit = &memoryLimit
		}
	}

	return nil
}
