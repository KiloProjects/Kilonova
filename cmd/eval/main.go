package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path"

	"github.com/KiloProjects/Kilonova/internal/box"
	"github.com/KiloProjects/Kilonova/internal/manager"
	"github.com/KiloProjects/Kilonova/internal/proto"
)

var (
	isolateBin  = flag.String("isolatePath", "/tmp/isolate", "The path to the isolate binary (if it does not exist, it will be created there")
	compilePath = flag.String("compilePath", "/tmp/kncompiles", "The path to a directory to store the resulting executable in")
	socketPath  = flag.String("socketPath", "/tmp/kiloeval.sock", "The path to the socket to listen on")
)

// Handle manages the connection to the platform
func Handle(ctx context.Context, send chan<- proto.Message, recv <-chan proto.Message) error {
	mgr, err := manager.New(2)
	if err != nil {
		return err
	}

	log.Println("Connection accepted")

	defer func() {
		log.Println("Closing connection")
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, more := <-recv:
			if !more {
				return nil
			}

			switch msg.Type {
			case "Compile":
				var args proto.Compile
				proto.DecodeArgs(msg, &args)

				resp := mgr.CompileSubmission(args)
				if resp != nil {
					send <- proto.ArgToMessage(resp)
				}
			case "Test":
				var args proto.Test
				proto.DecodeArgs(msg, &args)

				resp, err := mgr.ExecuteTest(args)
				if err != nil {
					send <- proto.WrapErr(err)
					continue
				}

				send <- proto.ArgToMessage(resp)
			case "TRemove":
				var args proto.TRemove
				proto.DecodeArgs(msg, &args)
				p := path.Join(*compilePath, fmt.Sprintf("%d.bin", args.ID))

				if err := os.Remove(p); err != nil {
					send <- proto.WrapErr(err)
				}
				// don't send anything on success
			/* Not yet
			case "Assign":
				send <- proto.ErrMessage("Not Implemented")
			case "QLen":
				send <- proto.ErrMessage("Not Implemented")
			*/
			default:
				send <- proto.ErrMessage("Unknown Type")
			}
		}
	}
}

// Serve accepts connections akin to http.Serve
func Serve(ctx context.Context, l net.Listener, handler proto.Handler) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			if nErr, ok := err.(*net.OpError); ok &&
				(nErr.Timeout() || nErr.Temporary()) {
				continue
			}
			if err.Error() == "use of closed network connection" {
				return err
			}
			continue
		}
		go func(conn net.Conn, handler proto.Handler) {
			defer conn.Close()
			if err := proto.Handle(ctx, conn, handler); err != nil {
				log.Printf("Handler error: %v\n", err)
			}
		}(conn, handler)
	}
}

func main() {
	// init stuff
	flag.Parse()
	if err := os.MkdirAll(*compilePath, 0777); err != nil {
		panic(err)
	}
	manager.SetCompilePath(*compilePath)
	box.Initialize(*isolateBin)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	os.RemoveAll(*socketPath)
	l, err := net.Listen("unix", *socketPath)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	// TODO: Setup something to allow only kilonova to access the socket
	if err := os.Chmod(*socketPath, 0777); err != nil {
		log.Fatal(err)
	}

	log.Println("Listening...")

	go func() {
		if err := Serve(ctx, l, Handle); err != nil {
			panic(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)
	<-sig
	cancel()

}
