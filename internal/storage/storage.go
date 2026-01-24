package storage

// Storage provides an abstraction for reading and writing data.
// On native platforms this uses the filesystem, on WASM it uses localStorage.
type Storage interface {
	Read(key string) ([]byte, error)
	Write(key string, data []byte) error
	Delete(key string) error
	Exists(key string) bool
}

// Default returns the default storage implementation for the current platform.
func Default() Storage {
	return defaultStorage()
}
