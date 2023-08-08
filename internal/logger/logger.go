package logger

import (
	"context"

	goka "github.com/lovoo/goka"

	zerolog "github.com/rs/zerolog"
)

type (
	GoKaZerolog struct {
		log zerolog.Logger
	}
)

var _ goka.Logger = (*GoKaZerolog)(nil)

func NewGoKaZerolog(ctx context.Context) goka.Logger {
	return &GoKaZerolog{
		log: zerolog.Ctx(ctx).With().Logger(),
	}
}

// Print will simply print the params
func (s *GoKaZerolog) Print(v ...interface{}) {
	s.log.Print(v...)
}

// Print will simply print the params
func (s *GoKaZerolog) Println(v ...interface{}) {
	s.log.Print(v...)
}

// Printf will be used for informational messages. These can be thought of
// having an 'Info'-level in a structured logger.
func (s *GoKaZerolog) Printf(f string, v ...interface{}) {
	s.log.Printf(f, v...)
}
