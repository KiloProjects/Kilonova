package scratch

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/fsevict"
	"github.com/google/uuid"
	"github.com/spf13/afero"
)

var _ eval.Scratch = (*scratch)(nil)

func New(fs afero.Fs) eval.Scratch {
	return &scratch{fs}
}

type scratch struct {
	fs afero.Fs
}

// Sweep deletes scratch files older than maxTTL. It is a pure orphan janitor:
// identifiers are UUIDv7 and live files are deleted explicitly by their owner,
// so a live file is always younger than maxTTL (which must be >> max eval
// duration). No size cap — scratch is bounded by the TTL, not by disk pressure.
func Sweep(fs afero.Fs, maxTTL time.Duration) (fsevict.Result, error) {
	return fsevict.Sweep(fs, ".", 0, maxTTL)
}

// PeriodicSweep runs Sweep on fs every interval until ctx is done. Grader-side.
func PeriodicSweep(ctx context.Context, fs afero.Fs, interval, maxTTL time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			res, err := Sweep(fs, maxTTL)
			if err != nil {
				logger.WarnContext(ctx, "Scratch GC sweep failed", slog.Any("err", err))
				continue
			}
			if res.Deleted > 0 {
				logger.InfoContext(ctx, "Scratch GC reclaimed orphans", slog.Int("deleted", res.Deleted), slog.Int64("bytes", res.DeletedSize))
			}
		}
	}
}

func (s *scratch) genIdentifier() string {
	return uuid.Must(uuid.NewV7()).String()
}

func (s *scratch) SaveFile(r io.Reader) (string, error) {
	identifier := s.genIdentifier()
	return identifier, afero.WriteReader(s.fs, identifier, r)
}

func (s *scratch) ReadFile(identifier string) (io.ReadCloser, error) {
	f, err := s.fs.Open(identifier)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (s *scratch) DeleteFile(identifier string) error {
	return s.fs.RemoveAll(identifier)
}
