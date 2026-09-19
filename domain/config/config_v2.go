package config

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"
)

var (
	flagMapMu sync.RWMutex
	allFlags  map[string]any = make(map[string]any)
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
		if onFlagUpdate != nil {
			onFlagUpdate(f.name)
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

// FlagStore persists runtime flags. It is implemented by the database layer;
// declared here as an interface because db imports this package.
type FlagStore interface {
	GetFlags(ctx context.Context) (map[string]json.RawMessage, error)
	SetFlag(ctx context.Context, key string, value json.RawMessage) error
}

// importSentinel marks a database whose flags were already seeded from a legacy
// flags.json. Keys starting with "_" are never registered flags, so the sentinel
// cannot collide with one.
const importSentinel = "_flags_file_imported"

// LoadFlagsFromDB applies stored flag values over the compiled-in defaults.
// A value that does not decode, or a key no longer registered in this build, is
// logged and skipped: neither aborts startup, and unknown keys are left in the
// store so a rollback still sees them. It reports whether the store held anything,
// which is what decides if a legacy file still needs importing.
func LoadFlagsFromDB(ctx context.Context, store FlagStore) (empty bool, err error) {
	stored, err := store.GetFlags(ctx)
	if err != nil {
		return false, err
	}
	if len(stored) == 0 {
		return true, nil
	}

	flagMapMu.RLock()
	defer flagMapMu.RUnlock()
	for key, val := range stored {
		if strings.HasPrefix(key, "_") {
			continue
		}
		flg, ok := allFlags[key]
		if !ok {
			slog.WarnContext(ctx, "Unknown flag in store, ignoring", slog.String("key", key))
			continue
		}
		v, ok := flg.(configFlag)
		if !ok {
			slog.WarnContext(ctx, "Could not sneak update", slog.String("key", key))
			continue
		}
		if err := v.sneakUpdate(val); err != nil {
			slog.WarnContext(ctx, "Couldn't apply stored flag, keeping default", slog.String("key", key), slog.Any("err", err))
		}
	}
	return false, nil
}

// ImportFlagsFile seeds an empty flag store from a legacy flags.json. The file is
// only ever read: after this runs, the store is the source of truth and the file
// is inert. Every registered flag is written, so the store is non-empty afterwards
// even when the file was missing and the sentinel is the only thing that changed.
func ImportFlagsFile(ctx context.Context, store FlagStore, configV2Path string) error {
	if err := LoadConfigV2(ctx, configV2Path, true); err != nil {
		slog.WarnContext(ctx, "Couldn't read flags file for import, seeding defaults instead",
			slog.String("path", configV2Path), slog.Any("err", err))
	}

	flagMapMu.RLock()
	defer flagMapMu.RUnlock()
	for key, flg := range allFlags {
		v, ok := flg.(configFlag)
		if !ok {
			continue
		}
		val, err := json.Marshal(v.getPtr())
		if err != nil {
			slog.WarnContext(ctx, "Couldn't encode flag for import", slog.String("key", key), slog.Any("err", err))
			continue
		}
		if err := store.SetFlag(ctx, key, val); err != nil {
			return fmt.Errorf("import flag %q: %w", key, err)
		}
	}
	if err := store.SetFlag(ctx, importSentinel, json.RawMessage(`true`)); err != nil {
		return fmt.Errorf("mark flags file as imported: %w", err)
	}
	slog.InfoContext(ctx, "Imported flags into the database", slog.String("path", configV2Path), slog.Int("flags", len(allFlags)))
	return nil
}

// PersistFlag writes one flag's current value to the store. Used as the update
// callback, so an admin edit touches exactly the flag that changed.
func PersistFlag(ctx context.Context, store FlagStore, name string) error {
	flagMapMu.RLock()
	flg, ok := allFlags[name]
	flagMapMu.RUnlock()
	if !ok {
		return fmt.Errorf("unknown flag %q", name)
	}
	v, ok := flg.(configFlag)
	if !ok {
		return fmt.Errorf("flag %q is not persistable", name)
	}
	val, err := json.Marshal(v.getPtr())
	if err != nil {
		return err
	}
	return store.SetFlag(ctx, name, val)
}

func LoadConfigV2(ctx context.Context, configV2Path string, skipUnknown bool) error {
	flagMapMu.RLock()
	defer flagMapMu.RUnlock()
	if configV2Path == "" {
		return errors.New("invalid config path")
	}
	// Read-only, and never created: the flag store is the database, this file is
	// only ever an import source.
	f, err := os.Open(configV2Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
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
			if skipUnknown {
				slog.WarnContext(ctx, "Unknown config key", slog.String("key", key))
			}
			continue
		}
		if v, ok := val.(configFlag); ok {
			if err := v.sneakUpdate(confVal); err != nil {
				slog.WarnContext(ctx, "Couldn't update key", slog.String("key", key), slog.Any("err", err))
			}
		} else {
			slog.WarnContext(ctx, "Could not sneak update")
		}
	}

	return nil
}

// ApplyFlagOverrides applies the per-process KN_FLAG_OVERRIDES escape hatch.
// Overrides are never persisted, and apply in processes with no flag store at all.
func ApplyFlagOverrides(ctx context.Context) {
	flagMapMu.RLock()
	defer flagMapMu.RUnlock()

	for override := range strings.SplitSeq(os.Getenv("KN_FLAG_OVERRIDES"), ",") {
		if override == "" {
			continue
		}
		key, val, found := strings.Cut(override, "=")
		if !found {
			slog.WarnContext(ctx, "Invalid override", slog.String("override", override))
			continue
		}
		flg, ok := allFlags[key]
		if !ok {
			slog.WarnContext(ctx, "Could not find flag", slog.String("name", key))
			continue
		}
		switch f := flg.(type) {
		case *flag[string]:
			// Strings are a bit special since they don't like the fact that overrides may not have quotes
			f.Update(val)
		case configFlag:
			if err := json.Unmarshal([]byte(val), f.getPtr()); err != nil {
				slog.WarnContext(ctx, "Invalid flag override", slog.Any("err", err), slog.String("key", key))
			}
		default:
			slog.WarnContext(ctx, "Unknown flag type")
		}
	}
}

var onFlagUpdate = func(string) {}

// SetOnFlagUpdate registers the persistence callback. It is handed the dotted
// name of the flag that changed, so only that flag is written.
func SetOnFlagUpdate(f func(name string)) {
	onFlagUpdate = f
}
