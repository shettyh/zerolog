package slog

import (
	"github.com/rs/zerolog"
	"os"
	"testing"
)

func TestLogger(t *testing.T) {
	//logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	//slogHandler := NewHandler(logger)
	//slog := slog.New(slogHandler)
	//
	////slog := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	//l1 := slog.With("module", "test").WithGroup("NestedGroup").With("first level key", "first level value")
	//l2 := l1.WithGroup("Nested 2 level").With("Ogh level 2", "l2").With("last key", "last value")
	//
	//l1.Info("test info")
	//l2.Info("test info")

	//l2.WithGroup("Nested 3 level").With("Ogh level 3", "l3").Info("test info")

	// Buil zerolog to get this output
	// {"time":"2023-08-14T10:14:58.680364+05:30","level":"INFO","msg":"test info","module":"test","NestedGroup":{"first level key":"first level value","Nested 2 level":{"Ogh level 2":"l2","last key":"last value"}}}

	logger := zerolog.New(os.Stdout)

	l := logger.WithNamespace("test").Str("Hello", "World").Logger()
	l.Info().Msg("test")
}
