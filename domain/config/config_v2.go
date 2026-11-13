package config

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
)

var (
	flagMapMu sync.RWMutex
	allFlags  = make(map[string]any)
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
func LoadFlagsFromDB(ctx context.Context, store FlagStore) (err error) {
	stored, err := store.GetFlags(ctx)
	if err != nil {
		return err
	}
	if len(stored) == 0 {
		return nil
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

var onFlagUpdate = func(string) {}

// SetOnFlagUpdate registers the persistence callback. It is handed the dotted
// name of the flag that changed, so only that flag is written.
func SetOnFlagUpdate(f func(name string)) {
	onFlagUpdate = f
}
