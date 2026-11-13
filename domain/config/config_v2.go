package config

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"
)

// Flag definitions (name, default, description) are compiled in and shared by
// every instance; their values live in a Flagset, one per instance. Reads go
// through Get(ctx) or Of(fs).
type Flag[T any] interface {
	// Get reads the flagset carried by ctx (see WithFlagset).
	Get(ctx context.Context) T
	// Of reads the given flagset.
	Of(fs *Flagset) T
	// Update sets the value in the flagset carried by ctx and persists it. On a
	// persistence error the new value stays active until restart.
	Update(ctx context.Context, newVal T) error
	InternalName() string
	HumanName() string
}

type flag[T any] struct {
	name      string
	humanName string
	def       T
}

// flagDef is the type-erased view of a definition the registry needs.
type flagDef interface {
	decode(raw json.RawMessage) (any, error)
}

// flagRegistry holds the compiled-in flag definitions. Definitions are code,
// not instance state, so one registry serves every Flagset.
type flagRegistry struct {
	mu   sync.RWMutex
	defs map[string]flagDef
}

func newFlagRegistry() *flagRegistry {
	return &flagRegistry{defs: map[string]flagDef{}}
}

var registry = newFlagRegistry()

// Flagset holds one instance's flag values and the store they persist to.
// A flag missing from vals reads as its compiled-in default.
type Flagset struct {
	mu    sync.RWMutex
	vals  map[string]any
	store FlagStore
}

func NewFlagset() *Flagset {
	return &Flagset{vals: map[string]any{}}
}

type flagsetKey struct{}

// WithFlagset returns a context whose flag reads and updates go to fs.
func WithFlagset(ctx context.Context, fs *Flagset) context.Context {
	return context.WithValue(ctx, flagsetKey{}, fs)
}

func flagsetFrom(ctx context.Context) *Flagset {
	fs, _ := ctx.Value(flagsetKey{}).(*Flagset)
	return fs
}

// Get reads the flagset carried by ctx. A ctx without one is a wiring bug: it is
// logged and the compiled-in default is returned, so a page still renders.
func (f *flag[T]) Get(ctx context.Context) T {
	fs := flagsetFrom(ctx)
	if fs == nil {
		slog.ErrorContext(ctx, "Flag read through a context without a flagset, using default", slog.String("flag", f.name))
		return f.def
	}
	return f.Of(fs)
}

func (f *flag[T]) InternalName() string { return f.name }
func (f *flag[T]) HumanName() string    { return f.humanName }

func (f *flag[T]) Of(fs *Flagset) T {
	fs.mu.RLock()
	v, ok := fs.vals[f.name]
	fs.mu.RUnlock()
	if !ok {
		return f.def
	}
	return v.(T)
}

// Update sets the value and writes it through to the flagset's store, if one is attached.
// The store write happens outside the lock, so readers never wait on the database.
// A failed write is returned, but the new value stays active until restart.
// ponytail: two concurrent edits of the same flag may persist out of order; fine for admin edits.
func (f *flag[T]) Update(ctx context.Context, newVal T) error {
	data, err := json.Marshal(newVal)
	if err != nil {
		return fmt.Errorf("couldn't encode flag %q: %w", f.name, err)
	}
	fs := flagsetFrom(ctx)
	if fs == nil {
		return fmt.Errorf("couldn't update flag %q: no flagset in context", f.name)
	}
	fs.mu.Lock()
	fs.vals[f.name] = newVal
	store := fs.store
	fs.mu.Unlock()

	if store == nil {
		slog.DebugContext(ctx, "No flag store attached, not persisting", slog.String("flag", f.name))
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := store.SetFlag(ctx, f.name, data); err != nil {
		return fmt.Errorf("couldn't persist flag %q: %w", f.name, err)
	}
	return nil
}

func (f *flag[T]) decode(raw json.RawMessage) (any, error) {
	var v T
	err := json.Unmarshal(raw, &v)
	return v, err
}

func (r *flagRegistry) GenFlag[T any](name string, defaultVal T, readableName string) Flag[T] {
	f := &flag[T]{name: name, humanName: readableName, def: defaultVal}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defs[name] = f
	return f
}

func (r *flagRegistry) GetFlag[T any](name string) (Flag[T], bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.defs[name].(*flag[T])
	return v, ok
}

func (r *flagRegistry) GetFlags[T any]() []Flag[T] {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var flags []Flag[T]
	for _, def := range r.defs {
		if f, ok := def.(*flag[T]); ok {
			flags = append(flags, f)
		}
	}
	slices.SortFunc(flags, func(a, b Flag[T]) int {
		return cmp.Compare(a.InternalName(), b.InternalName())
	})
	return flags
}

// decode parses stored values into their flags' types. Unknown keys and values
// that don't decode are logged and skipped, never fatal.
func (r *flagRegistry) decode(ctx context.Context, stored map[string]json.RawMessage) map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	vals := make(map[string]any, len(stored))
	for key, raw := range stored {
		if strings.HasPrefix(key, "_") {
			continue
		}
		def, ok := r.defs[key]
		if !ok {
			slog.WarnContext(ctx, "Unknown flag in store, ignoring", slog.String("key", key))
			continue
		}
		v, err := def.decode(raw)
		if err != nil {
			slog.WarnContext(ctx, "Couldn't apply stored flag, keeping default", slog.String("key", key), slog.Any("err", err))
			continue
		}
		vals[key] = v
	}
	return vals
}

// GenFlag registers a flag definition. Call it from package-level vars only.
func GenFlag[T any](name string, defaultVal T, readableName string) Flag[T] {
	return registry.GenFlag(name, defaultVal, readableName)
}

func GetFlag[T any](name string) (Flag[T], bool) {
	return registry.GetFlag[T](name)
}

func GetFlags[T any]() []Flag[T] {
	return registry.GetFlags[T]()
}

// FlagStore persists runtime flags. It is implemented by the database layer;
// declared here as an interface because db imports this package.
type FlagStore interface {
	GetFlags(ctx context.Context) (map[string]json.RawMessage, error)
	SetFlag(ctx context.Context, key string, value json.RawMessage) error
}

// InitializeFlags applies stored flag values over the compiled-in defaults and
// attaches the store, so later updates persist.
// A value that does not decode, or a key no longer registered in this build, is
// logged and skipped: neither aborts startup, and unknown keys are left in the
// store so a rollback still sees them.
func (fs *Flagset) InitializeFlags(ctx context.Context, store FlagStore) error {
	stored, err := store.GetFlags(ctx)
	if err != nil {
		return err
	}

	vals := registry.decode(ctx, stored)

	fs.mu.Lock()
	defer fs.mu.Unlock()
	maps.Copy(fs.vals, vals)
	fs.store = store
	return nil
}
