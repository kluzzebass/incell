package game

import (
	"incell/internal/assets"
	"testing"
)

// setupGame creates a game with all 52 cards dealt to tableau columns
func setupGame() *Game {
	g := &Game{}
	deck := NewDeck()
	cardIdx := 0
	for row := 0; row < 7; row++ {
		for col := 0; col < NumTableau; col++ {
			if cardIdx >= len(deck) {
				break
			}
			if row == 6 && col >= 4 {
				continue
			}
			g.Tableau[col] = append(g.Tableau[col], deck[cardIdx])
			cardIdx++
		}
	}
	return g
}

func TestMaxMovableCardsToEmptyColumn(t *testing.T) {
	g := setupGame()

	// Move all cards from column 0 to free cells and other columns to make it empty
	g.Tableau[0] = nil

	// Move 2 cards to free cells
	g.FreeCells[0] = &g.Tableau[1][len(g.Tableau[1])-1]
	g.Tableau[1] = g.Tableau[1][:len(g.Tableau[1])-1]
	g.FreeCells[1] = &g.Tableau[2][len(g.Tableau[2])-1]
	g.Tableau[2] = g.Tableau[2][:len(g.Tableau[2])-1]

	// 2 free cells used, 2 empty, one empty column (destination)
	// Should be able to move (1+2)*(1+0) = 3 cards
	if got := g.MaxMovableCardsTo(0); got != 3 {
		t.Errorf("MaxMovableCardsTo with 2 free cells, 1 empty dest = %d, want 3", got)
	}
}

func TestMaxMovableCardsWithEmptyColumns(t *testing.T) {
	g := setupGame()

	// Empty columns 0 and 1
	g.Tableau[0] = nil
	g.Tableau[1] = nil

	// All free cells empty, 2 empty columns (including destination)
	// Moving to col 0, col 1 is also empty
	// Should be (1+4)*(1+1) = 10 cards
	if got := g.MaxMovableCardsTo(0); got != 10 {
		t.Errorf("MaxMovableCardsTo with 4 free cells, 2 empty cols = %d, want 10", got)
	}

	// Put a card back in col 1
	g.Tableau[1] = []Card{{Rank: assets.King, Suit: assets.Spade}}

	// Should be (1+4)*(1+0) = 5 cards
	if got := g.MaxMovableCardsTo(0); got != 5 {
		t.Errorf("MaxMovableCardsTo with 4 free cells, 1 empty dest = %d, want 5", got)
	}
}

func TestCanMoveToTableauRespectsCapacity(t *testing.T) {
	g := setupGame()

	// Empty column 0
	g.Tableau[0] = nil

	// Fill all free cells with cards from tableau
	for i := 0; i < 4; i++ {
		col := i + 1
		g.FreeCells[i] = &g.Tableau[col][len(g.Tableau[col])-1]
		g.Tableau[col] = g.Tableau[col][:len(g.Tableau[col])-1]
	}

	// With 0 free cells and moving to empty column, can only move 1 card
	oneCard := []Card{{Rank: assets.King, Suit: assets.Spade}}
	if !g.CanMoveToTableau(oneCard, 0) {
		t.Error("Should be able to move 1 card to empty column with 0 free cells")
	}

	twoCards := []Card{
		{Rank: assets.King, Suit: assets.Spade},
		{Rank: assets.Queen, Suit: assets.Heart},
	}
	if g.CanMoveToTableau(twoCards, 0) {
		t.Error("Should NOT be able to move 2 cards to empty column with 0 free cells")
	}
}

func TestCanMoveToTableauAlternatingColors(t *testing.T) {
	g := setupGame()

	// Set up column 0 with a red King on top
	g.Tableau[0] = []Card{{Rank: assets.King, Suit: assets.Heart}}

	// Black Queen should be able to stack
	blackQueen := []Card{{Rank: assets.Queen, Suit: assets.Spade}}
	if !g.CanMoveToTableau(blackQueen, 0) {
		t.Error("Black Queen should stack on red King")
	}

	// Red Queen should NOT stack
	redQueen := []Card{{Rank: assets.Queen, Suit: assets.Diamond}}
	if g.CanMoveToTableau(redQueen, 0) {
		t.Error("Red Queen should NOT stack on red King")
	}

	// Black King should NOT stack (wrong rank)
	blackKing := []Card{{Rank: assets.King, Suit: assets.Club}}
	if g.CanMoveToTableau(blackKing, 0) {
		t.Error("Black King should NOT stack on red King")
	}
}

