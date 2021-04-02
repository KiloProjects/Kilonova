package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/boxmanager"
	"github.com/KiloProjects/kilonova/eval/tasks"
	pb "github.com/KiloProjects/kilonova/grpc"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type evalServer struct {
	mgr *boxmanager.BoxManager
	dm  kilonova.DataStore
	pb.UnimplementedEvalServer
}

func (s *evalServer) Compile(ctx context.Context, req *pb.CompileRequest) (*pb.CompileResponse, error) {
	job := &tasks.CompileTask{
		Req: &eval.CompileRequest{
			ID:   int(req.ID),
			Code: []byte(req.Code),
			Lang: req.Lang,
		},
	}
	if err := s.mgr.RunTask(ctx, job); err != nil {
		return &pb.CompileResponse{}, err
	}
	return &pb.CompileResponse{
		Output:  job.Resp.Output,
		Success: job.Resp.Success,
		Other:   job.Resp.Other,
	}, nil
}

func (s *evalServer) Execute(ctx context.Context, test *pb.Test) (*pb.TestResponse, error) {
	job := &tasks.ExecuteTask{
		Req: &eval.ExecRequest{
			SubID:       int(test.ID),
			SubtestID:   int(test.TID),
			TestID:      int(test.TestID),
			Filename:    test.Filename,
			StackLimit:  int(test.StackLimit),
			MemoryLimit: int(test.MemoryLimit),
			TimeLimit:   test.TimeLimit,
			Lang:        test.Lang,
		},
		Resp: &eval.ExecResponse{},
	}
	if err := s.mgr.RunTask(ctx, job); err != nil {
		return &pb.TestResponse{}, err
	}
	return &pb.TestResponse{
		Time:     job.Resp.Time,
		Memory:   int32(job.Resp.Memory),
		Status:   int32(job.Resp.ExitStatus),
		Comments: job.Resp.Comments,
	}, nil
}

func (s *evalServer) Clean(ctx context.Context, clean *pb.CleanArgs) (*pb.CleanResp, error) {
	err := eval.CleanCompilation(int(clean.ID))
	return &pb.CleanResp{Success: err == nil}, nil
}

func newEvalServer() (*evalServer, error) {
	dmgr, err := datastore.NewManager(config.Common.DataDir)
	if err != nil {
		return nil, err
	}

	mgr, err := boxmanager.New(config.Eval.NumConcurrent, dmgr)
	if err != nil {
		return nil, err
	}

	if config.Common.Debug {
		mgr.ToggleDebug()
	}

	return &evalServer{mgr: mgr, dm: dmgr}, nil
}

func Eval(_ *cli.Context) error {
	if err := os.MkdirAll(config.Eval.CompilePath, 0777); err != nil {
		return err
	}

	if !boxmanager.CheckCanRun() {
		return errors.New("Can't run without good permissions")
	}

	lis, err := net.Listen("tcp", config.Eval.Address)
	if err != nil {
		return err
	}

	sv, err := newEvalServer()
	if err != nil {
		return err
	}

	server := grpc.NewServer()
	pb.RegisterEvalServer(server, sv)
	reflection.Register(server)

	log.Println("Warmed up")

	return server.Serve(lis)
}
