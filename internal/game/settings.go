package game

import (
	"encoding/json"

	"incell/internal/storage"
)

const settingsKey = "settings"

// Settings holds the game configuration
type Settings struct {
	AutoMove bool `json:"auto_move"`
	IGetIt   bool `json:"i_get_it"`
}

// DefaultSettings returns the default settings
func DefaultSettings() Settings {
	return Settings{
		AutoMove: true,
		IGetIt:   false,
	}
}

// LoadSettings loads settings from storage, or returns defaults if not found
func LoadSettings() Settings {
	data, err := storage.Default().Read(settingsKey)
	if err != nil {
		return DefaultSettings()
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return DefaultSettings()
	}

	return settings
}

// SaveSettings saves settings to storage
func SaveSettings(settings Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return storage.Default().Write(settingsKey, data)
}
