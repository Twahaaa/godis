package main

import (
	"fmt"
	"testing"
)

func TestProtocol(t *testing.T){
	msg := "*3\r\n$3\r\nSET\r\n$5\r\nmykey\r\n$3\r\nbar\r\n"
	cmd, err := parseCommand(msg)
	if err!=nil{
		t.Fatal(err)
	}
	fmt.Println(cmd)
}