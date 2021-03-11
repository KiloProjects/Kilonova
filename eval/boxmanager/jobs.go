package boxmanager

import (
	"context"

	"github.com/KiloProjects/kilonova"
)

var _ kilonova.Job = &CompileJob{}
var _ kilonova.Job = &ExecuteJob{}

type CompileJob struct {
	Req  *kilonova.CompileRequest
	Resp *kilonova.CompileResponse
}

func (job *CompileJob) Execute(ctx context.Context, box kilonova.Sandbox) error {
	panic("TODO")
}

type ExecuteJob struct {
	Req  *kilonova.ExecRequest
	Resp *kilonova.ExecResponse
}

func (job *ExecuteJob) Execute(ctx context.Context, box kilonova.Sandbox) error {
	panic("TODO")
}
