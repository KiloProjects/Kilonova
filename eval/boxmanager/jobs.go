package boxmanager

import (
	"context"

	"github.com/KiloProjects/kilonova/eval"
)

var _ eval.Job = &CompileJob{}
var _ eval.Job = &ExecuteJob{}

type CompileJob struct {
	Req  *eval.CompileRequest
	Resp *eval.CompileResponse
}

func (job *CompileJob) Execute(ctx context.Context, box eval.Sandbox) error {
	panic("TODO")
}

type ExecuteJob struct {
	Req  *eval.ExecRequest
	Resp *eval.ExecResponse
}

func (job *ExecuteJob) Execute(ctx context.Context, box eval.Sandbox) error {
	panic("TODO")
}
