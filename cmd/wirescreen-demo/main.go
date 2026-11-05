package main

import (
	"fmt"
	"os"

	"api/internal/ui/wirescreen"

	tea "charm.land/bubbletea/v2"
)

func main() {
	url := "ws://localhost:8080/ws"
	if len(os.Args) > 1 {
		url = os.Args[1]
	}

	m := wirescreen.New(nil, url)
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running wirescreen demo: %v\n", err)
		os.Exit(1)
	}
}
