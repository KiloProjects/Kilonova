package db

import (
	"context"
	"encoding/json"
)

// GetFlags returns every stored runtime flag, keyed by its dotted internal name.
// Values are the same JSON encoding the legacy flags.json held, so they can be
// fed straight into the flag registry.
func (s *DB) GetFlags(ctx context.Context) (map[string]json.RawMessage, error) {
	rows, err := s.conn.Query(ctx, "SELECT key, value FROM flags")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	flags := make(map[string]json.RawMessage)
	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		flags[key] = json.RawMessage(value)
	}
	return flags, rows.Err()
}

// SetFlag persists a single flag value, leaving every other flag untouched.
func (s *DB) SetFlag(ctx context.Context, key string, value json.RawMessage) error {
	_, err := s.conn.Exec(ctx,
		"INSERT INTO flags (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()",
		key, []byte(value),
	)
	return err
}
