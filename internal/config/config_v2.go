package config

import (
	"cmp"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
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
	InternalName() string
	HumanName() string
}

type flag[T configT] struct {
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

func (f *flag[T]) sneakUpdate(newVal T) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.val = newVal
}

func GenFlag[T configT](name string, defaultVal T, readableName string) Flag[T] {
	flagMapMu.Lock()
	defer flagMapMu.Unlock()
	f := &flag[T]{name: name, val: defaultVal, humanName: readableName}
	allFlags[name] = f
	return f
}

func GetFlagVal[T configT](name string) (T, bool) {
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

func GetFlag[T configT](name string) (Flag[T], bool) {
	flagMapMu.RLock()
	defer flagMapMu.RUnlock()
	flg, ok := allFlags[name]
	if !ok {
		return nil, false
	}
	v, ok := flg.(*flag[T])
	return v, ok
}

func GetFlags[T configT]() []Flag[T] {
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

	overrides := strings.Split(os.Getenv("KN_FLAG_OVERRIDES"), ",")
	for _, override := range overrides {
		if override == "" {
			continue
		}
		vals := strings.SplitN(override, "=", 2)
		if len(vals) != 2 {
			zap.S().Warnf("Invalid override %q", override)
			continue
		}
		flag, ok := allFlags[vals[0]]
		if !ok {
			zap.S().Warnf("Could not find flag named %q", vals[0])
			continue
		}
		switch f := flag.(type) {
		case Flag[int]:
			val, err := strconv.Atoi(vals[1])
			if err != nil {
				zap.S().Warnf("Override for flag %q is not int", vals[0])
				continue
			}
			f.Update(val)
		case Flag[string]:
			f.Update(vals[1])
		case Flag[bool]:
			val, err := strconv.ParseBool(vals[1])
			if err != nil {
				zap.S().Warnf("Override for flag %q is not boolean", vals[0])
				continue
			}
			f.Update(val)
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
