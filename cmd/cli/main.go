package main

import (
	"log"

	"github.com/Twahaaa/godis/tui"
)

func main() {
	if err := tui.Start("localhost:5001"); err != nil {
		log.Fatal(err)
	}
}
