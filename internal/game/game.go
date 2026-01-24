package game

import (
	"incell/internal/assets"
	"math/rand"
)

const (
	NumFreeCells  = 4
	NumFoundation = 4
	NumTableau    = 8
)

// Location represents where a card can be
type Location int

const (
	LocFreeCell Location = iota
	LocFoundation
	LocTableau
)

// Position identifies a specific card position
type Position struct {
	Location Location
	Index    int // Which pile (0-3 for free cells/foundations, 0-7 for tableau)
	CardIdx  int // Index within the pile (for tableau)
}

// Game represents the game state
type Game struct {
	FreeCells   [NumFreeCells]*Card   // 4 free cells (nil if empty)
	Foundations [NumFoundation][]Card // 4 foundation piles (one per suit)
	Tableau     [NumTableau][]Card    // 8 tableau columns
	History     []Move                // Move history for undo
	Selected    *Selection            // Currently selected cards
}

// Selection represents the currently selected card(s)
type Selection struct {
	Position Position
	Cards    []Card
}

// Move represents a game move for undo
type Move struct {
	From  Position
	To    Position
	Cards []Card
}

// New creates a new game
func New(iGetIt bool) *Game {
	g := &Game{}
	g.Deal(iGetIt)
	return g
}

// Deal shuffles and deals a new game
func (g *Game) Deal(iGetIt bool) {
	// Reset state
	g.FreeCells = [NumFreeCells]*Card{}
	g.Foundations = [NumFoundation][]Card{}
	g.Tableau = [NumTableau][]Card{}
	g.History = nil
	g.Selected = nil

	// Create and shuffle deck
	deck := NewDeck()

	if !iGetIt {
		filtered := deck[:0]
		for _, card := range deck {
			if card.Rank != assets.Queen {
				filtered = append(filtered, card)
			}
		}
		deck = filtered
	}

	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})

	// Deal to tableau
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
}

// MaxMovableCards returns the maximum number of cards that can be moved as a stack
// Based on the number of empty free cells and empty tableau columns
func (g *Game) MaxMovableCards() int {
	return g.MaxMovableCardsTo(-1)
}

// MaxMovableCardsTo returns the maximum number of cards that can be moved to a specific column
// If destCol is an empty column, it's excluded from the count since we can't use it as intermediate storage
func (g *Game) MaxMovableCardsTo(destCol int) int {
	emptyFreeCells := 0
	for _, fc := range g.FreeCells {
		if fc == nil {
			emptyFreeCells++
		}
	}

	emptyTableau := 0
	for i, t := range g.Tableau {
		if len(t) == 0 {
			// Don't count destination column as available for intermediate storage
			if i != destCol {
				emptyTableau++
			}
		}
	}

	// Formula: (1 + emptyFreeCells) * 2^emptyTableau
	// But we use a simpler approximation: (1 + emptyFreeCells) * (1 + emptyTableau)
	return (1 + emptyFreeCells) * (1 + emptyTableau)
}

// IsValidTableauStack checks if a sequence of cards forms a valid tableau stack
// (alternating colors, descending ranks)
func IsValidTableauStack(cards []Card) bool {
	if len(cards) <= 1 {
		return true
	}
	for i := 1; i < len(cards); i++ {
		if !cards[i].CanStackOnTableau(cards[i-1]) {
			return false
		}
	}
	return true
}

// GetMovableCards returns the cards that can be moved from a tableau column
// starting from the given index
func (g *Game) GetMovableCards(col, startIdx int) []Card {
	if col < 0 || col >= NumTableau || startIdx < 0 {
		return nil
	}
	pile := g.Tableau[col]
	if startIdx >= len(pile) {
		return nil
	}

	cards := pile[startIdx:]
	if !IsValidTableauStack(cards) {
		return nil
	}

	// Check if we have enough free cells/empty columns to move this many cards
	if len(cards) > g.MaxMovableCards() {
		return nil
	}

	return cards
}

