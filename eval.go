package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path"

	"github.com/KiloProjects/Kilonova/internal/box"
	"github.com/KiloProjects/Kilonova/internal/boxmanager"
	"github.com/KiloProjects/Kilonova/internal/config"
	pb "github.com/KiloProjects/Kilonova/internal/grpc"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type evalServer struct {
	mgr *boxmanager.BoxManager
	pb.UnimplementedEvalServer
}

func (s *evalServer) Compile(_ context.Context, req *pb.CompileRequest) (*pb.CompileResponse, error) {
	return s.mgr.CompileSubmission(req)
}

func (s *evalServer) Execute(_ context.Context, test *pb.Test) (*pb.TestResponse, error) {
	return s.mgr.ExecuteTest(test)
}

func (s *evalServer) Clean(_ context.Context, clean *pb.CleanArgs) (*pb.Empty, error) {
	p := path.Join(config.C.Eval.CompilePath, fmt.Sprintf("%d.bin", clean.ID))
	return &pb.Empty{}, os.Remove(p)
}

func (s *evalServer) Ping(_ context.Context, _ *pb.Empty) (*pb.Empty, error) {
	return &pb.Empty{}, nil
}

func newEvalServer() *evalServer {
	mgr, err := boxmanager.New(2)
	if err != nil {
		panic(err)
	}

	return &evalServer{mgr: mgr}
}

func Eval(c *cli.Context) error {
	if err := os.MkdirAll(config.C.Eval.CompilePath, 0777); err != nil {
		return err
	}

	boxmanager.SetCompilePath(config.C.Eval.CompilePath)
	box.Initialize(config.C.Eval.IsolatePath)

	lis, err := net.Listen("tcp", "localhost:8001")
	if err != nil {
		return err
	}

	server := grpc.NewServer()
	pb.RegisterEvalServer(server, newEvalServer())
	reflection.Register(server)

	log.Println("Warmed up")

	return server.Serve(lis)
}
