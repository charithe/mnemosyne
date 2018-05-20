package cmd

import (
	"io"
	"log"

	"github.com/chzyer/readline"
)

var completer = readline.NewPrefixCompleter(
	readline.PcItem("get"),
	readline.PcItem("set"),
	readline.PcItem("add"),
	readline.PcItem("replace"),
	readline.PcItem("increment"),
	readline.PcItem("decrement"),
	readline.PcItem("delete"),
	readline.PcItem("append"),
	readline.PcItem("prepend"),
	readline.PcItem("version"),
	readline.PcItem("stats"),
	readline.PcItem("quit"),
)

func StartCLI(nodes ...string) error {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          ">> ",
		HistoryFile:     "/tmp/mnemosyne.hist",
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})

	if err != nil {
		return err
	}

	defer rl.Close()
	log.SetOutput(rl.Stderr())
	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt || err == io.EOF {
			break
		}

		log.Printf("GOT: %s\n", line)
	}

	return nil
}
