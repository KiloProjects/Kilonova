package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/KiloProjects/Kilonova/internal/manager"
	"github.com/KiloProjects/Kilonova/internal/proto"
	"github.com/davecgh/go-spew/spew"
)

// Handle manages the connection to the platform
func Handle(send chan<- proto.Message, recv <-chan proto.Message) error {
	mgr, err := manager.New(2)
	if err != nil {
		return err
	}
	mgr.ToggleDebug()

	log.Println("Connection accepted")

	for msg := range recv {
		switch msg.Type {
		case "Compile":
			var args proto.Compile
			proto.DecodeArgs(msg, &args)

			spew.Dump(args)
			resp := mgr.CompileTask(args)
			spew.Dump(resp)
			if resp != nil {
				send <- proto.ArgToMessage(resp)
			}
		case "STask":
			var args proto.STask
			proto.DecodeArgs(msg, &args)

			spew.Dump(args)
			resp, err := mgr.ExecuteSTask(args)
			spew.Dump(resp, err)
			if err != nil {
				send <- proto.WrapErr(err)
				continue
			}

			send <- proto.ArgToMessage(resp)
			// send <- proto.ErrMessage("Not Yet Implemented")
		case "TRemove":
			var args proto.TRemove
			proto.DecodeArgs(msg, &args)
			p := path.Join(*compilePath, fmt.Sprintf("%d.bin", args.ID))

			spew.Dump(args)
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

	log.Println("Closing connection")

	return nil
}
