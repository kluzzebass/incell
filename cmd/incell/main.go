package main

import (
	"log"

	"incell/internal/assets"
	"incell/internal/game"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// Load card assets
	if err := assets.LoadCards(); err != nil {
		log.Fatalf("Failed to load card assets: %v", err)
	}

	// Create game UI
	ui := game.NewUI()

	// Configure window
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowTitle("Incell")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Run game
	if err := ebiten.RunGame(ui); err != nil {
		log.Fatal(err)
	}
}
