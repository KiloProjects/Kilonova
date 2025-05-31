package kilonova

import (
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"io"
	"log/slog"
	"os"
	"time"
)

func logColors(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	if !isatty.IsTerminal(f.Fd()) {
		return false
	}

	return os.Getenv("TERM") != "dumb"
}

func GetSlogHandler(debug bool, out io.Writer) slog.Handler {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	return tint.NewHandler(out, &tint.Options{
		AddSource: true,
		Level:     level,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if _, ok := attr.Value.Any().(error); attr.Key == "err" || ok {
				return tint.Attr(9, attr)
			}
			return attr
		},
		TimeFormat: time.RFC3339,
		NoColor:    !logColors(out),
	})
}
