package game

import (
	"encoding/json"
	"os"
	"path/filepath"
)

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

// settingsPath returns the path to the settings file
func settingsPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "incell", "settings.json"), nil
}

// LoadSettings loads settings from disk, or returns defaults if not found
func LoadSettings() Settings {
	path, err := settingsPath()
	if err != nil {
		return DefaultSettings()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultSettings()
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return DefaultSettings()
	}

	return settings
}

// SaveSettings saves settings to disk
func SaveSettings(settings Settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
