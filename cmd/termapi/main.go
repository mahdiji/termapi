package main

import (
	"api/internal/model"
	"api/internal/ui"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func main() {
	zone.NewGlobal()

	p := tea.NewProgram(ui.NewAppModel(model.NewSharedState()))
	if _, err := p.Run(); err != nil {
		fmt.Println("aplikasi gagal jalan")
		os.Exit(0)
	}
}