// CanMoveToFreeCell checks if we can move a card to a free cell
func (g *Game) CanMoveToFreeCell(cellIdx int) bool {
	if cellIdx < 0 || cellIdx >= NumFreeCells {
		return false
	}
	return g.FreeCells[cellIdx] == nil
}

// CanMoveToFoundation checks if a card can be moved to a foundation
func (g *Game) CanMoveToFoundation(card Card, foundationIdx int) bool {
	if foundationIdx < 0 || foundationIdx >= NumFoundation {
		return false
	}
	pile := g.Foundations[foundationIdx]
	if len(pile) == 0 {
		return card.Rank == assets.Ace
	}
	top := pile[len(pile)-1]
	return card.Suit == top.Suit && card.Rank == top.Rank+1
}

// CanMoveToTableau checks if cards can be moved to a tableau column
func (g *Game) CanMoveToTableau(cards []Card, colIdx int) bool {
	if colIdx < 0 || colIdx >= NumTableau || len(cards) == 0 {
		return false
	}

	// Check if we have enough capacity to move this many cards
	if len(cards) > g.MaxMovableCardsTo(colIdx) {
		return false
	}

	pile := g.Tableau[colIdx]
	if len(pile) == 0 {
		return true // Any card can go on empty column
	}
	top := pile[len(pile)-1]
	return cards[0].CanStackOnTableau(top)
}

// FindFoundationForCard finds the foundation index where a card can be placed
// Returns -1 if no valid foundation
func (g *Game) FindFoundationForCard(card Card) int {
	for i := 0; i < NumFoundation; i++ {
		if g.CanMoveToFoundation(card, i) {
			return i
		}
	}
	return -1
}

// MoveToFreeCell moves a single card to a free cell
func (g *Game) MoveToFreeCell(from Position, cellIdx int) bool {
	if !g.CanMoveToFreeCell(cellIdx) {
		return false
	}

	var card Card
	switch from.Location {
	case LocFreeCell:
		if g.FreeCells[from.Index] == nil {
			return false
		}
		card = *g.FreeCells[from.Index]
		g.FreeCells[from.Index] = nil
	case LocTableau:
		pile := g.Tableau[from.Index]
		if len(pile) == 0 || from.CardIdx != len(pile)-1 {
			return false // Can only move top card
		}
		card = pile[len(pile)-1]
		g.Tableau[from.Index] = pile[:len(pile)-1]
	default:
		return false
	}

	g.FreeCells[cellIdx] = &card
	g.History = append(g.History, Move{
		From:  from,
		To:    Position{Location: LocFreeCell, Index: cellIdx},
		Cards: []Card{card},
	})
	return true
}

// MoveToFoundation moves a card to a foundation
func (g *Game) MoveToFoundation(from Position, foundationIdx int) bool {
	var card Card
	switch from.Location {
	case LocFreeCell:
		if g.FreeCells[from.Index] == nil {
			return false
		}
		card = *g.FreeCells[from.Index]
	case LocTableau:
		pile := g.Tableau[from.Index]
		if len(pile) == 0 || from.CardIdx != len(pile)-1 {
			return false
		}
		card = pile[len(pile)-1]
	default:
		return false
	}

	if !g.CanMoveToFoundation(card, foundationIdx) {
		return false
	}

	// Remove from source
	switch from.Location {
	case LocFreeCell:
		g.FreeCells[from.Index] = nil
	case LocTableau:
		g.Tableau[from.Index] = g.Tableau[from.Index][:len(g.Tableau[from.Index])-1]
	}

	g.Foundations[foundationIdx] = append(g.Foundations[foundationIdx], card)
	g.History = append(g.History, Move{
		From:  from,
		To:    Position{Location: LocFoundation, Index: foundationIdx},
		Cards: []Card{card},
	})
	return true
}

