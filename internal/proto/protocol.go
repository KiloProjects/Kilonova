// Package proto implements a simple protocol for communicating in messages
// It is by no means perfect, I will keep adjusting it as time goes on
package proto

// This file should be kept on par with the one in Kilonova

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
)

// Message represents a simple message, all messages are categorized by a Type argument and a list of arguments
type Message struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}

// Handler is the function called by Handle()
type Handler func(ctx context.Context, send chan<- Message, recv <-chan Message) error

// Handle is a handler for receiving and sending messages
// It calls the handler parameter with sender and receiver channels
// Please note that it is a blocking call
func Handle(ctx context.Context, conn net.Conn, handler Handler) error {
	// we want to create a waitgroup to avoid leaving the handler while a message is being written
	// it is expected that the caller might close the connection
	sendChan := make(chan Message, 4)
	recvChan := make(chan Message, 1)
	finished := make(chan bool, 1)
	go func() { // message receiving goroutine
		dec := json.NewDecoder(conn)
		for {
			var msg Message
			if err := dec.Decode(&msg); err != nil {
				close(recvChan)
				if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				} else {
					log.Printf("%v\n", err)
				}
				return
			}
			recvChan <- msg
		}
	}()
	go func() { // message sending goroutine
		enc := json.NewEncoder(conn)
		defer func() { close(sendChan) }()
		for {
			select {
			case msg, more := <-sendChan:
				if !more {
					return
				}
				if err := enc.Encode(msg); err != nil {
					log.Printf("Encoding error: %v\n", err)
				}
			case <-ctx.Done():
				return
			case <-finished:
				return
			}
		}
	}()

	err := handler(ctx, sendChan, recvChan)
	finished <- true
	return err
}
