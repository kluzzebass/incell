//go:build !js

package storage

import (
	"os"
	"path/filepath"
)

type fileStorage struct {
	dir string
}

func defaultStorage() Storage {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	return &fileStorage{dir: filepath.Join(configDir, "incell")}
}

func (f *fileStorage) path(key string) string {
	return filepath.Join(f.dir, key+".json")
}

func (f *fileStorage) Read(key string) ([]byte, error) {
	return os.ReadFile(f.path(key))
}

func (f *fileStorage) Write(key string, data []byte) error {
	if err := os.MkdirAll(f.dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(f.path(key), data, 0644)
}

func (f *fileStorage) Delete(key string) error {
	return os.Remove(f.path(key))
}

func (f *fileStorage) Exists(key string) bool {
	_, err := os.Stat(f.path(key))
	return err == nil
}
