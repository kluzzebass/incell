package assets

import (
	"bytes"
	"embed"
	"image"
	"image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed icon.png
var iconFile embed.FS

var iconImage *ebiten.Image

// LoadIcon loads the application icon
func LoadIcon() (image.Image, error) {
	data, err := iconFile.ReadFile("icon.png")
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Cache as ebiten image for UI use
	iconImage = ebiten.NewImageFromImage(img)

	return img, nil
}

// GetIconImage returns the icon as an ebiten image (must call LoadIcon first)
func GetIconImage() *ebiten.Image {
	return iconImage
}
