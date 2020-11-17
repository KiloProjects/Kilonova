package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path"

	"github.com/KiloProjects/Kilonova/internal/box"
	"github.com/KiloProjects/Kilonova/internal/boxmanager"
	pb "github.com/KiloProjects/Kilonova/internal/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	isolateBin  = flag.String("isolatePath", "/tmp/isolate", "The path to the isolate binary (if it does not exist, it will be created there")
	compilePath = flag.String("compilePath", "/tmp/kncompiles", "The path to a directory to store the resulting executable in")
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

func (s *evalServer) Clean(_ context.Context, clean *pb.CleanArgs) (*pb.EmptyResponse, error) {
	p := path.Join(*compilePath, fmt.Sprintf("%d.bin", clean.ID))
	return &pb.EmptyResponse{}, os.Remove(p)
}

func newEvalServer() *evalServer {
	mgr, err := boxmanager.New(2)
	if err != nil {
		panic(err)
	}

	return &evalServer{mgr: mgr}
}

func main() {
	flag.Parse()
	if err := os.MkdirAll(*compilePath, 0777); err != nil {
		panic(err)
	}

	boxmanager.SetCompilePath(*compilePath)
	box.Initialize(*isolateBin)

	lis, err := net.Listen("tcp", "localhost:8001")
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()
	pb.RegisterEvalServer(server, newEvalServer())
	reflection.Register(server)

	fmt.Println("Warmed up")

	if err := server.Serve(lis); err != nil {
		panic(err)
	}
}
