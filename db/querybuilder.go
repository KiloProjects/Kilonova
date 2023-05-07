package db

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/KiloProjects/kilonova"
)

type filterBuilder struct {
	mu sync.Mutex

	where []string
	args  []any
	pos   int
}

func (q *filterBuilder) Where() string {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.where) == 0 {
		return "1 = 1"
	}

	return strings.Join(q.where, " AND ")
}

func (q *filterBuilder) Args() []any {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.args
}

// AddConstraint inserts a new constraint with the correct positional parameters
// The `wh` string MUST have `%s` for each position to be replaced by a positional parameter
func (q *filterBuilder) AddConstraint(wh string, args ...any) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// If no parameters are added
	if len(args) == 0 {
		q.where = append(q.where, wh)
		return
	}

	positionals := []any{}
	for range args {
		positionals = append(positionals, "$"+strconv.Itoa(q.pos))
		q.pos++
	}
	q.where = append(q.where, fmt.Sprintf(wh, positionals...))
	q.args = append(q.args, args...)
}

func newFilterBuilder() *filterBuilder {
	return &filterBuilder{
		where: []string{},
		args:  []any{},
		pos:   1,
	}
}

func newFilterBuilderFromPos(args []any) *filterBuilder {
	return &filterBuilder{
		where: []string{},
		args:  slices.Clone(args),
		pos:   len(args) + 1,
	}
}

type updateBuilder struct {
	mu sync.Mutex

	toUpd []string
	args  []any
	pos   int
}

func (upd *updateBuilder) ToUpdate() string {
	upd.mu.Lock()
	defer upd.mu.Unlock()
	if len(upd.toUpd) == 0 {
		return ""
	}

	return strings.Join(upd.toUpd, ", ")
}

func (upd *updateBuilder) Args() []any {
	upd.mu.Lock()
	defer upd.mu.Unlock()
	return upd.args
}

func (upd *updateBuilder) CheckUpdates() error {
	if len(upd.toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	return nil
}

// AddUpdate inserts a new field update with the correct positional parameters
// The `wh` string MUST have `%s` for each position to be replaced by a positional parameter
func (upd *updateBuilder) AddUpdate(wh string, args ...any) {
	upd.mu.Lock()
	defer upd.mu.Unlock()

	// If no parameters are added, which is weird but ooook
	if len(args) == 0 {
		upd.toUpd = append(upd.toUpd, wh)
		return
	}

	positionals := []any{}
	for range args {
		positionals = append(positionals, "$"+strconv.Itoa(upd.pos))
		upd.pos++
	}
	upd.toUpd = append(upd.toUpd, fmt.Sprintf(wh, positionals...))
	upd.args = append(upd.args, args...)
}

func newUpdateBuilder() *updateBuilder {
	return &updateBuilder{
		toUpd: []string{},
		args:  []any{},
		pos:   1,
	}
}

func newUpdateBuilderFromPos(args []any) *updateBuilder {
	return &updateBuilder{
		toUpd: []string{},
		args:  slices.Clone(args),
		pos:   len(args) + 1,
	}
}
