package game

import "incell/internal/assets"

// Card represents a playing card
type Card struct {
	Rank assets.Rank
	Suit assets.Suit
}

// IsRed returns true if the card is red (hearts or diamonds)
func (c Card) IsRed() bool {
	return c.Suit.IsRed()
}

// CanStackOnTableau returns true if this card can be placed on top of another card in a tableau
// In FreeCell, cards must alternate colors and descend in rank
func (c Card) CanStackOnTableau(other Card) bool {
	// Must be opposite color
	if c.IsRed() == other.IsRed() {
		return false
	}
	// Must be one rank lower
	return c.Rank == other.Rank-1
}

// CanStackOnFoundation returns true if this card can be placed on a foundation pile
// Foundations build up by suit from Ace to King
func (c Card) CanStackOnFoundation(top *Card) bool {
	if top == nil {
		// Empty foundation - only Aces can be placed
		return c.Rank == assets.Ace
	}
	// Must be same suit and one rank higher
	return c.Suit == top.Suit && c.Rank == top.Rank+1
}

// NewDeck creates a standard 52-card deck
func NewDeck() []Card {
	deck := make([]Card, 0, 52)
	for suit := assets.Club; suit <= assets.Spade; suit++ {
		for rank := assets.Ace; rank <= assets.King; rank++ {
			deck = append(deck, Card{Rank: rank, Suit: suit})
		}
	}
	return deck
}