// MoveToTableau moves cards to a tableau column
func (g *Game) MoveToTableau(from Position, cards []Card, colIdx int) bool {
	if !g.CanMoveToTableau(cards, colIdx) {
		return false
	}

	// Remove from source
	switch from.Location {
	case LocFreeCell:
		if len(cards) != 1 || g.FreeCells[from.Index] == nil {
			return false
		}
		g.FreeCells[from.Index] = nil
	case LocTableau:
		pile := g.Tableau[from.Index]
		if from.CardIdx+len(cards) != len(pile) {
			return false
		}
		g.Tableau[from.Index] = pile[:from.CardIdx]
	default:
		return false
	}

	g.Tableau[colIdx] = append(g.Tableau[colIdx], cards...)
	g.History = append(g.History, Move{
		From:  from,
		To:    Position{Location: LocTableau, Index: colIdx},
		Cards: cards,
	})
	return true
}

// Undo undoes the last move
func (g *Game) Undo() bool {
	if len(g.History) == 0 {
		return false
	}

	move := g.History[len(g.History)-1]
	g.History = g.History[:len(g.History)-1]

	// Remove from destination
	switch move.To.Location {
	case LocFreeCell:
		g.FreeCells[move.To.Index] = nil
	case LocFoundation:
		pile := g.Foundations[move.To.Index]
		g.Foundations[move.To.Index] = pile[:len(pile)-len(move.Cards)]
	case LocTableau:
		pile := g.Tableau[move.To.Index]
		g.Tableau[move.To.Index] = pile[:len(pile)-len(move.Cards)]
	}

	// Add back to source
	switch move.From.Location {
	case LocFreeCell:
		g.FreeCells[move.From.Index] = &move.Cards[0]
	case LocTableau:
		g.Tableau[move.From.Index] = append(g.Tableau[move.From.Index], move.Cards...)
	}

	g.Selected = nil
	return true
}

// AutoMoveToFoundation attempts to automatically move cards to foundations
// Returns true if any card was moved
func (g *Game) AutoMoveToFoundation() bool {
	moved := false

	// Try free cells
	for i := 0; i < NumFreeCells; i++ {
		if g.FreeCells[i] == nil {
			continue
		}
		card := *g.FreeCells[i]
		if foundIdx := g.FindFoundationForCard(card); foundIdx >= 0 {
			if g.MoveToFoundation(Position{Location: LocFreeCell, Index: i}, foundIdx) {
				moved = true
			}
		}
	}

	// Try tableau
	for col := 0; col < NumTableau; col++ {
		pile := g.Tableau[col]
		if len(pile) == 0 {
			continue
		}
		card := pile[len(pile)-1]
		if foundIdx := g.FindFoundationForCard(card); foundIdx >= 0 {
			pos := Position{Location: LocTableau, Index: col, CardIdx: len(pile) - 1}
			if g.MoveToFoundation(pos, foundIdx) {
				moved = true
			}
		}
	}

	return moved
}