func TestCanMoveToFoundation(t *testing.T) {
	g := setupGame()

	// Ace can go on empty foundation
	ace := Card{Rank: assets.Ace, Suit: assets.Heart}
	if !g.CanMoveToFoundation(ace, 0) {
		t.Error("Ace should be able to go on empty foundation")
	}

	// 2 cannot go on empty foundation
	two := Card{Rank: 2, Suit: assets.Heart}
	if g.CanMoveToFoundation(two, 0) {
		t.Error("2 should NOT go on empty foundation")
	}

	// Put Ace of hearts on foundation 0
	g.Foundations[0] = []Card{{Rank: assets.Ace, Suit: assets.Heart}}

	// 2 of hearts can now go on it
	if !g.CanMoveToFoundation(two, 0) {
		t.Error("2 of hearts should stack on Ace of hearts")
	}

	// 2 of spades cannot
	twoSpades := Card{Rank: 2, Suit: assets.Spade}
	if g.CanMoveToFoundation(twoSpades, 0) {
		t.Error("2 of spades should NOT stack on Ace of hearts")
	}

	// 3 of hearts cannot (skipping rank)
	three := Card{Rank: 3, Suit: assets.Heart}
	if g.CanMoveToFoundation(three, 0) {
		t.Error("3 of hearts should NOT stack on Ace of hearts")
	}
}

func TestIsValidTableauStack(t *testing.T) {
	// Valid stack: K(red), Q(black), J(red)
	valid := []Card{
		{Rank: assets.King, Suit: assets.Heart},
		{Rank: assets.Queen, Suit: assets.Spade},
		{Rank: assets.Jack, Suit: assets.Diamond},
	}
	if !IsValidTableauStack(valid) {
		t.Error("K-Q-J alternating colors should be valid")
	}

	// Invalid: same color
	invalid := []Card{
		{Rank: assets.King, Suit: assets.Heart},
		{Rank: assets.Queen, Suit: assets.Diamond},
	}
	if IsValidTableauStack(invalid) {
		t.Error("K-Q same color should be invalid")
	}

	// Invalid: wrong order
	wrongOrder := []Card{
		{Rank: assets.Queen, Suit: assets.Spade},
		{Rank: assets.King, Suit: assets.Heart},
	}
	if IsValidTableauStack(wrongOrder) {
		t.Error("Q-K should be invalid (wrong order)")
	}

	// Single card is always valid
	single := []Card{{Rank: 5, Suit: assets.Club}}
	if !IsValidTableauStack(single) {
		t.Error("Single card should be valid")
	}

	// Empty is valid
	if !IsValidTableauStack([]Card{}) {
		t.Error("Empty stack should be valid")
	}
}

func TestGetMovableCards(t *testing.T) {
	g := setupGame()

	// Set up a column with a valid stack at the bottom
	g.Tableau[0] = []Card{
		{Rank: 10, Suit: assets.Heart},  // Can't include this
		{Rank: 6, Suit: assets.Club},    // Start of valid stack
		{Rank: 5, Suit: assets.Diamond}, // Part of stack
		{Rank: 4, Suit: assets.Spade},   // Part of stack
	}

	// Getting from index 1 should return 3 cards
	cards := g.GetMovableCards(0, 1)
	if len(cards) != 3 {
		t.Errorf("GetMovableCards from idx 1 = %d cards, want 3", len(cards))
	}

	// Getting from index 0 should return nil (10H doesn't connect to 6C)
	cards = g.GetMovableCards(0, 0)
	if cards != nil {
		t.Errorf("GetMovableCards from idx 0 should be nil, got %d cards", len(cards))
	}
}

func TestMoveToFreeCell(t *testing.T) {
	g := setupGame()

	// Get the top card from column 0
	topCard := g.Tableau[0][len(g.Tableau[0])-1]
	from := Position{Location: LocTableau, Index: 0, CardIdx: len(g.Tableau[0]) - 1}

	// Move to empty free cell should succeed
	if !g.MoveToFreeCell(from, 0) {
		t.Error("Move to empty free cell should succeed")
	}

	if g.FreeCells[0] == nil || *g.FreeCells[0] != topCard {
		t.Error("Free cell should have the card")
	}

	// Move to occupied free cell should fail
	from = Position{Location: LocTableau, Index: 1, CardIdx: len(g.Tableau[1]) - 1}
	if g.MoveToFreeCell(from, 0) {
		t.Error("Move to occupied free cell should fail")
	}
}

func TestUndo(t *testing.T) {
	g := setupGame()

	origLen := len(g.Tableau[0])
	from := Position{Location: LocTableau, Index: 0, CardIdx: origLen - 1}

	// Move to free cell
	g.MoveToFreeCell(from, 0)

	// Undo
	g.Undo()

	// Card should be back
	if len(g.Tableau[0]) != origLen {
		t.Error("After undo, card should be back in tableau")
	}
	if g.FreeCells[0] != nil {
		t.Error("After undo, free cell should be empty")
	}
}

func TestIsWon(t *testing.T) {
	g := &Game{}

	// Empty game is not won
	if g.IsWon() {
		t.Error("Empty game should not be won")
	}

	// Fill all foundations with 13 cards each
	for suit := 0; suit < 4; suit++ {
		for rank := 1; rank <= 13; rank++ {
			g.Foundations[suit] = append(g.Foundations[suit], Card{
				Rank: assets.Rank(rank),
				Suit: assets.Suit(suit),
			})
		}
	}

	if !g.IsWon() {
		t.Error("Full foundations should be won")
	}
}
