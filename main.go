package main

import (
	"log"

	"github.com/charithe/mnemosyne/cmd"
)

func main() {
	if err := cmd.StartCLI("localhost:11211"); err != nil {
		log.Fatalf("ERROR: %+v\n", err)
	}
}