// FindBestDestination finds the best destination for cards at the given position
// Returns nil if no valid destination exists
// Priority for single card in tableau: Foundation > Free Cell > Tableau (non-empty first, leftmost)
// Priority for stack in tableau: Tableau only (non-empty first, leftmost)
// Priority for card in free cell: Foundation > Tableau (non-empty first, leftmost)
// Cards on foundations cannot be moved
func (g *Game) FindBestDestination(from Position) *Position {
	var cards []Card

	switch from.Location {
	case LocFoundation:
		// Cards on foundations are locked
		return nil

	case LocFreeCell:
		if g.FreeCells[from.Index] == nil {
			return nil
		}
		cards = []Card{*g.FreeCells[from.Index]}

		// Try foundation first
		if foundIdx := g.FindFoundationForCard(cards[0]); foundIdx >= 0 {
			return &Position{Location: LocFoundation, Index: foundIdx}
		}

		// Try tableau (non-empty columns first, leftmost)
		if dest := g.findBestTableauDestination(cards); dest != nil {
			return dest
		}

		return nil

	case LocTableau:
		cards = g.GetMovableCards(from.Index, from.CardIdx)
		if len(cards) == 0 {
			return nil
		}

		isSingleCard := len(cards) == 1

		if isSingleCard {
			// Single card: try foundation first
			if foundIdx := g.FindFoundationForCard(cards[0]); foundIdx >= 0 {
				return &Position{Location: LocFoundation, Index: foundIdx}
			}
		}

		// Try tableau (non-empty columns first, leftmost)
		// Skip the source column
		if dest := g.findBestTableauDestinationExcept(cards, from.Index); dest != nil {
			return dest
		}

		// Last resort for single card: try free cell
		if isSingleCard {
			for i := 0; i < NumFreeCells; i++ {
				if g.FreeCells[i] == nil {
					return &Position{Location: LocFreeCell, Index: i}
				}
			}
		}

		return nil
	}

	return nil
}

// findBestTableauDestination finds the best tableau column to move cards to
// Prefers non-empty columns (smallest stack first), then empty columns (leftmost)
func (g *Game) findBestTableauDestination(cards []Card) *Position {
	return g.findBestTableauDestinationExcept(cards, -1)
}

// findBestTableauDestinationExcept finds the best tableau column, excluding one column
func (g *Game) findBestTableauDestinationExcept(cards []Card, excludeCol int) *Position {
	// First pass: non-empty columns (smallest stack first)
	bestCol := -1
	bestSize := -1
	for col := 0; col < NumTableau; col++ {
		if col == excludeCol {
			continue
		}
		size := len(g.Tableau[col])
		if size > 0 && g.CanMoveToTableau(cards, col) {
			if bestCol == -1 || size < bestSize {
				bestCol = col
				bestSize = size
			}
		}
	}
	if bestCol >= 0 {
		return &Position{Location: LocTableau, Index: bestCol}
	}

	// Second pass: empty columns (leftmost)
	for col := 0; col < NumTableau; col++ {
		if col == excludeCol {
			continue
		}
		if len(g.Tableau[col]) == 0 && g.CanMoveToTableau(cards, col) {
			return &Position{Location: LocTableau, Index: col}
		}
	}

	return nil
}

// IsWon returns true if the game is won (all cards on foundations)
func (g *Game) IsWon() bool {
	for _, pile := range g.Foundations {
		if len(pile) != 13 {
			return false
		}
	}
	return true
}

// FindAllHints returns all positions that have valid moves
func (g *Game) FindAllHints() []Position {
	var hints []Position
	seen := make(map[Position]bool)

	addHint := func(pos Position) {
		if !seen[pos] {
			seen[pos] = true
			hints = append(hints, pos)
		}
	}

	// Check tableau first - prefer moves that build sequences
	for col := 0; col < NumTableau; col++ {
		pile := g.Tableau[col]
		if len(pile) == 0 {
			continue
		}

		// Check top card for foundation (best move)
		topCard := pile[len(pile)-1]
		if g.FindFoundationForCard(topCard) >= 0 {
			addHint(Position{Location: LocTableau, Index: col, CardIdx: len(pile) - 1})
		}
	}

	// Check free cells for foundation moves
	for i := 0; i < NumFreeCells; i++ {
		if g.FreeCells[i] == nil {
			continue
		}
		card := *g.FreeCells[i]
		if g.FindFoundationForCard(card) >= 0 {
			addHint(Position{Location: LocFreeCell, Index: i})
		}
	}

	// Check tableau stacks for tableau moves
	for col := 0; col < NumTableau; col++ {
		pile := g.Tableau[col]
		if len(pile) == 0 {
			continue
		}

		// Check all movable stacks
		for startIdx := 0; startIdx < len(pile); startIdx++ {
			cards := g.GetMovableCards(col, startIdx)
			if len(cards) == 0 {
				continue
			}
			// Can this stack move to another non-empty column?
			for destCol := 0; destCol < NumTableau; destCol++ {
				if destCol == col || len(g.Tableau[destCol]) == 0 {
					continue
				}
				if g.CanMoveToTableau(cards, destCol) {
					addHint(Position{Location: LocTableau, Index: col, CardIdx: startIdx})
					break // Only need one valid destination
				}
			}
		}
	}

	// Check free cells for tableau moves (including empty columns)
	for i := 0; i < NumFreeCells; i++ {
		if g.FreeCells[i] == nil {
			continue
		}
		card := *g.FreeCells[i]
		for col := 0; col < NumTableau; col++ {
			if g.CanMoveToTableau([]Card{card}, col) {
				addHint(Position{Location: LocFreeCell, Index: i})
				break // Only need one valid destination
			}
		}
	}

	return hints
}

