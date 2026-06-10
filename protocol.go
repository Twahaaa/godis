package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/tidwall/resp"
)

const (
	CommandSet    string = "SET"
	CommandGet    string = "GET"
	CommandDel    string = "DEL"
	CommandExists string = "EXISTS"
	CommandKeys   string = "KEYS"
)

type Command interface {
}

type SetCommand struct {
	key, val, ttl []byte
}

type GetCommand struct {
	key []byte
}

type DelCommand struct {
	key []byte
}
type ExistsCommand struct {
	key []byte
}
type KeysCommand struct {
}

func parseTTL(ttl_byte []byte) (time.Duration ,error){
	secs, err := strconv.Atoi(string(ttl_byte))

	if err!=nil{
		return 0, err
	}

	ttl := time.Duration(secs) * time.Second

	return ttl, nil
}

func parseCommand(msg resp.Value) (Command, error) {
	if msg.Type() == resp.Array {
		for _, value := range msg.Array() {
			switch value.String() {
			case CommandSet:
				if len(msg.Array()) != 3 && len(msg.Array()) != 4 {
					return nil, fmt.Errorf("invalid number of variables for SET command")
				}
				cmd := SetCommand{
					key: msg.Array()[1].Bytes(),
					val: msg.Array()[2].Bytes(),
				}

				if len(msg.Array()) == 4 {
					cmd.ttl = msg.Array()[3].Bytes()
				}

				return cmd, nil

			case CommandGet:
				if len(msg.Array()) != 2 {
					return nil, fmt.Errorf("invalid number of vairables for the GET command")
				}
				return GetCommand{
					key: msg.Array()[1].Bytes(),
				}, nil

			case CommandDel:
				if len(msg.Array()) != 2 {
					return nil, fmt.Errorf("invalid number of varaibles for DEL command")
				}
				return DelCommand{
					key: msg.Array()[1].Bytes(),
				}, nil

			case CommandExists:
				if len(msg.Array()) != 2 {
					return nil, fmt.Errorf("invalid number of varibles for EXISTS command")
				}
				return ExistsCommand{
					key: msg.Array()[1].Bytes(),
				}, nil

			case CommandKeys:
				if len(msg.Array()) != 2 {
					return nil, fmt.Errorf("invalid number of variables for the Keys command")
				}
				return KeysCommand{}, nil

			}
		}
	}
	return nil, fmt.Errorf("invalid or unknown command received: %s", msg)
}
