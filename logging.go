package kilonova

import (
	"golang.org/x/term"
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func logColors(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	if !term.IsTerminal(int(f.Fd())) {
		return false
	}

	return os.Getenv("TERM") != "dumb"
}

func GetZapCore(debug bool, out io.Writer) zapcore.Core {
	var cfg zapcore.EncoderConfig
	if debug {
		cfg = zap.NewDevelopmentEncoderConfig()
	} else {
		cfg = zap.NewProductionEncoderConfig()
	}
	cfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format(time.RFC3339))
	}
	if logColors(out) {
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}
	return zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		zapcore.AddSync(out),
		level,
	)
}
