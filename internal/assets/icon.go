package assets

import (
	"bytes"
	"embed"
	"image"
	"image/png"
)

//go:embed icon.png
var iconFile embed.FS

// LoadIcon loads the application icon
func LoadIcon() (image.Image, error) {
	data, err := iconFile.ReadFile("icon.png")
	if err != nil {
		return nil, err
	}

	return png.Decode(bytes.NewReader(data))
}
