package main

import (
	"fmt"
	"github.com/tidwall/resp"
)

const (
	CommandSet  string = "SET"
	CommandGet  string = "GET"
	CommandDel  string = "DEL"
	CommandKeys string = "KEYS"
)

type Command interface {
}

type SetCommand struct {
	key, val []byte
}

type GetCommand struct {
	key []byte
}

type DelCommand struct {
	key []byte
}
type KeysCommand struct {
}

func parseCommand(msg resp.Value) (Command, error) {
	if msg.Type() == resp.Array {
		for _, value := range msg.Array() {
			switch value.String() {
			case CommandSet:
				if len(msg.Array()) != 3 {
					return nil, fmt.Errorf("invalid number of variables for SET commands")
				}
				return SetCommand{
					key: msg.Array()[1].Bytes(),
					val: msg.Array()[2].Bytes(),
				}, nil

			case CommandGet:
				if len(msg.Array()) != 2 {
					return nil, fmt.Errorf("invalid number of vairables for the GET commands")
				}
				return GetCommand{
					key: msg.Array()[1].Bytes(),
				}, nil

			case CommandDel:
				if len(msg.Array()) != 2 {
					return nil, fmt.Errorf("invalid number of varaibles for DEL commands")
				}
				return DelCommand{
					key: msg.Array()[1].Bytes(),
				}, nil

			case CommandKeys:
				if len(msg.Array()) != 2 {
					return nil, fmt.Errorf("invalid number of variables for the Keys commands")
				}
				return KeysCommand{}, nil
			}
		}
	}
	return nil, fmt.Errorf("invalid or unknown command received: %s", msg)
}
