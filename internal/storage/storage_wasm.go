//go:build js

package storage

import (
	"errors"
	"syscall/js"
)

type localStorageImpl struct {
	prefix string
}

func defaultStorage() Storage {
	return &localStorageImpl{prefix: "incell_"}
}

func (l *localStorageImpl) key(name string) string {
	return l.prefix + name
}

func (l *localStorageImpl) Read(key string) ([]byte, error) {
	localStorage := js.Global().Get("localStorage")
	if localStorage.IsUndefined() {
		return nil, errors.New("localStorage not available")
	}

	val := localStorage.Call("getItem", l.key(key))
	if val.IsNull() {
		return nil, errors.New("key not found")
	}

	return []byte(val.String()), nil
}

func (l *localStorageImpl) Write(key string, data []byte) error {
	localStorage := js.Global().Get("localStorage")
	if localStorage.IsUndefined() {
		return errors.New("localStorage not available")
	}

	localStorage.Call("setItem", l.key(key), string(data))
	return nil
}

func (l *localStorageImpl) Delete(key string) error {
	localStorage := js.Global().Get("localStorage")
	if localStorage.IsUndefined() {
		return errors.New("localStorage not available")
	}

	localStorage.Call("removeItem", l.key(key))
	return nil
}

func (l *localStorageImpl) Exists(key string) bool {
	localStorage := js.Global().Get("localStorage")
	if localStorage.IsUndefined() {
		return false
	}

	val := localStorage.Call("getItem", l.key(key))
	return !val.IsNull()
}
