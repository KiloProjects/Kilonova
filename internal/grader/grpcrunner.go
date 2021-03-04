package grader

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KiloProjects/kilonova"
	pb "github.com/KiloProjects/kilonova/grpc"
	"google.golang.org/grpc"
)

var _ kilonova.Runner = &grpcRunner{}

type grpcRunner struct {
	conn   *grpc.ClientConn
	client pb.EvalClient
}

func (g *grpcRunner) Compile(ctx context.Context, cr *kilonova.CompileRequest) (*kilonova.CompileResponse, error) {
	resp, err := g.client.Compile(ctx, &pb.CompileRequest{ID: int32(cr.ID), Code: string(cr.Code), Lang: cr.Lang})
	return &kilonova.CompileResponse{
		Output:  resp.Output,
		Success: resp.Success,
		Other:   resp.Other,
	}, err
}

func (g *grpcRunner) Execute(ctx context.Context, er *kilonova.ExecRequest) (*kilonova.ExecResponse, error) {
	resp, err := g.client.Execute(ctx, &pb.Test{
		ID:          int32(er.SubID),
		TID:         int32(er.SubtestID),
		TestID:      int64(er.TestID),
		Filename:    er.Filename,
		StackLimit:  int32(er.StackLimit),
		MemoryLimit: int32(er.MemoryLimit),
		TimeLimit:   er.TimeLimit,
		Lang:        er.Lang,
	})
	return &kilonova.ExecResponse{
		SubtestID:  int(resp.TID),
		Time:       resp.Time,
		Memory:     int(resp.Memory),
		ExitStatus: int(resp.Status),
		Comments:   resp.Comments,
	}, err
}

func (g *grpcRunner) Clean(ctx context.Context, subid int) error {
	_, err := g.client.Clean(ctx, &pb.CleanArgs{ID: int32(subid)})
	return err
}

func (g *grpcRunner) Close(_ context.Context) error {
	return g.conn.Close()
}

func newGrpcRunner(address string) (*grpcRunner, error) {
	// Dial here to pre-emptively exit in case it fails
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		if err == context.DeadlineExceeded {
			return nil, errors.New("WARNING: No grader found, will not grade submissions")
		}
		return nil, fmt.Errorf("Dialing error: %w", err)
	}

	client := pb.NewEvalClient(conn)
	return &grpcRunner{conn, client}, nil
}
