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
	"time"
)

type flagset struct {
	mu    sync.RWMutex
	flags map[string]any

	store FlagStore
}

var globalFlags = newFlagset()

func newFlagset() *flagset {
	return &flagset{flags: map[string]any{}}
}

type configFlag interface {
	sneakUpdate(newVal any) error
}

type Flag[T any] interface {
	Value() T
	Update(context.Context, T)
	InternalName() string
	HumanName() string
}

type flag[T any] struct {
	mu        sync.RWMutex
	name      string
	val       T
	humanName string
	fs        *flagset
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

// Update sets the value and writes it through to the flag store, if one is attached.
// The store write happens outside every lock, so readers never wait on the database.
// ponytail: two concurrent edits of the same flag may persist out of order; fine for admin edits.
func (f *flag[T]) Update(ctx context.Context, newVal T) {
	f.mu.Lock()
	f.val = newVal
	data, err := json.Marshal(newVal)
	f.mu.Unlock()
	if err != nil {
		slog.WarnContext(ctx, "Couldn't encode flag", slog.String("flag", f.name), slog.Any("err", err))
		return
	}
	if f.fs != nil {
		f.fs.persist(ctx, f.name, data)
	}
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

func (fs *flagset) GenFlag[T any](name string, defaultVal T, readableName string) Flag[T] {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	f := &flag[T]{name: name, val: defaultVal, humanName: readableName, fs: fs}
	fs.flags[name] = f
	return f
}

func GenFlag[T any](name string, defaultVal T, readableName string) Flag[T] {
	return globalFlags.GenFlag(name, defaultVal, readableName)
}

func (fs *flagset) GetFlag[T any](name string) (Flag[T], bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	flg, ok := fs.flags[name]
	if !ok {
		return nil, false
	}
	v, ok := flg.(*flag[T])
	return v, ok
}

func GetFlag[T any](name string) (Flag[T], bool) {
	return globalFlags.GetFlag[T](name)
}

func (fs *flagset) GetFlags[T any]() []Flag[T] {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	var flags []Flag[T]
	for _, flg := range fs.flags {
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

func GetFlags[T any]() []Flag[T] {
	return globalFlags.GetFlags[T]()
}

// FlagStore persists runtime flags. It is implemented by the database layer;
// declared here as an interface because db imports this package.
type FlagStore interface {
	GetFlags(ctx context.Context) (map[string]json.RawMessage, error)
	SetFlag(ctx context.Context, key string, value json.RawMessage) error
}

func (fs *flagset) InitializeFlags(ctx context.Context, store FlagStore) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.store = store

	stored, err := fs.store.GetFlags(ctx)
	if err != nil {
		return err
	}

	for key, val := range stored {
		if strings.HasPrefix(key, "_") {
			continue
		}
		flg, ok := fs.flags[key]
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

// InitializeFlags applies stored flag values over the compiled-in defaults.
// A value that does not decode, or a key no longer registered in this build, is
// logged and skipped: neither aborts startup, and unknown keys are left in the
// store so a rollback still sees them.
func InitializeFlags(ctx context.Context, store FlagStore) error {
	return globalFlags.InitializeFlags(ctx, store)
}

// persist writes one flag's encoded value to the store, so an edit touches exactly
// the flag that changed. A write failure is logged, never fatal.
func (fs *flagset) persist(ctx context.Context, name string, data json.RawMessage) {
	fs.mu.RLock()
	store := fs.store
	fs.mu.RUnlock()
	if store == nil {
		slog.DebugContext(ctx, "No flag store attached, not persisting", slog.String("flag", name))
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := store.SetFlag(ctx, name, data); err != nil {
		slog.WarnContext(ctx, "Couldn't persist flag", slog.String("flag", name), slog.Any("err", err))
	}
}
