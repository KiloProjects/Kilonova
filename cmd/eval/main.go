package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/KiloProjects/Kilonova/internal/box"
	"github.com/KiloProjects/Kilonova/internal/manager"
	"github.com/KiloProjects/Kilonova/internal/proto"
)

var (
	isolateBin  = flag.String("isolatePath", "/tmp/isolate", "The path to the isolate binary (if it does not exist, it will be created there")
	compilePath = flag.String("compilePath", "/tmp/kncompiles", "The path to a directory to store the resulting executable in")
	socketPath  = flag.String("socketPath", "/tmp/kiloeval.sock", "The path to the socket to listen on")
)

// Serve accepts connections akin to http.Serve
func Serve(l net.Listener, handler proto.Handler) error {
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
			if err := proto.Handle(conn, handler); err != nil {
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
		if err := Serve(l, Handle); err != nil {
			panic(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)
	<-sig

}
