package game

import (
	"encoding/json"
	"incell/internal/assets"
	"os"
	"path/filepath"
)

// GameState represents the serializable game state
type GameState struct {
	FreeCells   [NumFreeCells]*CardData   `json:"free_cells"`
	Foundations [NumFoundation][]CardData `json:"foundations"`
	Tableau     [NumTableau][]CardData    `json:"tableau"`
}

// CardData is a JSON-serializable representation of a card
type CardData struct {
	Rank int `json:"rank"`
	Suit int `json:"suit"`
}

func cardToData(c *Card) *CardData {
	if c == nil {
		return nil
	}
	return &CardData{Rank: int(c.Rank), Suit: int(c.Suit)}
}

func dataToCard(d *CardData) *Card {
	if d == nil {
		return nil
	}
	return &Card{Rank: assets.Rank(d.Rank), Suit: assets.Suit(d.Suit)}
}

func cardsToData(cards []Card) []CardData {
	data := make([]CardData, len(cards))
	for i, c := range cards {
		data[i] = CardData{Rank: int(c.Rank), Suit: int(c.Suit)}
	}
	return data
}

func dataToCards(data []CardData) []Card {
	cards := make([]Card, len(data))
	for i, d := range data {
		cards[i] = Card{Rank: assets.Rank(d.Rank), Suit: assets.Suit(d.Suit)}
	}
	return cards
}

// statePath returns the path to the saved game state file
func statePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "incell", "state.json"), nil
}

// SaveState saves the current game state to disk
func (g *FreeCell) SaveState() error {
	path, err := statePath()
	if err != nil {
		return err
	}

	state := GameState{}

	for i, fc := range g.FreeCells {
		state.FreeCells[i] = cardToData(fc)
	}

	for i, foundation := range g.Foundations {
		state.Foundations[i] = cardsToData(foundation)
	}

	for i, tableau := range g.Tableau {
		state.Tableau[i] = cardsToData(tableau)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadState loads a saved game state from disk
func (g *FreeCell) LoadState() error {
	path, err := statePath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var state GameState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	for i, fc := range state.FreeCells {
		g.FreeCells[i] = dataToCard(fc)
	}

	for i, foundation := range state.Foundations {
		g.Foundations[i] = dataToCards(foundation)
	}

	for i, tableau := range state.Tableau {
		g.Tableau[i] = dataToCards(tableau)
	}

	// Clear history and selection when loading
	g.History = nil
	g.Selected = nil

	return nil
}

// HasSavedState returns true if there's a saved game state
func HasSavedState() bool {
	path, err := statePath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// DeleteSavedState removes the saved game state
func DeleteSavedState() error {
	path, err := statePath()
	if err != nil {
		return err
	}
	return os.Remove(path)
}
