package app

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

// NewLogger создаёт zerolog.Logger. Формат и уровень задаются в одном месте,
// чтобы переключение между ConsoleWriter и JSON было изменением одной функции.
func NewLogger(debug bool) zerolog.Logger {
	var output io.Writer = os.Stderr
	if debug {
		output = zerolog.ConsoleWriter{Out: os.Stderr}
	}

	return zerolog.New(output).
		With().
		Timestamp().
		Logger().
		Level(zerolog.InfoLevel)
}