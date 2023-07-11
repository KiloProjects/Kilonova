package test

import (
	"io"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/antchfx/xmlquery"
)

func ProcessProblemXMLFile(actx *ArchiveCtx, file io.Reader) *kilonova.StatusError {
	node, err := xmlquery.Parse(file)
	if err != nil {
		return kilonova.WrapError(err, "Could not read problem.xml")
	}

	if actx.props == nil {
		actx.props = &properties{}
	}

	nodes := xmlquery.Find(node, "//judging/testset")

	var setNode *xmlquery.Node
	for _, node := range nodes {
		node := node
		if node.SelectAttr("name") == "tests" {
			setNode = node
			break
		}
	}

	if setNode == nil {
		return kilonova.Statusf(400, "There must be a `tests` testset")
	}

	if node := xmlquery.FindOne(setNode, "//time-limit"); node != nil {
		timeLimit, err := strconv.Atoi(node.Data)
		if err == nil {
			timeLimitF := float64(timeLimit) / 1000.0
			actx.props.TimeLimit = &timeLimitF
		}
	}

	if node := xmlquery.FindOne(setNode, "//memory-limit"); node != nil {
		memoryLimit, err := strconv.Atoi(node.Data)
		if err == nil {
			memoryLimit /= 1024
			actx.props.MemoryLimit = &memoryLimit
		}
	}

	return nil
}
