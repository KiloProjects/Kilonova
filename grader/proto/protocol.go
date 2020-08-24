// Package proto implements a simple protocol for communicating in messages
// It is used by the Kilonova project (github.com/KiloProjects/Kilonova) to interact with the Grader bilaterally
// It is by no means perfect, I will keep adjusting it as time goes on
package proto

// This file should be kept on par with the one in Kilonova

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"sync"
)

// Message represents a simple message, all messages are categorized by a Type argument and a list of arguments
type Message struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}

// Handler is the function called by Handle()
type Handler func(send chan<- Message, recv <-chan Message) error

// Handle is a handler for receiving and sending messages
// It calls the handler parameter with sender and receiver channels
// Please note that it is a blocking call
func Handle(conn net.Conn, handler Handler) error {
	// we want to create a waitgroup to avoid leaving the handler while a message is being written
	// it is expected that the caller might close the connection
	var wg sync.WaitGroup
	sendChan := make(chan Message, 4)
	recvChan := make(chan Message, 1)
	go func() { // message receiving goroutine
		dec := json.NewDecoder(conn)
		for {
			var msg Message
			if err := dec.Decode(&msg); err != nil {
				close(recvChan)
				if errors.Is(err, io.EOF) {
					// Connection closed
				} else {
					log.Printf("%v\n", err)
				}
				/* Most of the time it's a read after connection close, they don't really matter
				else {
					log.Printf("Decoding error: %v\n", err)
				}
				*/
				return
			}
			recvChan <- msg
		}
	}()
	wg.Add(1)
	go func() { // message sending goroutine
		defer wg.Done()
		enc := json.NewEncoder(conn)
		for msg := range sendChan {
			if err := enc.Encode(msg); err != nil {
				log.Printf("Encoding error: %v\n", err)
			}
		}
	}()
	return func() error {
		err := handler(sendChan, recvChan)
		close(sendChan)
		if err != nil {
			return err
		}
		wg.Wait()
		return err
	}()
}
