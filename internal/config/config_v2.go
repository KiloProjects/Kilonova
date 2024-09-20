package config

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"go.uber.org/zap"
)

var (
	configV2Path string
	flagMapMu    sync.RWMutex
	allFlags     map[string]any = make(map[string]any)
)

type configFlag interface {
	getPtr() any
	sneakUpdate(newVal any) error
}

type Flag[T any] interface {
	Value() T
	Update(T)
	InternalName() string
	HumanName() string
}

type flag[T any] struct {
	mu        sync.RWMutex
	name      string
	val       T
	humanName string
}

func (f *flag[T]) Value() T {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.val
}

func (f *flag[T]) InternalName() string {
	return f.name
}

func (f *flag[T]) HumanName() string {
	return f.humanName
}

func (f *flag[T]) MarshalJSON() ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return json.Marshal(&struct {
		InternalName string `json:"internal_name"`
		HumanName    string `json:"human_name"`
		Value        T      `json:"value"`
	}{
		InternalName: f.name,
		HumanName:    f.humanName,
		Value:        f.val,
	})
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

func (f *flag[T]) getPtr() any {
	return &f.val
}

func (f *flag[T]) sneakUpdate(newVal any) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch v := newVal.(type) {
	case json.RawMessage:
		if err := json.Unmarshal(v, &f.val); err != nil {
			return fmt.Errorf("invalid key, flag expected %T", f.val)
		}
		return nil
	default:
		return fmt.Errorf("expected json.RawMessage, got %T", newVal)
	}
}

func GenFlag[T any](name string, defaultVal T, readableName string) Flag[T] {
	flagMapMu.Lock()
	defer flagMapMu.Unlock()
	f := &flag[T]{name: name, val: defaultVal, humanName: readableName}
	allFlags[name] = f
	return f
}

func GetFlagVal[T any](name string) (T, bool) {
	flagMapMu.RLock()
	defer flagMapMu.RUnlock()
	flg, ok := allFlags[name]
	if !ok {
		return *new(T), false
	}
	if v, ok := flg.(*flag[T]); ok {
		return v.Value(), true
	}
	return *new(T), false
}

func GetFlag[T any](name string) (Flag[T], bool) {
	flagMapMu.RLock()
	defer flagMapMu.RUnlock()
	flg, ok := allFlags[name]
	if !ok {
		return nil, false
	}
	v, ok := flg.(*flag[T])
	return v, ok
}

func GetFlags[T any]() []Flag[T] {
	flagMapMu.RLock()
	defer flagMapMu.RUnlock()
	var flags []Flag[T]
	for _, flg := range allFlags {
		flag, ok := flg.(*flag[T])
		if ok {
			flags = append(flags, flag)
		}
	}
	slices.SortFunc(flags, func(a, b Flag[T]) int {
		return cmp.Compare(a.InternalName(), b.InternalName())
	})
	return flags
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

	var data = make(map[string]json.RawMessage)
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	for key, confVal := range data {
		// Do sneak update
		val, ok := allFlags[key]
		if !ok {
			slog.Warn("Unknown config key", slog.String("key", key))
			continue
		}
		if v, ok := val.(configFlag); ok {
			if err := v.sneakUpdate(confVal); err != nil {
				zap.S().Warnf("Could not update key %q: %v", key, err)
			}
		} else {
			zap.S().Warn("Could not sneak update")
		}
	}

	overrides := strings.Split(os.Getenv("KN_FLAG_OVERRIDES"), ",")
	for _, override := range overrides {
		if override == "" {
			continue
		}
		key, val, found := strings.Cut(override, "=")
		if !found {
			zap.S().Warnf("Invalid override %q", override)
			continue
		}
		flg, ok := allFlags[key]
		if !ok {
			zap.S().Warnf("Could not find flag named %q", key)
			continue
		}
		switch f := flg.(type) {
		case *flag[string]:
			// Strings are a bit special since they don't like the fact that overrides may not have quotes
			f.Update(val)
		case configFlag:
			if err := json.Unmarshal([]byte(val), f.getPtr()); err != nil {
				zap.S().Warnf("Override for flag %q is invalid: %v", key, err)
			}
		default:
			zap.S().Warnf("Unknown flag type")
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
		case configFlag:
			data[key] = v.getPtr()
		default:
			zap.S().Warnf("Unknown type %T", v)
		}
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "\t")
	if err := enc.Encode(data); err != nil {
		file.Close() // We don't care if it errors out, the JSON is errored
		return err
	}

	return file.Close()
}

func SetConfigV2Path(path string) {
	configV2Path = path
}
