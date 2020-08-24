package proto

import (
	"encoding/json"
	"log"
	"reflect"
)

func ArgToMessage(args interface{}) Message {
	arg, err := json.Marshal(args)
	if err != nil {
		log.Printf("Could not marshsal '%v': %s\n", args, err)
		arg = []byte{}
	}

	// Handle pointers
	t := reflect.TypeOf(args)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return Message{
		Type: t.Name(),
		Args: json.RawMessage(arg),
	}
}

func ErrMessage(str string) Message {
	return ArgToMessage(Error{Value: str})
}

func DecodeArgs(msg Message, to interface{}) error {
	return json.Unmarshal(msg.Args, to)
}

func WrapErr(err error) Message {
	return ArgToMessage(Error{Value: err.Error()})
}
