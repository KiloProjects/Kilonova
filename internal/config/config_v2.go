package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
)

var (
	configV2Path string
	flagMapMu    sync.RWMutex
	allFlags     map[string]any = make(map[string]any)
)

type configT interface{ string | bool | int }

type Flag[T configT] interface {
	Value() T
	Update(T)
}

type flag[T configT] struct {
	mu   sync.RWMutex
	name string
	val  T
}

func (f *flag[T]) Value() T {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.val
}

func (f *flag[T]) Update(newVal T) {
	defer func() {
		if err := SaveConfigV2(); err != nil {
			zap.S().Warn("Couldn't save flag: ", err)
		}
	}()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.val = newVal
}

func (f *flag[T]) sneakUpdate(newVal T) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.val = newVal
}

func GenFlag[T configT](name string, defaultVal T) Flag[T] {
	flagMapMu.Lock()
	defer flagMapMu.Unlock()
	f := &flag[T]{name: name, val: defaultVal}
	allFlags[name] = f
	return f
}

func trySneakUpdate[T configT](name string, newVal T) {
	val, ok := allFlags[name]
	if !ok {
		zap.S().Warnf("Unknown key %s", name)
		return
	}
	switch v := val.(type) {
	case *flag[T]:
		v.sneakUpdate(newVal)
	default:
		zap.S().Warnf("Flag type mismatch: expected %T, got flag[%T]", v, newVal)
	}
}

func LoadConfigV2() error {
	flagMapMu.RLock()
	defer flagMapMu.RUnlock()
	if configV2Path == "" {
		return errors.New("invalid config path")
	}
	f, err := os.OpenFile(configV2Path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	var data = make(map[string]any)
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	for key, val := range data {
		switch v := val.(type) {
		case string:
			trySneakUpdate(key, v)
		case bool:
			trySneakUpdate(key, v)
		case float64:
			trySneakUpdate(key, int(v))
		default:
			zap.S().Warnf("Unknown type in config flags: %T", v)
		}
	}
	return nil
}

func SaveConfigV2() error {
	if configV2Path == "" {
		return errors.New("invalid config path")
	}
	// Make the directories just in case they don't exist
	if err := os.MkdirAll(filepath.Dir(configV2Path), 0666); err != nil {
		return err
	}
	flagMapMu.RLock()
	defer flagMapMu.RUnlock()

	file, err := os.Create(configV2Path)
	if err != nil {
		return err
	}

	var data = make(map[string]any)
	for key, flg := range allFlags {
		switch v := flg.(type) {
		case *flag[string]:
			data[key] = v.Value()
		case *flag[int]:
			data[key] = v.Value()
		case *flag[bool]:
			data[key] = v.Value()
		default:
			zap.S().Warnf("Unknown type %T", v)
		}
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "\t")
	if err := enc.Encode(data); err != nil {
		file.Close() // We don't care if it errors out, it's over anyway
		return err
	}

	return file.Close()
}

func SetConfigV2Path(path string) {
	configV2Path = path
}