// HasValidMoves returns true if there are any valid moves available
func (g *Game) HasValidMoves() bool {
	// Check if any free cell card can move
	for i := 0; i < NumFreeCells; i++ {
		if g.FreeCells[i] == nil {
			continue
		}
		card := *g.FreeCells[i]
		// Can it go to foundation?
		if g.FindFoundationForCard(card) >= 0 {
			return true
		}
		// Can it go to tableau?
		for col := 0; col < NumTableau; col++ {
			if g.CanMoveToTableau([]Card{card}, col) {
				return true
			}
		}
	}

	// Check if any tableau card/stack can move
	for col := 0; col < NumTableau; col++ {
		pile := g.Tableau[col]
		if len(pile) == 0 {
			continue
		}

		// Check top card for foundation
		topCard := pile[len(pile)-1]
		if g.FindFoundationForCard(topCard) >= 0 {
			return true
		}

		// Check top card for free cell (if any empty)
		for i := 0; i < NumFreeCells; i++ {
			if g.FreeCells[i] == nil {
				return true // Can always move top card to empty free cell
			}
		}

		// Check all movable stacks for tableau moves
		for startIdx := 0; startIdx < len(pile); startIdx++ {
			cards := g.GetMovableCards(col, startIdx)
			if len(cards) == 0 {
				continue
			}
			// Can this stack move to another column?
			for destCol := 0; destCol < NumTableau; destCol++ {
				if destCol == col {
					continue
				}
				if g.CanMoveToTableau(cards, destCol) {
					return true
				}
			}
		}
	}

	return false
}

// Select selects cards at the given position
func (g *Game) Select(pos Position) bool {
	g.Selected = nil

	switch pos.Location {
	case LocFreeCell:
		if g.FreeCells[pos.Index] == nil {
			return false
		}
		g.Selected = &Selection{
			Position: pos,
			Cards:    []Card{*g.FreeCells[pos.Index]},
		}
		return true

	case LocTableau:
		cards := g.GetMovableCards(pos.Index, pos.CardIdx)
		if len(cards) == 0 {
			return false
		}
		g.Selected = &Selection{
			Position: pos,
			Cards:    append([]Card{}, cards...),
		}
		return true
	}

	return false
}

// ClearSelection clears the current selection
func (g *Game) ClearSelection() {
	g.Selected = nil
}

// TryMove attempts to move the selected cards to the given position
func (g *Game) TryMove(to Position) bool {
	if g.Selected == nil {
		return false
	}

	from := g.Selected.Position
	cards := g.Selected.Cards
	g.Selected = nil

	switch to.Location {
	case LocFreeCell:
		if len(cards) == 1 {
			return g.MoveToFreeCell(from, to.Index)
		}
	case LocFoundation:
		if len(cards) == 1 {
			return g.MoveToFoundation(from, to.Index)
		}
	case LocTableau:
		return g.MoveToTableau(from, cards, to.Index)
	}

	return false
}
