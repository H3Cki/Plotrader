package eier

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

type ExchangeInfoer[T any] struct {
	DirPath string
}

func New[T any](dirPath string) (*ExchangeInfoer[T], error) {
	f, err := os.Stat(dirPath)
	if err == nil && !f.IsDir() {
		return nil, fmt.Errorf("path %s already exists and is not a directory", dirPath)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("unexpected error: %v", err)
	}
	return &ExchangeInfoer[T]{DirPath: dirPath}, nil
}

func (e *ExchangeInfoer[T]) Exists(name string) bool {
	_, err := os.Stat("/path/to/your-file")
	return os.IsNotExist(err)
}

func (e *ExchangeInfoer[T]) Save(name string, exchangeInfo T) error {
	if _, err := os.Stat(e.DirPath); os.IsNotExist(err) {
		return os.MkdirAll(e.DirPath, 0700)
	}
	bytes, err := json.Marshal(exchangeInfo)
	if err != nil {
		return err
	}
	return os.WriteFile(e.path(name), bytes, 0o777)
}

func (e *ExchangeInfoer[T]) Read(name string) (*T, error) {
	if _, err := os.Stat(e.DirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(e.DirPath, 0700); err != nil {
			return nil, err
		}
	}
	bytes, err := os.ReadFile(e.path(name))
	if err != nil {
		return nil, err
	}

	ei := new(T)
	if err := json.Unmarshal(bytes, ei); err != nil {
		return nil, err
	}

	return ei, nil
}

func (e *ExchangeInfoer[T]) path(name string) string {
	return path.Join(e.DirPath, name)
}
