package assets

import (
	"bytes"
	"embed"
	"fmt"
	"image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed cards/*.png
var cardFiles embed.FS

// Suit represents a card suit
type Suit int

const (
	Club Suit = iota
	Diamond
	Heart
	Spade
)

func (s Suit) String() string {
	return []string{"clubs", "diamonds", "hearts", "spades"}[s]
}

// IsRed returns true if the suit is red (hearts or diamonds)
func (s Suit) IsRed() bool {
	return s == Diamond || s == Heart
}

// Rank represents a card rank (1=Ace, 11=Jack, 12=Queen, 13=King)
type Rank int

const (
	Ace   Rank = 1
	Jack  Rank = 11
	Queen Rank = 12
	King  Rank = 13
)

func (r Rank) String() string {
	switch r {
	case Ace:
		return "ace"
	case Jack:
		return "jack"
	case Queen:
		return "queen"
	case King:
		return "king"
	default:
		return fmt.Sprintf("%d", r)
	}
}

// Card images
var cardImages map[string]*ebiten.Image

// Card dimensions from the PNGs
const (
	baseWidth  = 500
	baseHeight = 726
)

// AspectRatio returns the card aspect ratio (width/height)
func AspectRatio() float64 {
	return float64(baseWidth) / float64(baseHeight)
}

// LoadCards loads all card PNG images
func LoadCards() error {
	cardImages = make(map[string]*ebiten.Image)

	suits := []Suit{Club, Diamond, Heart, Spade}
	ranks := []Rank{Ace, 2, 3, 4, 5, 6, 7, 8, 9, 10, Jack, Queen, King}

	for _, suit := range suits {
		for _, rank := range ranks {
			filename := fmt.Sprintf("cards/%s_of_%s.png", rank.String(), suit.String())
			img, err := loadPNG(filename)
			if err != nil {
				return fmt.Errorf("failed to load %s: %w", filename, err)
			}
			cardImages[cardKey(rank, suit)] = img
		}
	}

	return nil
}

func loadPNG(filename string) (*ebiten.Image, error) {
	data, err := cardFiles.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return ebiten.NewImageFromImage(img), nil
}

func cardKey(rank Rank, suit Suit) string {
	return fmt.Sprintf("%d_%d", rank, suit)
}

// GetCardImage returns the image for a specific card
func GetCardImage(rank Rank, suit Suit) *ebiten.Image {
	return cardImages[cardKey(rank, suit)]
}

// BaseWidth returns the original card width
func BaseWidth() int {
	return baseWidth
}

// BaseHeight returns the original card height
func BaseHeight() int {
	return baseHeight
}
