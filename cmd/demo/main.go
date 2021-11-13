package main

import (
	"log"
	"strings"

	"github.com/petermattis/prompt"
)

func main() {
	inputFinished := func(text string) bool {
		text = strings.TrimSpace(text)
		return strings.HasSuffix(text, ";")
	}

	p := prompt.New(prompt.WithInputFinished(inputFinished))
	for {
		_, err := p.ReadLine("> ")
		if err != nil {
			log.Fatal(err)
		}
	}
}
