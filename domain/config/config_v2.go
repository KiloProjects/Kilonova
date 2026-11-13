package config

import (
	"cmp"
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Flag definitions (name, default, description) are compiled in and shared by
// every instance; their values live in a Flagset, one per instance. Reads go
// through Get(ctx) or Of(fs); Value() is the transitional read of the global set.
type Flag[T any] interface {
	// Value reads the process-wide flagset. Transitional: migrate callers to Get or Of.
	Value() T
	// Get reads the flagset carried by ctx (see WithFlagset).
	Get(ctx context.Context) T
	// Of reads the given flagset.
	Of(fs *Flagset) T
	// Update sets the value in the flagset carried by ctx and persists it.
	Update(ctx context.Context, newVal T)
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

// globalFlags backs Value() and any ctx without a flagset until every caller is migrated.
var globalFlags = NewFlagset()

var flagsetFallbacks = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "kn_flagset_fallback_total",
	Help: "Flag reads/updates through a context without a flagset, served from the global flagset",
}, []string{"flag"})

type flagsetKey struct{}

// WithFlagset returns a context whose flag reads and updates go to fs.
func WithFlagset(ctx context.Context, fs *Flagset) context.Context {
	return context.WithValue(ctx, flagsetKey{}, fs)
}

func flagsetFrom(ctx context.Context, name string) *Flagset {
	if fs, ok := ctx.Value(flagsetKey{}).(*Flagset); ok && fs != nil {
		return fs
	}
	flagsetFallbacks.WithLabelValues(name).Inc()
	return globalFlags
}

func (f *flag[T]) Value() T                  { return f.Of(globalFlags) }
func (f *flag[T]) Get(ctx context.Context) T { return f.Of(flagsetFrom(ctx, f.name)) }
func (f *flag[T]) InternalName() string      { return f.name }
func (f *flag[T]) HumanName() string         { return f.humanName }

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
// ponytail: two concurrent edits of the same flag may persist out of order; fine for admin edits.
func (f *flag[T]) Update(ctx context.Context, newVal T) {
	data, err := json.Marshal(newVal)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't encode flag", slog.String("flag", f.name), slog.Any("err", err))
		return
	}
	fs := flagsetFrom(ctx, f.name)
	fs.mu.Lock()
	fs.vals[f.name] = newVal
	store := fs.store
	fs.mu.Unlock()

	if store == nil {
		slog.DebugContext(ctx, "No flag store attached, not persisting", slog.String("flag", f.name))
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := store.SetFlag(ctx, f.name, data); err != nil {
		slog.WarnContext(ctx, "Couldn't persist flag", slog.String("flag", f.name), slog.Any("err", err))
	}
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

// GlobalFlagset returns the process-wide flagset. Transitional: the platform must
// hand this exact set to BaseAPI while any caller still reads through Value(),
// or admin edits and Value() reads would see different values.
func GlobalFlagset() *Flagset {
	return globalFlags
}
